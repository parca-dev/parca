// Copyright 2022-2025 The Parca Authors
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

package symbolizer

import (
	"context"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/filesystem"
	"gopkg.in/yaml.v3"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profile"
)

type NoopSymbolizerCache struct{}

func (n *NoopSymbolizerCache) Get(ctx context.Context, buildID string, addr uint64) ([]profile.LocationLine, bool, error) {
	return nil, false, nil
}

func (n *NoopSymbolizerCache) Set(ctx context.Context, buildID string, addr uint64, lines []profile.LocationLine) error {
	return nil
}

func TestSymbolizer(t *testing.T) {
	logger := log.NewNopLogger()
	symbolizerCacheDir, err := os.MkdirTemp("", "parca-symbolizer-test-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(symbolizerCacheDir)
	})

	cfg, err := yaml.Marshal(&client.BucketConfig{
		Type: client.FILESYSTEM,
		Config: filesystem.Config{
			Directory: "testdata/",
		},
	})
	require.NoError(t, err)

	bucket, err := client.NewBucket(logger, cfg, "parca/store")
	require.NoError(t, err)

	metadata := debuginfo.NewObjectStoreMetadata(logger, bucket)
	debuginfodClient := debuginfo.NopDebuginfodClients{}

	sym := New(
		logger,
		metadata,
		&NoopSymbolizerCache{},
		debuginfo.NewFetcher(debuginfodClient, bucket),
		symbolizerCacheDir,
		"",
	)

	ctx := context.Background()

	mapping := &pb.Mapping{
		Start:   4194304,
		Limit:   4603904,
		BuildId: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}

	location := &profile.Location{
		Mapping: mapping,
		Address: 0x463781,
	}

	err = sym.Symbolize(ctx, SymbolizationRequest{
		BuildID: mapping.BuildId,
		Mappings: []SymbolizationRequestMappingAddrs{{
			Locations: []*profile.Location{location},
		}},
	})
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, 3, len(location.Lines))

	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", location.Lines[0].Function.Filename)
	require.Equal(t, "main.main", location.Lines[0].Function.Name)
	require.Equal(t, int64(7), location.Lines[0].Line) // llvm-addr2line gives 10

	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", location.Lines[1].Function.Filename)
	require.Equal(t, "main.iterate", location.Lines[1].Function.Name)
	require.Equal(t, int64(27), location.Lines[1].Line)

	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", location.Lines[2].Function.Filename)
	require.Equal(t, "main.iteratePerTenant", location.Lines[2].Function.Name)
	require.Equal(t, int64(23), location.Lines[2].Line)
}
