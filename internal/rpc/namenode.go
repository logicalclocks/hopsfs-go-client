package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	krb "github.com/jcmturner/gokrb5/v8/client"
	"google.golang.org/protobuf/proto"
)

const (
	rpcVersion            byte = 0x09
	serviceClass          byte = 0x0
	noneAuthProtocol      byte = 0x0
	saslAuthProtocol      byte = 0xdf
	protocolClass              = "org.apache.hadoop.hdfs.protocol.ClientProtocol"
	protocolClassVersion       = 1
	handshakeCallID            = -3
	standbyExceptionClass      = "org.apache.hadoop.ipc.StandbyException"
)

const (
	backoffDuration    = 5 * time.Second
	leaseRenewInterval = 1 * time.Second

	DefaultDialTimeout    = 30 * time.Second
	DefaultTCPUserTimeout = 30 * time.Second
	// The TCP keep-alive probe interval used for idle namenode connections.
	keepAliveInterval = 15 * time.Second
)

// NamenodeConnection represents an open connection to a namenode.
type NamenodeConnection struct {
	ClientID   []byte
	ClientName string
	User       string

	currentRequestID int32

	kerberosClient               *krb.Client
	kerberosServicePrincipleName string
	kerberosRealm                string

	// Use SSL
	TLS bool
	// if TLS is set then also set the following parameters
	RootCABundle      string
	ClientCertificate string
	ClientKey         string

	dialFunc       func(ctx context.Context, network, addr string) (net.Conn, error)
	dialTimeout    time.Duration
	tcpUserTimeout time.Duration
	conn           net.Conn
	host           *namenodeHost
	hostList       []*namenodeHost
	transport      transport

	reqLock sync.Mutex
	done    chan struct{}
}

// NamenodeConnectionOptions represents the configurable options available
// for a NamenodeConnection.
type NamenodeConnectionOptions struct {
	// Addresses specifies the namenode(s) to connect to.
	Addresses []string
	// User specifies which HDFS user the client will act as. It is required
	// unless kerberos authentication is enabled, in which case it is overridden
	// by the username set in KerberosClient.
	User string
	// DialFunc is used to connect to the namenodes. If nil, then
	// (&net.Dialer{}).DialContext is used.
	DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)
	// KerberosClient is used to connect to kerberized HDFS clusters. If provided,
	// the NamenodeConnection will always mutually athenticate when connecting
	// to the namenode(s).
	KerberosClient *krb.Client
	// KerberosServicePrincipleName specifiesthe Service Principle Name
	// (<SERVICE>/<FQDN>) for the namenode(s). Like in the
	// dfs.namenode.kerberos.principal property of core-site.xml, the special
	// string '_HOST' can be substituted for the hostname in a multi-namenode
	// setup (for example: 'nn/_HOST@EXAMPLE.COM'). It is required if
	// KerberosClient is provided.
	KerberosServicePrincipleName string

	// Use SSL
	TLS               bool // if TLS is set then also set the following parameters
	RootCABundle      string
	ClientCertificate string
	ClientKey         string

	// Bounds how long establishing a TCP (and TLS) connection to a namenode may take.
	DialTimeout time.Duration
	// Bounds how long transmitted data may remain unacknowledged before aborting the connection.
	// Only applied on platforms that support TCP_USER_TIMEOUT.
	TCPUserTimeout time.Duration
}

func effectiveTimeout(configured, def time.Duration) time.Duration {
	if configured == 0 {
		return def
	}
	if configured < 0 {
		return 0
	}
	return configured
}

type namenodeHost struct {
	address     string
	lastError   error
	lastErrorAt time.Time
}

// NewNamenodeConnectionWithOptions creates a new connection to a namenode with
// the given options and performs an initial handshake.
func NewNamenodeConnection(options NamenodeConnectionOptions) (*NamenodeConnection, error) {
	// Build the list of hosts to be used for failover.
	hostList := make([]*namenodeHost, len(options.Addresses))
	for i, addr := range options.Addresses {
		hostList[i] = &namenodeHost{address: addr}
	}

	var user, realm string
	user = options.User
	if options.KerberosClient != nil {
		creds := options.KerberosClient.Credentials
		user = creds.UserName()
		realm = creds.Realm()
	} else if user == "" {
		return nil, errors.New("user not specified")
	}

	// The ClientID is reused here both in the RPC headers (which requires a
	// "globally unique" ID) and as the "client name" in various requests.
	clientId := newClientID()
	c := &NamenodeConnection{
		ClientID:   clientId,
		ClientName: "GO-HopsFS-" + string(clientId),
		User:       user,

		kerberosClient:               options.KerberosClient,
		kerberosServicePrincipleName: options.KerberosServicePrincipleName,
		kerberosRealm:                realm,

		TLS:               options.TLS,
		RootCABundle:      options.RootCABundle,
		ClientCertificate: options.ClientCertificate,
		ClientKey:         options.ClientKey,

		dialFunc:       options.DialFunc,
		dialTimeout:    effectiveTimeout(options.DialTimeout, DefaultDialTimeout),
		tcpUserTimeout: effectiveTimeout(options.TCPUserTimeout, DefaultTCPUserTimeout),
		hostList:       hostList,
		transport:      &basicTransport{clientID: clientId},

		done: make(chan struct{}),
	}

	if options.TLS {
		c.dialFunc = c.tlsDialFunction
	} else if c.dialFunc == nil {
		c.dialFunc = c.newDialer().DialContext
	}

	err := c.resolveConnection()
	if err != nil {
		return nil, err
	}

	// Periodically renew any file leases.
	go c.renewLeases()

	return c, nil
}

func (c *NamenodeConnection) resolveConnection() error {
	if c.conn != nil {
		return nil
	}

	var err error
	if c.host != nil {
		err = c.host.lastError
	}

	for _, host := range c.hostList {
		if time.Since(host.lastErrorAt) < backoffDuration {
			continue
		}

		c.host = host
		deadline := c.connectDeadline()
		c.conn, err = c.dial(host.address, deadline)
		if err != nil {
			c.markFailure(err)
			continue
		}

		err = c.handshake(deadline)
		if err != nil {
			c.markFailure(err)
			continue
		}

		break
	}

	if c.conn == nil {
		return fmt.Errorf("no available namenodes: %s", err)
	}

	return nil
}

// newDialer returns the dialer used for namenode connections when no custom
// DialFunc is supplied.
func (c *NamenodeConnection) newDialer() *net.Dialer {
	d := &net.Dialer{
		Timeout:   c.dialTimeout,
		KeepAlive: keepAliveInterval,
	}
	if c.tcpUserTimeout > 0 {
		// nil on platforms without TCP_USER_TIMEOUT.
		d.Control = tcpUserTimeoutControl(c.tcpUserTimeout)
	}
	return d
}

func (c *NamenodeConnection) connectDeadline() time.Time {
	if c.dialTimeout <= 0 {
		return time.Time{}
	}
	return time.Now().Add(c.dialTimeout)
}

// dial connects to address, giving up at deadline unless it is zero.
func (c *NamenodeConnection) dial(address string, deadline time.Time) (net.Conn, error) {
	ctx := context.Background()
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	return c.dialFunc(ctx, "tcp", address)
}

func (c *NamenodeConnection) handshake(deadline time.Time) error {
	if deadline.IsZero() {
		return c.doNamenodeHandshake()
	}
	if err := c.conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("setting handshake deadline: %w", err)
	}
	err := c.doNamenodeHandshake()
	if clearErr := c.conn.SetDeadline(time.Time{}); clearErr != nil && err == nil {
		err = fmt.Errorf("clearing handshake deadline: %w", clearErr)
	}
	return err
}

func (c *NamenodeConnection) markFailure(err error) {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.host.lastError = err
	c.host.lastErrorAt = time.Now()
}

// Execute performs an rpc call. It does this by sending req over the wire and
// unmarshaling the result into resp.
func (c *NamenodeConnection) Execute(method string, req proto.Message, resp proto.Message) error {
	c.reqLock.Lock()
	defer c.reqLock.Unlock()

	c.currentRequestID++
	requestID := c.currentRequestID

	for {
		err := c.resolveConnection()
		if err != nil {
			return err
		}

		err = c.transport.writeRequest(c.conn, method, requestID, req)
		if err != nil {
			c.markFailure(err)
			continue
		}

		err = c.transport.readResponse(c.conn, method, requestID, resp)
		if err != nil {
			if nerr, ok := err.(*NamenodeError); ok {
				// The namenode answered, so the connection itself is healthy.
				// Only retry on a standby exception.
				if nerr.exception == standbyExceptionClass {
					c.markFailure(err)
					continue
				}
				return err
			}

			// Anything else (EOF, reset, user timeout, framing error) means
			// the connection can no longer be trusted.
			c.markFailure(err)
			return err
		}

		break
	}

	return nil
}

// A handshake packet:
// +-----------------------------------------------------------+
// |  Header, 4 bytes ("hrpc")                                 |
// +-----------------------------------------------------------+
// |  Version, 1 byte (default verion 0x09)                    |
// +-----------------------------------------------------------+
// |  RPC service class, 1 byte (0x00)                         |
// +-----------------------------------------------------------+
// |  Auth protocol, 1 byte (Auth method None = 0x00)          |
// +-----------------------------------------------------------+
//
//	If the auth protocol is something other than 'none', the authentication
//	handshake happens here. Otherwise, everything can be sent as one packet.
//
// +-----------------------------------------------------------+
// |  uint32 length of the next two parts                      |
// +-----------------------------------------------------------+
// |  varint length + RpcRequestHeaderProto                    |
// +-----------------------------------------------------------+
// |  varint length + IpcConnectionContextProto                |
// +-----------------------------------------------------------+
func (c *NamenodeConnection) doNamenodeHandshake() error {
	authProtocol := noneAuthProtocol
	kerberos := false
	if c.kerberosClient != nil {
		authProtocol = saslAuthProtocol
		kerberos = true
	}

	rpcHeader := []byte{
		0x68, 0x72, 0x70, 0x63, // "hrpc"
		rpcVersion, serviceClass, authProtocol,
	}

	_, err := c.conn.Write(rpcHeader)
	if err != nil {
		return err
	}

	if kerberos {
		err = c.doKerberosHandshake()
		if err != nil {
			return fmt.Errorf("SASL handshake: %s", err)
		}
	}

	rrh := newRPCRequestHeader(handshakeCallID, c.ClientID)
	cc := newConnectionContext(c.User, c.kerberosRealm)
	packet, err := makeRPCPacket(rrh, cc)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(packet)
	return err
}

// renewLeases periodically renews all leases for the connection.
func (c *NamenodeConnection) renewLeases() {
	ticker := time.NewTicker(leaseRenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			req := &hdfs.RenewLeaseRequestProto{ClientName: proto.String(c.ClientName)}
			resp := &hdfs.RenewLeaseResponseProto{}

			// Ignore any errors.
			c.Execute("renewLease", req, resp)
		case <-c.done:
			return
		}
	}
}

// Close terminates all underlying socket connections to remote server.
func (c *NamenodeConnection) Close() error {
	close(c.done)

	// Ensure that we're not concurrently renewing leases.
	c.reqLock.Lock()
	defer c.reqLock.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func (c *NamenodeConnection) tlsDialFunction(ctx context.Context, network, address string) (net.Conn, error) {
	// Load client's certificate(including the intermediate) and private key
	clientCert, err := tls.LoadX509KeyPair(c.ClientCertificate, c.ClientKey)
	if err != nil {
		return nil, err
	}

	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile(c.RootCABundle)
	if err != nil {
		return nil, err
	}

	certChain := decodePem(pemServerCA)

	config := &tls.Config{}

	config.RootCAs = x509.NewCertPool()
	for _, cert := range certChain.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			panic(err)
		}
		config.RootCAs.AddCert(x509Cert)
	}

	config.Certificates = []tls.Certificate{clientCert}
	config.InsecureSkipVerify = true

	config.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		// If this is the first handshake on a connection, process and
		// (optionally) verify the server's certificates.
		certs := make([]*x509.Certificate, len(rawCerts))

		for i, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				panic("Failed to parse certificate from server: " + err.Error())
			}
			certs[i] = cert
		}

		opts := x509.VerifyOptions{
			Roots:         config.RootCAs,
			CurrentTime:   time.Now(),
			DNSName:       "", // <- skip hostname verification
			Intermediates: x509.NewCertPool(),
		}

		for i, cert := range certs {
			if i == 0 {
				continue
			}
			opts.Intermediates.AddCert(cert)
		}
		_, err := certs[0].Verify(opts)
		return err
	}

	dialer := &tls.Dialer{NetDialer: c.newDialer(), Config: config}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return conn, nil
}

func decodePem(certInput []byte) tls.Certificate {
	var cert tls.Certificate
	certPEMBlock := certInput
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}
	return cert
}

func newRPCRequestHeader(id int32, clientID []byte) *hadoop.RpcRequestHeaderProto {
	epoch := getRpcEpochSec()
	return &hadoop.RpcRequestHeaderProto{
		RpcKind:  hadoop.RpcKindProto_RPC_PROTOCOL_BUFFER.Enum(),
		RpcOp:    hadoop.RpcRequestHeaderProto_RPC_FINAL_PACKET.Enum(),
		CallId:   proto.Int32(id),
		ClientId: clientID,
		Epoch:    &epoch,
	}
}

func newRequestHeader(methodName string) *hadoop.RequestHeaderProto {
	return &hadoop.RequestHeaderProto{
		MethodName:                 proto.String(methodName),
		DeclaringClassProtocolName: proto.String(protocolClass),
		ClientProtocolVersion:      proto.Uint64(uint64(protocolClassVersion)),
	}
}

func newConnectionContext(user, kerberosRealm string) *hadoop.IpcConnectionContextProto {
	if kerberosRealm != "" {
		user = user + "@" + kerberosRealm
	}

	return &hadoop.IpcConnectionContextProto{
		UserInfo: &hadoop.UserInformationProto{
			EffectiveUser: proto.String(user),
		},
		Protocol: proto.String(protocolClass),
	}
}

var serverReportedEpoch int64 = 0
var epochReportTime time.Time = time.Now()

func SetEpoch(reportTime time.Time, epoch int64) {
	serverReportedEpoch = epoch
	epochReportTime = reportTime
}

func getRpcEpochSec() int64 {
	if serverReportedEpoch == 0 {
		return 0
	} else {
		timePassed := time.Since(epochReportTime).Milliseconds()
		currentTime := serverReportedEpoch + timePassed
		epoch := currentTime / 1000
		return epoch
	}
}
