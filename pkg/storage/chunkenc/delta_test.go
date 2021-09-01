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

package chunkenc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeltaNonZeroFirstValue(t *testing.T) {
	c := NewDeltaChunk()
	app, err := c.Appender()
	require.NoError(t, err)

	app.Append(3)
	app.Append(5)
	app.Append(7)

	it := c.Iterator(nil)
	require.True(t, it.Next())
	require.Equal(t, int64(3), it.At())
	require.True(t, it.Next())
	require.Equal(t, int64(5), it.At())
	require.True(t, it.Next())
	require.Equal(t, int64(7), it.At())
	require.False(t, it.Next())
}
