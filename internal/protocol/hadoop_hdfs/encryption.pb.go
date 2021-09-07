// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        (unknown)
// source: encryption.proto

package hadoop_hdfs

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type CreateEncryptionZoneRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Src     *string `protobuf:"bytes,1,req,name=src" json:"src,omitempty"`
	KeyName *string `protobuf:"bytes,2,opt,name=keyName" json:"keyName,omitempty"`
}

func (x *CreateEncryptionZoneRequestProto) Reset() {
	*x = CreateEncryptionZoneRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateEncryptionZoneRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateEncryptionZoneRequestProto) ProtoMessage() {}

func (x *CreateEncryptionZoneRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateEncryptionZoneRequestProto.ProtoReflect.Descriptor instead.
func (*CreateEncryptionZoneRequestProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{0}
}

func (x *CreateEncryptionZoneRequestProto) GetSrc() string {
	if x != nil && x.Src != nil {
		return *x.Src
	}
	return ""
}

func (x *CreateEncryptionZoneRequestProto) GetKeyName() string {
	if x != nil && x.KeyName != nil {
		return *x.KeyName
	}
	return ""
}

type CreateEncryptionZoneResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *CreateEncryptionZoneResponseProto) Reset() {
	*x = CreateEncryptionZoneResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateEncryptionZoneResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateEncryptionZoneResponseProto) ProtoMessage() {}

func (x *CreateEncryptionZoneResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateEncryptionZoneResponseProto.ProtoReflect.Descriptor instead.
func (*CreateEncryptionZoneResponseProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{1}
}

type ListEncryptionZonesRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id *int64 `protobuf:"varint,1,req,name=id" json:"id,omitempty"`
}

func (x *ListEncryptionZonesRequestProto) Reset() {
	*x = ListEncryptionZonesRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListEncryptionZonesRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListEncryptionZonesRequestProto) ProtoMessage() {}

func (x *ListEncryptionZonesRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListEncryptionZonesRequestProto.ProtoReflect.Descriptor instead.
func (*ListEncryptionZonesRequestProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{2}
}

func (x *ListEncryptionZonesRequestProto) GetId() int64 {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return 0
}

type EncryptionZoneProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                    *int64                      `protobuf:"varint,1,req,name=id" json:"id,omitempty"`
	Path                  *string                     `protobuf:"bytes,2,req,name=path" json:"path,omitempty"`
	Suite                 *CipherSuiteProto           `protobuf:"varint,3,req,name=suite,enum=hadoop.hdfs.CipherSuiteProto" json:"suite,omitempty"`
	CryptoProtocolVersion *CryptoProtocolVersionProto `protobuf:"varint,4,req,name=cryptoProtocolVersion,enum=hadoop.hdfs.CryptoProtocolVersionProto" json:"cryptoProtocolVersion,omitempty"`
	KeyName               *string                     `protobuf:"bytes,5,req,name=keyName" json:"keyName,omitempty"`
}

func (x *EncryptionZoneProto) Reset() {
	*x = EncryptionZoneProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EncryptionZoneProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptionZoneProto) ProtoMessage() {}

func (x *EncryptionZoneProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptionZoneProto.ProtoReflect.Descriptor instead.
func (*EncryptionZoneProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{3}
}

func (x *EncryptionZoneProto) GetId() int64 {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return 0
}

func (x *EncryptionZoneProto) GetPath() string {
	if x != nil && x.Path != nil {
		return *x.Path
	}
	return ""
}

func (x *EncryptionZoneProto) GetSuite() CipherSuiteProto {
	if x != nil && x.Suite != nil {
		return *x.Suite
	}
	return CipherSuiteProto_UNKNOWN
}

func (x *EncryptionZoneProto) GetCryptoProtocolVersion() CryptoProtocolVersionProto {
	if x != nil && x.CryptoProtocolVersion != nil {
		return *x.CryptoProtocolVersion
	}
	return CryptoProtocolVersionProto_UNKNOWN_PROTOCOL_VERSION
}

func (x *EncryptionZoneProto) GetKeyName() string {
	if x != nil && x.KeyName != nil {
		return *x.KeyName
	}
	return ""
}

type ListEncryptionZonesResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Zones   []*EncryptionZoneProto `protobuf:"bytes,1,rep,name=zones" json:"zones,omitempty"`
	HasMore *bool                  `protobuf:"varint,2,req,name=hasMore" json:"hasMore,omitempty"`
}

func (x *ListEncryptionZonesResponseProto) Reset() {
	*x = ListEncryptionZonesResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListEncryptionZonesResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListEncryptionZonesResponseProto) ProtoMessage() {}

func (x *ListEncryptionZonesResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListEncryptionZonesResponseProto.ProtoReflect.Descriptor instead.
func (*ListEncryptionZonesResponseProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{4}
}

func (x *ListEncryptionZonesResponseProto) GetZones() []*EncryptionZoneProto {
	if x != nil {
		return x.Zones
	}
	return nil
}

func (x *ListEncryptionZonesResponseProto) GetHasMore() bool {
	if x != nil && x.HasMore != nil {
		return *x.HasMore
	}
	return false
}

type GetEZForPathRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Src *string `protobuf:"bytes,1,req,name=src" json:"src,omitempty"`
}

func (x *GetEZForPathRequestProto) Reset() {
	*x = GetEZForPathRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetEZForPathRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEZForPathRequestProto) ProtoMessage() {}

func (x *GetEZForPathRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEZForPathRequestProto.ProtoReflect.Descriptor instead.
func (*GetEZForPathRequestProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{5}
}

func (x *GetEZForPathRequestProto) GetSrc() string {
	if x != nil && x.Src != nil {
		return *x.Src
	}
	return ""
}

type GetEZForPathResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Zone *EncryptionZoneProto `protobuf:"bytes,1,opt,name=zone" json:"zone,omitempty"`
}

func (x *GetEZForPathResponseProto) Reset() {
	*x = GetEZForPathResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encryption_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetEZForPathResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEZForPathResponseProto) ProtoMessage() {}

func (x *GetEZForPathResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_encryption_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEZForPathResponseProto.ProtoReflect.Descriptor instead.
func (*GetEZForPathResponseProto) Descriptor() ([]byte, []int) {
	return file_encryption_proto_rawDescGZIP(), []int{6}
}

func (x *GetEZForPathResponseProto) GetZone() *EncryptionZoneProto {
	if x != nil {
		return x.Zone
	}
	return nil
}

var File_encryption_proto protoreflect.FileDescriptor

var file_encryption_proto_rawDesc = []byte{
	0x0a, 0x10, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0b, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x1a,
	0x0a, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x40, 0x0a, 0x20, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5a,
	0x6f, 0x6e, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0b, 0x0a, 0x03, 0x73, 0x72, 0x63, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x12, 0x0f, 0x0a, 0x07,
	0x6b, 0x65, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x22, 0x23, 0x0a,
	0x21, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x5a, 0x6f, 0x6e, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x2d, 0x0a, 0x1f, 0x4c, 0x69, 0x73, 0x74, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x6f, 0x6e, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x02, 0x28,
	0x03, 0x22, 0xb6, 0x01, 0x0a, 0x13, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x5a, 0x6f, 0x6e, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x02, 0x28, 0x03, 0x12, 0x0c, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20,
	0x02, 0x28, 0x09, 0x12, 0x2c, 0x0a, 0x05, 0x73, 0x75, 0x69, 0x74, 0x65, 0x18, 0x03, 0x20, 0x02,
	0x28, 0x0e, 0x32, 0x1d, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73,
	0x2e, 0x43, 0x69, 0x70, 0x68, 0x65, 0x72, 0x53, 0x75, 0x69, 0x74, 0x65, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x46, 0x0a, 0x15, 0x63, 0x72, 0x79, 0x70, 0x74, 0x6f, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x02, 0x28, 0x0e,
	0x32, 0x27, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x43,
	0x72, 0x79, 0x70, 0x74, 0x6f, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x56, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x0a, 0x07, 0x6b, 0x65, 0x79,
	0x4e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x02, 0x28, 0x09, 0x22, 0x64, 0x0a, 0x20, 0x4c, 0x69,
	0x73, 0x74, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x6f, 0x6e, 0x65,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x2f,
	0x0a, 0x05, 0x7a, 0x6f, 0x6e, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e,
	0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x45, 0x6e, 0x63, 0x72,
	0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x6f, 0x6e, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0f, 0x0a, 0x07, 0x68, 0x61, 0x73, 0x4d, 0x6f, 0x72, 0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x08,
	0x22, 0x27, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x45, 0x5a, 0x46, 0x6f, 0x72, 0x50, 0x61, 0x74, 0x68,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x0a, 0x03,
	0x73, 0x72, 0x63, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x22, 0x4b, 0x0a, 0x19, 0x47, 0x65, 0x74,
	0x45, 0x5a, 0x46, 0x6f, 0x72, 0x50, 0x61, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x2e, 0x0a, 0x04, 0x7a, 0x6f, 0x6e, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64,
	0x66, 0x73, 0x2e, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x6f, 0x6e,
	0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x42, 0x41, 0x0a, 0x25, 0x6f, 0x72, 0x67, 0x2e, 0x61, 0x70,
	0x61, 0x63, 0x68, 0x65, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x42,
	0x15, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x6f, 0x6e, 0x65, 0x73,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0xa0, 0x01, 0x01,
}

var (
	file_encryption_proto_rawDescOnce sync.Once
	file_encryption_proto_rawDescData = file_encryption_proto_rawDesc
)

func file_encryption_proto_rawDescGZIP() []byte {
	file_encryption_proto_rawDescOnce.Do(func() {
		file_encryption_proto_rawDescData = protoimpl.X.CompressGZIP(file_encryption_proto_rawDescData)
	})
	return file_encryption_proto_rawDescData
}

var file_encryption_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_encryption_proto_goTypes = []interface{}{
	(*CreateEncryptionZoneRequestProto)(nil),  // 0: hadoop.hdfs.CreateEncryptionZoneRequestProto
	(*CreateEncryptionZoneResponseProto)(nil), // 1: hadoop.hdfs.CreateEncryptionZoneResponseProto
	(*ListEncryptionZonesRequestProto)(nil),   // 2: hadoop.hdfs.ListEncryptionZonesRequestProto
	(*EncryptionZoneProto)(nil),               // 3: hadoop.hdfs.EncryptionZoneProto
	(*ListEncryptionZonesResponseProto)(nil),  // 4: hadoop.hdfs.ListEncryptionZonesResponseProto
	(*GetEZForPathRequestProto)(nil),          // 5: hadoop.hdfs.GetEZForPathRequestProto
	(*GetEZForPathResponseProto)(nil),         // 6: hadoop.hdfs.GetEZForPathResponseProto
	(CipherSuiteProto)(0),                     // 7: hadoop.hdfs.CipherSuiteProto
	(CryptoProtocolVersionProto)(0),           // 8: hadoop.hdfs.CryptoProtocolVersionProto
}
var file_encryption_proto_depIdxs = []int32{
	7, // 0: hadoop.hdfs.EncryptionZoneProto.suite:type_name -> hadoop.hdfs.CipherSuiteProto
	8, // 1: hadoop.hdfs.EncryptionZoneProto.cryptoProtocolVersion:type_name -> hadoop.hdfs.CryptoProtocolVersionProto
	3, // 2: hadoop.hdfs.ListEncryptionZonesResponseProto.zones:type_name -> hadoop.hdfs.EncryptionZoneProto
	3, // 3: hadoop.hdfs.GetEZForPathResponseProto.zone:type_name -> hadoop.hdfs.EncryptionZoneProto
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_encryption_proto_init() }
func file_encryption_proto_init() {
	if File_encryption_proto != nil {
		return
	}
	file_hdfs_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_encryption_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateEncryptionZoneRequestProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateEncryptionZoneResponseProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListEncryptionZonesRequestProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EncryptionZoneProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListEncryptionZonesResponseProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetEZForPathRequestProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_encryption_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetEZForPathResponseProto); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_encryption_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_encryption_proto_goTypes,
		DependencyIndexes: file_encryption_proto_depIdxs,
		MessageInfos:      file_encryption_proto_msgTypes,
	}.Build()
	File_encryption_proto = out.File
	file_encryption_proto_rawDesc = nil
	file_encryption_proto_goTypes = nil
	file_encryption_proto_depIdxs = nil
}
