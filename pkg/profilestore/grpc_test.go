// Copyright 2026 The Parca Authors
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

package profilestore

import (
	"bytes"
	"context"
	"net"
	"sync"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
	otlppb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
)

type otlpTestServer struct {
	otlppb.UnimplementedProfilesServiceServer
	mu     sync.Mutex
	export func(context.Context, *otlppb.ExportProfilesServiceRequest) (*otlppb.ExportProfilesServiceResponse, error)
}

func (s *otlpTestServer) Export(ctx context.Context, req *otlppb.ExportProfilesServiceRequest) (*otlppb.ExportProfilesServiceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.export(ctx, req)
}

func TestOTLPForwarder(t *testing.T) {
	ctx := t.Context()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	upstream := &otlpTestServer{}
	// Register only OTLP: forwarding must not depend on any Parca upstream API.
	otlppb.RegisterProfilesServiceServer(server, upstream)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///upstream",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	forwarder := NewGRPCForwarder(NewClient(nil, otlppb.NewProfilesServiceClient(conn)), log.NewNopLogger(), true)

	request := &profilestorepb.WriteRawRequest{}
	for i, name := range []string{"checkout", "inventory"} {
		fn := &profile.Function{ID: 1, Name: name + ".work", Filename: name + ".go"}
		loc := &profile.Location{ID: 1, Address: uint64(0x1000 + i), Line: []profile.Line{{Function: fn, Line: 42}}}
		p := &profile.Profile{
			SampleType: []*profile.ValueType{{Type: "alloc_objects", Unit: "count"}, {Type: "alloc_space", Unit: "bytes"}},
			Sample: []*profile.Sample{{
				Location: []*profile.Location{loc}, Value: []int64{int64(3 + i), int64(512 + i)},
				Label:    map[string][]string{"service.name": {"sample-" + name}},
				NumLabel: map[string][]int64{"request.size": {int64(17 + i)}},
				NumUnit:  map[string][]string{"request.size": {"bytes"}},
			}},
			Function: []*profile.Function{fn}, Location: []*profile.Location{loc},
			TimeNanos: 1700000000000000000, DurationNanos: 123456789,
			// No PeriodType: valid pprof profiles can omit it.
		}
		var data bytes.Buffer
		if i == 0 {
			require.NoError(t, p.Write(&data))
		} else {
			require.NoError(t, p.WriteUncompressed(&data))
		}
		request.Series = append(request.Series, &profilestorepb.RawProfileSeries{
			Labels: &profilestorepb.LabelSet{Labels: []*profilestorepb.Label{
				{Name: "service.name", Value: name}, {Name: "k8s.namespace.name", Value: "shop"},
			}},
			Samples: []*profilestorepb.RawSample{{RawProfile: data.Bytes()}},
		})
	}

	requests := make(chan *otlppb.ExportProfilesServiceRequest, 1)
	upstream.export = func(_ context.Context, req *otlppb.ExportProfilesServiceRequest) (*otlppb.ExportProfilesServiceResponse, error) {
		requests <- req
		return &otlppb.ExportProfilesServiceResponse{}, nil
	}
	_, err = forwarder.WriteRaw(ctx, request)
	require.NoError(t, err)
	received := <-requests
	require.NotNil(t, received)
	require.Len(t, received.ResourceProfiles, 2)
	dictionary := received.Dictionary
	ids := map[string]bool{}
	for i, resource := range received.ResourceProfiles {
		name := []string{"checkout", "inventory"}[i]
		attrs := map[string]string{}
		for _, attr := range resource.Resource.Attributes {
			attrs[attr.Key] = attr.Value.GetStringValue()
		}
		require.Equal(t, map[string]string{"service.name": name, "k8s.namespace.name": "shop"}, attrs)
		require.Len(t, resource.ScopeProfiles, 1)
		require.Len(t, resource.ScopeProfiles[0].Profiles, 2)
		values := map[string]int64{}
		for _, p := range resource.ScopeProfiles[0].Profiles {
			require.Len(t, p.ProfileId, 16)
			require.NotEqual(t, make([]byte, 16), p.ProfileId)
			require.False(t, ids[string(p.ProfileId)])
			ids[string(p.ProfileId)] = true
			require.Equal(t, uint64(1700000000000000000), p.TimeUnixNano)
			require.Equal(t, uint64(123456789), p.DurationNano)
			require.Len(t, p.Samples, 1)
			sample := p.Samples[0]
			require.Len(t, sample.Values, 1)
			values[dictionary.StringTable[p.SampleType.TypeStrindex]] = sample.Values[0]
			sampleAttrs := map[string]*commonpb.AnyValue{}
			for _, index := range sample.AttributeIndices {
				attr := dictionary.AttributeTable[index]
				sampleAttrs[dictionary.StringTable[attr.KeyStrindex]] = attr.Value
				if dictionary.StringTable[attr.KeyStrindex] == "request.size" {
					require.Equal(t, "bytes", dictionary.StringTable[attr.UnitStrindex])
				}
			}
			require.Len(t, sampleAttrs, 2)
			require.Equal(t, "sample-"+name, sampleAttrs["service.name"].GetStringValue())
			require.Equal(t, int64(17+i), sampleAttrs["request.size"].GetIntValue())
			stack := dictionary.StackTable[sample.StackIndex]
			require.Len(t, stack.LocationIndices, 1)
			loc := dictionary.LocationTable[stack.LocationIndices[0]]
			require.Equal(t, uint64(0x1000+i), loc.Address)
			require.Equal(t, name+".work", dictionary.StringTable[dictionary.FunctionTable[loc.Lines[0].FunctionIndex].NameStrindex])
		}
		require.Equal(t, map[string]int64{"alloc_objects": int64(3 + i), "alloc_space": int64(512 + i)}, values)
	}

	t.Run("OTLP conversion stays opt-in", func(t *testing.T) {
		legacy := NewGRPCForwarder(NewClient(profilestorepb.NewProfileStoreServiceClient(conn),
			otlppb.NewProfilesServiceClient(conn)), log.NewNopLogger(), false)
		_, err := legacy.WriteRaw(ctx, request)
		require.Equal(t, codes.Unimplemented, status.Code(err), "Parca mode must still use the Parca upstream API")
	})
	t.Run("invalid pprof", func(t *testing.T) {
		_, err := forwarder.WriteRaw(ctx, &profilestorepb.WriteRawRequest{Series: []*profilestorepb.RawProfileSeries{{
			Samples: []*profilestorepb.RawSample{{RawProfile: []byte("not pprof")}},
		}}})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
	t.Run("upstream error", func(t *testing.T) {
		upstream.mu.Lock()
		upstream.export = func(context.Context, *otlppb.ExportProfilesServiceRequest) (*otlppb.ExportProfilesServiceResponse, error) {
			return nil, status.Error(codes.Unavailable, "upstream offline")
		}
		upstream.mu.Unlock()
		_, err := forwarder.WriteRaw(ctx, request)
		require.Equal(t, codes.Unavailable, status.Code(err))
	})
	t.Run("partial rejection", func(t *testing.T) {
		upstream.mu.Lock()
		upstream.export = func(context.Context, *otlppb.ExportProfilesServiceRequest) (*otlppb.ExportProfilesServiceResponse, error) {
			return &otlppb.ExportProfilesServiceResponse{PartialSuccess: &otlppb.ExportProfilesPartialSuccess{
				RejectedProfiles: 1, ErrorMessage: "unsupported sample type",
			}}, nil
		}
		upstream.mu.Unlock()
		_, err := forwarder.WriteRaw(ctx, request)
		require.Equal(t, codes.DataLoss, status.Code(err))
		require.Contains(t, err.Error(), "unsupported sample type")
	})
}
