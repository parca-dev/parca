// Copyright 2021 The conprof Authors
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

package api

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestConsistentFlamegraph(t *testing.T) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	var res []byte

	for i := 0; i < 100; i++ {
		root, err := generateFlamegraphReport(p, "")
		require.NoError(t, err)

		newRes, err := json.Marshal(root)
		require.NoError(t, err)

		// Just for the first iteration.
		if res == nil {
			res = newRes
			continue
		}

		if !bytes.Equal(res, newRes) {
			t.Fatal("Flamegraphs are not generated consistently.")
		}
	}
}

func TestDiffFlamegraph(t *testing.T) {
	f, err := os.Open("testdata/diff.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	root, err := generateFlamegraphReport(p, "")
	// Can create the graph nodes without error.
	require.NoError(t, err)

	_, err = json.Marshal(root)
	// Marshals successfully to json.
	require.NoError(t, err)
}
