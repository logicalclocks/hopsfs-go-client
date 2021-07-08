// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.17.3
// source: GetUserMappingsProtocol.proto

package hadoop_common

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

//*
//  Get groups for user request.
type GetGroupsForUserRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User *string `protobuf:"bytes,1,req,name=user" json:"user,omitempty"`
}

func (x *GetGroupsForUserRequestProto) Reset() {
	*x = GetGroupsForUserRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_GetUserMappingsProtocol_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGroupsForUserRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGroupsForUserRequestProto) ProtoMessage() {}

func (x *GetGroupsForUserRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_GetUserMappingsProtocol_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGroupsForUserRequestProto.ProtoReflect.Descriptor instead.
func (*GetGroupsForUserRequestProto) Descriptor() ([]byte, []int) {
	return file_GetUserMappingsProtocol_proto_rawDescGZIP(), []int{0}
}

func (x *GetGroupsForUserRequestProto) GetUser() string {
	if x != nil && x.User != nil {
		return *x.User
	}
	return ""
}

//*
// Response for get groups.
type GetGroupsForUserResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Groups []string `protobuf:"bytes,1,rep,name=groups" json:"groups,omitempty"`
}

func (x *GetGroupsForUserResponseProto) Reset() {
	*x = GetGroupsForUserResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_GetUserMappingsProtocol_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGroupsForUserResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGroupsForUserResponseProto) ProtoMessage() {}

func (x *GetGroupsForUserResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_GetUserMappingsProtocol_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGroupsForUserResponseProto.ProtoReflect.Descriptor instead.
func (*GetGroupsForUserResponseProto) Descriptor() ([]byte, []int) {
	return file_GetUserMappingsProtocol_proto_rawDescGZIP(), []int{1}
}

func (x *GetGroupsForUserResponseProto) GetGroups() []string {
	if x != nil {
		return x.Groups
	}
	return nil
}

var File_GetUserMappingsProtocol_proto protoreflect.FileDescriptor

var file_GetUserMappingsProtocol_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0d, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x22, 0x32,
	0x0a, 0x1c, 0x47, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12,
	0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x04, 0x75, 0x73,
	0x65, 0x72, 0x22, 0x37, 0x0a, 0x1d, 0x47, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x46,
	0x6f, 0x72, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x16, 0x0a, 0x06, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x06, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x32, 0x8f, 0x01, 0x0a, 0x1e,
	0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6d,
	0x0a, 0x10, 0x67, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73,
	0x65, 0x72, 0x12, 0x2b, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x46, 0x6f, 0x72, 0x55,
	0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x2c, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x47, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73, 0x65, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x42, 0x44, 0x0a,
	0x1d, 0x6f, 0x72, 0x67, 0x2e, 0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x68, 0x61, 0x64, 0x6f,
	0x6f, 0x70, 0x2e, 0x74, 0x6f, 0x6f, 0x6c, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x42, 0x1d,
	0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x88, 0x01, 0x01,
	0xa0, 0x01, 0x01,
}

var (
	file_GetUserMappingsProtocol_proto_rawDescOnce sync.Once
	file_GetUserMappingsProtocol_proto_rawDescData = file_GetUserMappingsProtocol_proto_rawDesc
)

func file_GetUserMappingsProtocol_proto_rawDescGZIP() []byte {
	file_GetUserMappingsProtocol_proto_rawDescOnce.Do(func() {
		file_GetUserMappingsProtocol_proto_rawDescData = protoimpl.X.CompressGZIP(file_GetUserMappingsProtocol_proto_rawDescData)
	})
	return file_GetUserMappingsProtocol_proto_rawDescData
}

var file_GetUserMappingsProtocol_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_GetUserMappingsProtocol_proto_goTypes = []interface{}{
	(*GetGroupsForUserRequestProto)(nil),  // 0: hadoop.common.GetGroupsForUserRequestProto
	(*GetGroupsForUserResponseProto)(nil), // 1: hadoop.common.GetGroupsForUserResponseProto
}
var file_GetUserMappingsProtocol_proto_depIdxs = []int32{
	0, // 0: hadoop.common.GetUserMappingsProtocolService.getGroupsForUser:input_type -> hadoop.common.GetGroupsForUserRequestProto
	1, // 1: hadoop.common.GetUserMappingsProtocolService.getGroupsForUser:output_type -> hadoop.common.GetGroupsForUserResponseProto
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_GetUserMappingsProtocol_proto_init() }
func file_GetUserMappingsProtocol_proto_init() {
	if File_GetUserMappingsProtocol_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_GetUserMappingsProtocol_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGroupsForUserRequestProto); i {
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
		file_GetUserMappingsProtocol_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGroupsForUserResponseProto); i {
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
			RawDescriptor: file_GetUserMappingsProtocol_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_GetUserMappingsProtocol_proto_goTypes,
		DependencyIndexes: file_GetUserMappingsProtocol_proto_depIdxs,
		MessageInfos:      file_GetUserMappingsProtocol_proto_msgTypes,
	}.Build()
	File_GetUserMappingsProtocol_proto = out.File
	file_GetUserMappingsProtocol_proto_rawDesc = nil
	file_GetUserMappingsProtocol_proto_goTypes = nil
	file_GetUserMappingsProtocol_proto_depIdxs = nil
}
