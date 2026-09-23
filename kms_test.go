package hdfs

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKmsParseProviderUri(t *testing.T) {
	assert.Equal(t, nil, nil)

	urls, err := kmsParseProviderUri("")
	assert.Error(t, err)

	urls, err = kmsParseProviderUri("http")
	assert.Error(t, err)

	urls, err = kmsParseProviderUri("kms://https@localhost:9600/kms")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, "https://localhost:9600/kms", urls[0])

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com:9600;kms02.example.com")
	assert.Error(t, err)

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com/kms;kms02.example.com")
	assert.Error(t, err)

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com;kms02.example.com:9600/kms")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(urls))
	assert.Equal(t, "http://kms01.example.com:9600/kms", urls[0])
	assert.Equal(t, "http://kms02.example.com:9600/kms", urls[1])

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com;kms02.example.com/kms")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(urls))
	assert.Equal(t, "http://kms01.example.com:9600/kms", urls[0])
	assert.Equal(t, "http://kms02.example.com:9600/kms", urls[1])

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com;kms02.example.com:9600")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(urls))
	assert.Equal(t, "http://kms01.example.com:9600", urls[0])
	assert.Equal(t, "http://kms02.example.com:9600", urls[1])

	urls, err = kmsParseProviderUri("kms://http@kms01.example.com;kms02.example.com;kms03.example.com")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(urls))
	assert.Equal(t, "http://kms01.example.com:9600", urls[0])
	assert.Equal(t, "http://kms02.example.com:9600", urls[1])
	assert.Equal(t, "http://kms03.example.com:9600", urls[2])
}

func TestNewKMSHTTPClientIgnoresProxyEnvironment(t *testing.T) {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := newKMSHTTPClient(jar)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "expected *http.Transport, got %T", client.Transport)
	// Asserted on the transport rather than through a live proxy: ProxyFromEnvironment
	// reads the environment once per process, so a Setenv here could be ignored.
	assert.Nil(t, transport.Proxy)
	assert.Same(t, jar, client.Jar)
}

func TestNewKMSHTTPClientKeepsCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("hadoop.auth"); err != nil {
			http.SetCookie(w, &http.Cookie{Name: "hadoop.auth", Value: "token"})
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := newKMSHTTPClient(jar)

	first, err := client.Post(server.URL, "application/json", nil)
	require.NoError(t, err)
	first.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, first.StatusCode)

	second, err := client.Post(server.URL, "application/json", nil)
	require.NoError(t, err)
	second.Body.Close()
	assert.Equal(t, http.StatusOK, second.StatusCode)
}
