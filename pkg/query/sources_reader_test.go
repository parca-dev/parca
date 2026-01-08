// Copyright 2023-2026 The Parca Authors
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

package query

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindSourcesFile(t *testing.T) {
	cases := []struct {
		name     string
		filename string
	}{
		{
			name:     "gzip",
			filename: "testdata/source.tar.gz",
		},
		{
			name:     "zstd",
			filename: "testdata/source.tar.zstd",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.filename)
			require.NoError(t, err)
			defer f.Close()

			s, err := NewSourcesReader(f)
			require.NoError(t, err)

			_, err = s.Find("reader.go")
			require.NoError(t, err)
		})
	}
}

func BenchmarkFindSourcesFile(b *testing.B) {
	b.Skip()

	resp, err := http.Get("https://github.com/parca-dev/parca/archive/refs/tags/v0.18.0.tar.gz")
	require.NoError(b, err)
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "sources_test")
	require.NoError(b, err)
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	_, err = io.Copy(tmp, resp.Body)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := NewSourcesReader(tmp)
		require.NoError(b, err)

		_, err = s.Find("parca-0.18.0/cmd/parca/main.go")
		require.NoError(b, err)
	}
}
