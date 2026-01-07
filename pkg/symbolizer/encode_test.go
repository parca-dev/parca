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

package symbolizer

import (
	"testing"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestEncodeDecode(t *testing.T) {
	encoded := encodeLines([]profile.LocationLine{{
		Line: 1,
		Function: &pb.Function{
			Name:       "main",
			Filename:   "main.go",
			SystemName: "main",
			StartLine:  1,
		},
	}})

	decoded := decodeLines(encoded)

	if len(decoded) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(decoded))
	}
	if decoded[0].Line != 1 {
		t.Fatalf("Expected line 1, got %d", decoded[0].Line)
	}
	if decoded[0].Function.Name != "main" {
		t.Fatalf("Expected function name main, got %s", decoded[0].Function.Name)
	}
	if decoded[0].Function.Filename != "main.go" {
		t.Fatalf("Expected function filename main.go, got %s", decoded[0].Function.Filename)
	}
	if decoded[0].Function.SystemName != "main" {
		t.Fatalf("Expected function system name main, got %s", decoded[0].Function.SystemName)
	}
	if decoded[0].Function.StartLine != 1 {
		t.Fatalf("Expected function start line 1, got %d", decoded[0].Function.StartLine)
	}
}
