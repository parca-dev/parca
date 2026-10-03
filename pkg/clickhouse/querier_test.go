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

package clickhouse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// HasProfileData asks whether the table holds anything at all, so it must read
// one row rather than compute every distinct profile type.
//
// It used to answer by calling ProfileTypes(UnixMilli(0), UnixMilli(0)), and
// ProfileTypes only applies its time filter when both bounds are non-zero --
// so the cheapest question in the API ran an unbounded SELECT DISTINCT over six
// columns, scanning a table that grows without bound. On a server with 72h of
// retention that read 108,591,120 rows and 7.9 GB to return 9 rows, every time
// the UI asked.
//
// The shape is asserted rather than the result because this query needs a
// server to run, while the regression is entirely visible in the statement.
func TestHasProfileDataQueryShape(t *testing.T) {
	q := hasProfileDataQuery("parca.stacktraces")

	require.Equal(t, "SELECT 1 FROM parca.stacktraces LIMIT 1", q)

	upper := strings.ToUpper(q)
	require.Contains(t, upper, "LIMIT 1", "existence needs one row")
	require.NotContains(t, upper, "DISTINCT", "existence must not be answered with a distinct scan")
	require.NotContains(t, upper, "GROUP BY")
	require.NotContains(t, upper, "ORDER BY")
}
