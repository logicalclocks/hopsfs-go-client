//*
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//*
// These .proto interfaces are private and stable.
// Please see http://wiki.apache.org/hadoop/Compatibility
// for what changes are allowed for a *stable* .proto interface.

// This file contains protocol buffers that are used throughout HDFS -- i.e.
// by the client, server, and data transfer protocols.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v5.28.2
// source: InterDatanodeProtocol.proto

package hadoop_hdfs

import (
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

// *
// Block with location information and new generation stamp
// to be used for recovery.
type InitReplicaRecoveryRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Block *RecoveringBlockProto `protobuf:"bytes,1,req,name=block" json:"block,omitempty"`
}

func (x *InitReplicaRecoveryRequestProto) Reset() {
	*x = InitReplicaRecoveryRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_InterDatanodeProtocol_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InitReplicaRecoveryRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitReplicaRecoveryRequestProto) ProtoMessage() {}

func (x *InitReplicaRecoveryRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_InterDatanodeProtocol_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitReplicaRecoveryRequestProto.ProtoReflect.Descriptor instead.
func (*InitReplicaRecoveryRequestProto) Descriptor() ([]byte, []int) {
	return file_InterDatanodeProtocol_proto_rawDescGZIP(), []int{0}
}

func (x *InitReplicaRecoveryRequestProto) GetBlock() *RecoveringBlockProto {
	if x != nil {
		return x.Block
	}
	return nil
}

// *
// Repica recovery information
type InitReplicaRecoveryResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ReplicaFound *bool `protobuf:"varint,1,req,name=replicaFound" json:"replicaFound,omitempty"`
	// The following entries are not set if there was no replica found.
	State *ReplicaStateProto `protobuf:"varint,2,opt,name=state,enum=hadoop.hdfs.ReplicaStateProto" json:"state,omitempty"` // State of the replica
	Block *BlockProto        `protobuf:"bytes,3,opt,name=block" json:"block,omitempty"`                                     // block information
}

func (x *InitReplicaRecoveryResponseProto) Reset() {
	*x = InitReplicaRecoveryResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_InterDatanodeProtocol_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InitReplicaRecoveryResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitReplicaRecoveryResponseProto) ProtoMessage() {}

func (x *InitReplicaRecoveryResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_InterDatanodeProtocol_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitReplicaRecoveryResponseProto.ProtoReflect.Descriptor instead.
func (*InitReplicaRecoveryResponseProto) Descriptor() ([]byte, []int) {
	return file_InterDatanodeProtocol_proto_rawDescGZIP(), []int{1}
}

func (x *InitReplicaRecoveryResponseProto) GetReplicaFound() bool {
	if x != nil && x.ReplicaFound != nil {
		return *x.ReplicaFound
	}
	return false
}

func (x *InitReplicaRecoveryResponseProto) GetState() ReplicaStateProto {
	if x != nil && x.State != nil {
		return *x.State
	}
	return ReplicaStateProto_FINALIZED
}

func (x *InitReplicaRecoveryResponseProto) GetBlock() *BlockProto {
	if x != nil {
		return x.Block
	}
	return nil
}

// *
// Update replica with new generation stamp and length
type UpdateReplicaUnderRecoveryRequestProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Block      *ExtendedBlockProto `protobuf:"bytes,1,req,name=block" json:"block,omitempty"`            // Block identifier
	RecoveryId *uint64             `protobuf:"varint,2,req,name=recoveryId" json:"recoveryId,omitempty"` // New genstamp of the replica
	NewLength  *uint64             `protobuf:"varint,3,req,name=newLength" json:"newLength,omitempty"`   // New length of the replica
	// New blockId for copy (truncate), default is 0.
	NewBlockId *uint64 `protobuf:"varint,4,opt,name=newBlockId,def=0" json:"newBlockId,omitempty"`
}

// Default values for UpdateReplicaUnderRecoveryRequestProto fields.
const (
	Default_UpdateReplicaUnderRecoveryRequestProto_NewBlockId = uint64(0)
)

func (x *UpdateReplicaUnderRecoveryRequestProto) Reset() {
	*x = UpdateReplicaUnderRecoveryRequestProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_InterDatanodeProtocol_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateReplicaUnderRecoveryRequestProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateReplicaUnderRecoveryRequestProto) ProtoMessage() {}

func (x *UpdateReplicaUnderRecoveryRequestProto) ProtoReflect() protoreflect.Message {
	mi := &file_InterDatanodeProtocol_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateReplicaUnderRecoveryRequestProto.ProtoReflect.Descriptor instead.
func (*UpdateReplicaUnderRecoveryRequestProto) Descriptor() ([]byte, []int) {
	return file_InterDatanodeProtocol_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateReplicaUnderRecoveryRequestProto) GetBlock() *ExtendedBlockProto {
	if x != nil {
		return x.Block
	}
	return nil
}

func (x *UpdateReplicaUnderRecoveryRequestProto) GetRecoveryId() uint64 {
	if x != nil && x.RecoveryId != nil {
		return *x.RecoveryId
	}
	return 0
}

func (x *UpdateReplicaUnderRecoveryRequestProto) GetNewLength() uint64 {
	if x != nil && x.NewLength != nil {
		return *x.NewLength
	}
	return 0
}

func (x *UpdateReplicaUnderRecoveryRequestProto) GetNewBlockId() uint64 {
	if x != nil && x.NewBlockId != nil {
		return *x.NewBlockId
	}
	return Default_UpdateReplicaUnderRecoveryRequestProto_NewBlockId
}

// *
// Response returns updated block information
type UpdateReplicaUnderRecoveryResponseProto struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StorageUuid *string `protobuf:"bytes,1,opt,name=storageUuid" json:"storageUuid,omitempty"` // ID of the storage that stores replica
}

func (x *UpdateReplicaUnderRecoveryResponseProto) Reset() {
	*x = UpdateReplicaUnderRecoveryResponseProto{}
	if protoimpl.UnsafeEnabled {
		mi := &file_InterDatanodeProtocol_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateReplicaUnderRecoveryResponseProto) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateReplicaUnderRecoveryResponseProto) ProtoMessage() {}

func (x *UpdateReplicaUnderRecoveryResponseProto) ProtoReflect() protoreflect.Message {
	mi := &file_InterDatanodeProtocol_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateReplicaUnderRecoveryResponseProto.ProtoReflect.Descriptor instead.
func (*UpdateReplicaUnderRecoveryResponseProto) Descriptor() ([]byte, []int) {
	return file_InterDatanodeProtocol_proto_rawDescGZIP(), []int{3}
}

func (x *UpdateReplicaUnderRecoveryResponseProto) GetStorageUuid() string {
	if x != nil && x.StorageUuid != nil {
		return *x.StorageUuid
	}
	return ""
}

var File_InterDatanodeProtocol_proto protoreflect.FileDescriptor

var file_InterDatanodeProtocol_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x44, 0x61, 0x74, 0x61, 0x6e, 0x6f, 0x64, 0x65, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x68,
	0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x1a, 0x0a, 0x68, 0x64, 0x66, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x10, 0x48, 0x64, 0x66, 0x73, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5a, 0x0a, 0x1f, 0x49, 0x6e, 0x69, 0x74,
	0x52, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x37, 0x0a, 0x05, 0x62,
	0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x68, 0x61, 0x64,
	0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72,
	0x69, 0x6e, 0x67, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x05, 0x62,
	0x6c, 0x6f, 0x63, 0x6b, 0x22, 0xab, 0x01, 0x0a, 0x20, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x22, 0x0a, 0x0c, 0x72, 0x65, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x46, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x02, 0x28, 0x08, 0x52,
	0x0c, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x46, 0x6f, 0x75, 0x6e, 0x64, 0x12, 0x34, 0x0a,
	0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1e, 0x2e, 0x68,
	0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x52, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x53, 0x74, 0x61, 0x74, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x05, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x12, 0x2d, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x17, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73,
	0x2e, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x05, 0x62, 0x6c, 0x6f,
	0x63, 0x6b, 0x22, 0xc0, 0x01, 0x0a, 0x26, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72,
	0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x35, 0x0a,
	0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x68,
	0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x45, 0x78, 0x74, 0x65, 0x6e,
	0x64, 0x65, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x05, 0x62,
	0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79,
	0x49, 0x64, 0x18, 0x02, 0x20, 0x02, 0x28, 0x04, 0x52, 0x0a, 0x72, 0x65, 0x63, 0x6f, 0x76, 0x65,
	0x72, 0x79, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x65, 0x77, 0x4c, 0x65, 0x6e, 0x67, 0x74,
	0x68, 0x18, 0x03, 0x20, 0x02, 0x28, 0x04, 0x52, 0x09, 0x6e, 0x65, 0x77, 0x4c, 0x65, 0x6e, 0x67,
	0x74, 0x68, 0x12, 0x21, 0x0a, 0x0a, 0x6e, 0x65, 0x77, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x49, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x3a, 0x01, 0x30, 0x52, 0x0a, 0x6e, 0x65, 0x77, 0x42, 0x6c,
	0x6f, 0x63, 0x6b, 0x49, 0x64, 0x22, 0x4b, 0x0a, 0x27, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52,
	0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x52, 0x65, 0x63, 0x6f, 0x76,
	0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x20, 0x0a, 0x0b, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x55, 0x75, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x55, 0x75,
	0x69, 0x64, 0x32, 0x9c, 0x02, 0x0a, 0x1c, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x44, 0x61, 0x74, 0x61,
	0x6e, 0x6f, 0x64, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x72, 0x0a, 0x13, 0x69, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x12, 0x2c, 0x2e, 0x68, 0x61, 0x64,
	0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2d, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f,
	0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x87, 0x01, 0x0a, 0x1a, 0x75, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x52, 0x65,
	0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x12, 0x33, 0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e,
	0x68, 0x64, 0x66, 0x73, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x52, 0x65, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x34, 0x2e, 0x68, 0x61,
	0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65,
	0x52, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x52, 0x65, 0x63, 0x6f,
	0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x42, 0x86, 0x01, 0x0a, 0x25, 0x6f, 0x72, 0x67, 0x2e, 0x61, 0x70, 0x61, 0x63, 0x68, 0x65,
	0x2e, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x2e, 0x68, 0x64, 0x66, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x42, 0x1b, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x44, 0x61, 0x74, 0x61, 0x6e, 0x6f, 0x64, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x5a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6c, 0x69, 0x6e, 0x6d, 0x61, 0x72, 0x63, 0x2f, 0x68,
	0x64, 0x66, 0x73, 0x2f, 0x76, 0x32, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0x68, 0x61, 0x64, 0x6f, 0x6f, 0x70, 0x5f,
	0x68, 0x64, 0x66, 0x73, 0x88, 0x01, 0x01, 0xa0, 0x01, 0x01,
}

var (
	file_InterDatanodeProtocol_proto_rawDescOnce sync.Once
	file_InterDatanodeProtocol_proto_rawDescData = file_InterDatanodeProtocol_proto_rawDesc
)

func file_InterDatanodeProtocol_proto_rawDescGZIP() []byte {
	file_InterDatanodeProtocol_proto_rawDescOnce.Do(func() {
		file_InterDatanodeProtocol_proto_rawDescData = protoimpl.X.CompressGZIP(file_InterDatanodeProtocol_proto_rawDescData)
	})
	return file_InterDatanodeProtocol_proto_rawDescData
}

var file_InterDatanodeProtocol_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_InterDatanodeProtocol_proto_goTypes = []interface{}{
	(*InitReplicaRecoveryRequestProto)(nil),         // 0: hadoop.hdfs.InitReplicaRecoveryRequestProto
	(*InitReplicaRecoveryResponseProto)(nil),        // 1: hadoop.hdfs.InitReplicaRecoveryResponseProto
	(*UpdateReplicaUnderRecoveryRequestProto)(nil),  // 2: hadoop.hdfs.UpdateReplicaUnderRecoveryRequestProto
	(*UpdateReplicaUnderRecoveryResponseProto)(nil), // 3: hadoop.hdfs.UpdateReplicaUnderRecoveryResponseProto
	(*RecoveringBlockProto)(nil),                    // 4: hadoop.hdfs.RecoveringBlockProto
	(ReplicaStateProto)(0),                          // 5: hadoop.hdfs.ReplicaStateProto
	(*BlockProto)(nil),                              // 6: hadoop.hdfs.BlockProto
	(*ExtendedBlockProto)(nil),                      // 7: hadoop.hdfs.ExtendedBlockProto
}
var file_InterDatanodeProtocol_proto_depIdxs = []int32{
	4, // 0: hadoop.hdfs.InitReplicaRecoveryRequestProto.block:type_name -> hadoop.hdfs.RecoveringBlockProto
	5, // 1: hadoop.hdfs.InitReplicaRecoveryResponseProto.state:type_name -> hadoop.hdfs.ReplicaStateProto
	6, // 2: hadoop.hdfs.InitReplicaRecoveryResponseProto.block:type_name -> hadoop.hdfs.BlockProto
	7, // 3: hadoop.hdfs.UpdateReplicaUnderRecoveryRequestProto.block:type_name -> hadoop.hdfs.ExtendedBlockProto
	0, // 4: hadoop.hdfs.InterDatanodeProtocolService.initReplicaRecovery:input_type -> hadoop.hdfs.InitReplicaRecoveryRequestProto
	2, // 5: hadoop.hdfs.InterDatanodeProtocolService.updateReplicaUnderRecovery:input_type -> hadoop.hdfs.UpdateReplicaUnderRecoveryRequestProto
	1, // 6: hadoop.hdfs.InterDatanodeProtocolService.initReplicaRecovery:output_type -> hadoop.hdfs.InitReplicaRecoveryResponseProto
	3, // 7: hadoop.hdfs.InterDatanodeProtocolService.updateReplicaUnderRecovery:output_type -> hadoop.hdfs.UpdateReplicaUnderRecoveryResponseProto
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_InterDatanodeProtocol_proto_init() }
func file_InterDatanodeProtocol_proto_init() {
	if File_InterDatanodeProtocol_proto != nil {
		return
	}
	file_hdfs_proto_init()
	file_HdfsServer_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_InterDatanodeProtocol_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InitReplicaRecoveryRequestProto); i {
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
		file_InterDatanodeProtocol_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InitReplicaRecoveryResponseProto); i {
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
		file_InterDatanodeProtocol_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateReplicaUnderRecoveryRequestProto); i {
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
		file_InterDatanodeProtocol_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateReplicaUnderRecoveryResponseProto); i {
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
			RawDescriptor: file_InterDatanodeProtocol_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_InterDatanodeProtocol_proto_goTypes,
		DependencyIndexes: file_InterDatanodeProtocol_proto_depIdxs,
		MessageInfos:      file_InterDatanodeProtocol_proto_msgTypes,
	}.Build()
	File_InterDatanodeProtocol_proto = out.File
	file_InterDatanodeProtocol_proto_rawDesc = nil
	file_InterDatanodeProtocol_proto_goTypes = nil
	file_InterDatanodeProtocol_proto_depIdxs = nil
}
