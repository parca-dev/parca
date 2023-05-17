// Copyright 2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Based on:
// https://github.com/planetscale/vtprotobuf#mixing-protobuf-implementations-with-grpc
// https://github.com/vitessio/vitess/blob/main/go/vt/servenv/grpc_codec.go

package server

import (
	"fmt"

	// use the original golang/protobuf package we can continue serializing
	// messages from our dependencies, particularly for:
	//   - grpc.health.v1.Health
	//   - grpc.reflection.v1alpha.ServerReflection
	//   - grpc.reflection.v1.ServerReflection
	"google.golang.org/protobuf/proto"

	"google.golang.org/grpc/encoding"
	_ "google.golang.org/grpc/encoding/proto"
)

// Name is the name registered for the proto compressor.
const Name = "proto"

type vtprotoCodec struct{}

type vtprotoMessage interface {
	MarshalVT() ([]byte, error)
	UnmarshalVT([]byte) error
}

func (vtprotoCodec) Marshal(v any) ([]byte, error) {
	switch v := v.(type) {
	case vtprotoMessage:
		return v.MarshalVT()
	case proto.Message:
		return proto.Marshal(v)
	default:
		return nil, fmt.Errorf("failed to marshal, message is %T, must satisfy the vtprotoMessage interface or want proto.Message", v)
	}
}

func (vtprotoCodec) Unmarshal(data []byte, v any) error {
	switch v := v.(type) {
	case vtprotoMessage:
		return v.UnmarshalVT(data)
	case proto.Message:
		return proto.Unmarshal(data, v)
	default:
		return fmt.Errorf("failed to unmarshal, message is %T, must satisfy the vtprotoMessage interface or want proto.Message", v)
	}
}

func (vtprotoCodec) Name() string {
	return Name
}

func init() {
	encoding.RegisterCodec(vtprotoCodec{})
}
