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

package symbol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testResponse() *SymbolicateResponse {
	return &SymbolicateResponse{Modules: []Module{Module{Type: "elf", CodeID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085", ImageAddr: "0x400000"}}, Stacktraces: []Stacktrace{Stacktrace{Frames: []Frame{Frame{Status: "symbolicated", Lang: "go", Symbol: "main.iterate", SymAddr: "0x46377c", Function: "main.iterate", Filename: "main.go", AbsPath: "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", LineNo: 27, InstructionAddr: "0x463781"}, Frame{Status: "symbolicated", Lang: "go", Symbol: "main.iteratePerTenant", SymAddr: "0x463748", Function: "main.iteratePerTenant", Filename: "main.go", AbsPath: "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", LineNo: 23, InstructionAddr: "0x463781"}, Frame{Status: "symbolicated", Lang: "go", Symbol: "main.main", SymAddr: "0x463720", Function: "main.main", Filename: "main.go", AbsPath: "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", LineNo: 10, InstructionAddr: "0x463781"}}}}}
}

func TestSymbolServerClient(t *testing.T) {
	expResp := testResponse()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(expResp)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	c := NewSymbolServerClient(ts.URL)
	res, err := c.Symbolicate(context.Background(), &SymbolicateRequest{
		Modules: []Module{{
			Type:      "elf",
			CodeID:    "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
			ImageAddr: "0x400000",
		}},
		Stacktraces: []Stacktrace{{
			Frames: []Frame{{
				InstructionAddr: "0x463781",
			}},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, expResp, res)
}
