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

package index

import (
	"math"
	"testing"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestMemPostings(t *testing.T) {
	p := NewMemPostings()
	p.Add(42, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test1"}})
	p.Add(123, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test2"}})

	empty := []uint64{math.MaxUint64}
	require.Equal(t, []uint64{42, 123}, p.Get("foo", "bar").ToArray())
	require.Equal(t, []uint64{42}, p.Get("container", "test1").ToArray())
	require.Equal(t, []uint64{123}, p.Get("container", "test2").ToArray())
	require.Equal(t, empty, p.Get("container", "test3").ToArray())

	require.ElementsMatch(t, []string{"foo", "container"}, p.LabelNames())
	require.ElementsMatch(t, []string{"test1", "test2"}, p.LabelValues("container"))
}

// TestBitmap was a good learning experience for the sroar.Bitmap.
// I'll leave it for future developers and to some degree as integration test with the library.
func TestBitmap(t *testing.T) {
	b1 := sroar.NewBitmap()
	b1.Set(123)
	require.Equal(t, []uint64{123}, b1.ToArray())

	b2 := sroar.NewBitmap()
	b2.Set(42)
	require.Equal(t, []uint64{42}, b2.ToArray())

	b3 := b1.Clone() // we would be mutating b1 so instead clone as b3
	b3.Or(b2)        // all data in b1 OR b2 (union)
	require.Equal(t, []uint64{42, 123}, b3.ToArray())

	b4 := sroar.NewBitmap()
	b4.SetMany([]uint64{123, 66})
	b4.And(b1) // all data in b4 AND b1 (intersection)
	require.Equal(t, []uint64{123}, b4.ToArray())
}
