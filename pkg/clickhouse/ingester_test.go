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
	"testing"

	"github.com/stretchr/testify/require"
	pprofextended "go.opentelemetry.io/proto/otlp/profiles/v1development"

	"github.com/parca-dev/parca/pkg/profile"
)

func TestDecodeLineInfoReadsFunctionAfterColumn(t *testing.T) {
	stringTable := []string{"", "handle_request", "app.py"}
	funcs := []*pprofextended.Function{{NameStrindex: 1, FilenameStrindex: 2, StartLine: 10}}

	for _, column := range []int64{0, 1, 300} {
		loc := &pprofextended.Location{Lines: []*pprofextended.Line{{FunctionIndex: 1, Line: 42, Column: column}}}
		info := decodeLineInfo(profile.EncodeOtelLocation(nil, loc, nil, funcs, stringTable))

		require.Equal(t, int64(42), info.LineNumber)
		require.Equal(t, int64(10), info.FunctionStartLine)
		require.Equal(t, "handle_request", info.FunctionName)
		require.Equal(t, "app.py", info.FunctionFilename)
	}
}
