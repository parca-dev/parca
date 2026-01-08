// Copyright 2024-2026 The Parca Authors
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

package normalizer

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func MustReadAllGzip(t testing.TB, filename string) []byte {
	t.Helper()

	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

func Test_Normalizer(t *testing.T) {
	ctx := context.Background()

	fileContent := MustReadAllGzip(t, "./profile.pb.gz")

	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(fileContent))

	_, err := NormalizePprof(ctx, t.Name(), nil, p, true, nil)
	require.NoError(t, err)
}
