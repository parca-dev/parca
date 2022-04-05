// Copyright (c) 2022 The Parca Authors
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
//

package addr2line

import (
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/internal/go/debug/elf"
	"github.com/parca-dev/parca/pkg/metastore"
)

func TestSymtabLiner_PCToLines(t *testing.T) {
	type fields struct {
		symbols []elf.Symbol
	}
	type args struct {
		addr uint64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantLines []metastore.LocationLine
		wantErr   bool
	}{
		{
			name: "no symbols",
			fields: fields{
				symbols: []elf.Symbol{},
			},
			args: args{
				addr: 1,
			},
			wantErr: true,
		},
		{
			name: "no matching symbols",
			fields: fields{
				symbols: []elf.Symbol{
					{
						Name:  "foo",
						Value: 1,
						Size:  3,
					},
					{
						Name:  "bar",
						Value: 2,
						Size:  3,
					},
				},
			},
			args: args{
				addr: 4,
			},
			wantErr: true,
		},
		{
			name: "matching symbols",
			fields: fields{
				symbols: []elf.Symbol{
					{
						Name:  "foo",
						Value: 1,
						Size:  3,
					},
					{
						Name:  "bar",
						Value: 2,
						Size:  3,
					},
					{
						Name:  "baz",
						Value: 3,
						Size:  3,
					},
				},
			},
			args: args{
				addr: 1,
			},
			wantLines: []metastore.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:     "foo",
						Filename: "?",
					},
					Line: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lnr := &SymtabLiner{
				logger:  log.NewNopLogger(),
				symbols: tt.fields.symbols,
			}
			gotLines, err := lnr.PCToLines(tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("PCToLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.wantLines, gotLines)
		})
	}
}
