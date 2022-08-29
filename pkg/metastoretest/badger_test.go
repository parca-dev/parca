// Copyright 2022 The Parca Authors
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

package metastoretest

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

func TestUnsymbolizedLocationsPaging(t *testing.T) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx := context.Background()

	metastore := NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	mres, err := metastore.GetOrCreateMappings(ctx, &pb.GetOrCreateMappingsRequest{
		Mappings: []*pb.Mapping{{
			Start:   4194304,
			Limit:   4603904,
			BuildId: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(mres.Mappings))
	m := mres.Mappings[0]

	_, err = metastore.GetOrCreateLocations(ctx, &pb.GetOrCreateLocationsRequest{
		Locations: []*pb.Location{{
			MappingId: m.Id,
			Address:   0x463781,
		}, {
			MappingId: m.Id,
			Address:   0x463782,
		}, {
			MappingId: m.Id,
			Address:   0x463783,
		}},
	})
	require.NoError(t, err)

	lres1, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{
		Limit: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres1.Locations))
	require.Equal(t, uint64(0x463781), lres1.Locations[0].Address)

	lres2, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{
		Limit:  1,
		MinKey: lres1.MaxKey,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres2.Locations))
	require.Equal(t, uint64(0x463783), lres2.Locations[0].Address)

	lres3, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{
		Limit:  1,
		MinKey: lres2.MaxKey,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres3.Locations))
	require.Equal(t, uint64(0x463782), lres3.Locations[0].Address)

	lres4, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{
		Limit:  1,
		MinKey: lres3.MaxKey,
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(lres4.Locations))
}
