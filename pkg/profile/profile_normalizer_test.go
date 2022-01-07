// Copyright 2021 The Parca Authors
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

package profile

import (
	"testing"

	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/stretchr/testify/require"
)

func TestMakeStacktraceKey(t *testing.T) {
	g := metastore.NewLinearUUIDGenerator()

	s := &Sample{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string][]string{"foo": {"bar", "baz"}, "bar": {"baz"}},
		NumLabel: map[string][]int64{"foo": {0, 1}},
		NumUnit:  map[string][]string{"foo": {"cpu", "memory"}},
	}

	k := []byte(MakeStacktraceKey(s))

	require.Len(t, k, 119)

	require.Equal(t,
		[]byte{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
			'|',
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
			'|',
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3,
		},
		k[0:50],
	)

	require.Equal(t,
		[]byte(`"bar"["baz"]"foo"["bar" "baz"]`),
		k[50:80],
	)

	require.Equal(t,
		[]byte{
			'"', 'f', 'o', 'o', '"',
			'[',
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 1,
			']',
			'[', '"', 'c', 'p', 'u', '"', ' ', '"', 'm', 'e', 'm', 'o', 'r', 'y', '"', ']',
		},
		k[80:],
	)
}

func BenchmarkMakeStacktraceKey(b *testing.B) {
	g := metastore.NewLinearUUIDGenerator()
	s := &Sample{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string][]string{"foo": {"bar", "baz"}},
		NumLabel: map[string][]int64{"foo": {0, 1}},
		NumUnit:  map[string][]string{"foo": {"cpu", "memory"}},
	}

	b.ReportAllocs()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = MakeStacktraceKey(s)
	}
}
