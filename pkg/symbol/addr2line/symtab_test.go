// Copyright 2022-2023 The Parca Authors
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
	"debug/elf"
	"fmt"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/symbolsearcher"
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
		wantLines []profile.LocationLine
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
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "bar",
						SystemName: "bar",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		{
			name: "first exact address",
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
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "foo",
						SystemName: "foo",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		{
			name: "first non exact address",
			fields: fields{
				symbols: []elf.Symbol{
					{
						Name:  "foo",
						Value: 1,
						Size:  3,
					},
					{
						Name:  "bar",
						Value: 10,
						Size:  3,
					},
					{
						Name:  "baz",
						Value: 20,
						Size:  3,
					},
				},
			},
			args: args{
				addr: 3,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "foo",
						SystemName: "foo",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		{
			name: "last address",
			fields: fields{
				symbols: []elf.Symbol{
					{
						Name:  "foo",
						Value: 1,
						Size:  3,
					},
					{
						Name:  "bar",
						Value: 10,
						Size:  3,
					},
					{
						Name:  "baz",
						Value: 20,
						Size:  3,
					},
				},
			},
			args: args{
				addr: 30,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "baz",
						SystemName: "baz",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		{
			name: "C++ symbols are demangled",
			fields: fields{
				symbols: []elf.Symbol{
					{
						Name:  "_Z2b1v",
						Value: 20,
						Size:  3,
					},
				},
			},
			args: args{
				addr: 20,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "b1",
						SystemName: "_Z2b1v",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// to pass elfSymIsFunction check
			for i := range tt.fields.symbols {
				tt.fields.symbols[i].Section = elf.SectionIndex(1)
				tt.fields.symbols[i].Info = elf.ST_INFO(elf.STB_GLOBAL, elf.STT_FUNC)
			}
			searcher := symbolsearcher.New(tt.fields.symbols)
			lnr := &SymtabLiner{
				logger:    log.NewNopLogger(),
				searcher:  searcher,
				demangler: demangle.NewDemangler("simple", false),
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

func TestSymtabLinerNoPIE(t *testing.T) {
	/* The sampled program below was compiled with gcc 11.3.0 on Ubuntu 22.04.
	gcc -Og -fno-pie -no-pie -fcf-protection=none -o fib-nopie main.c

	#include <stdio.h>

	long fibNaive(long n) {
		if (n <= 2) {
			return 1;
		}
		return fibNaive(n-2) + fibNaive(n-1);
	}

	int main() {
		long n = 50;
		long res = fibNaive(n);
		printf("Fibonacci number %li: %li\n", n, res);
		return 0;
	}
	*/
	filename := "testdata/fib-nopie"
	f, err := elf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})

	// Offset and memory address taken from pprof sample,
	// i.e., mapping.file_offset and mapping.memory_start respectively.
	const (
		mmapOffset = 0x1000
		mmapStart  = 0x401000
	)
	lnr, err := Symbols(
		log.NewNopLogger(),
		filename,
		f,
		mmapOffset,
		mmapStart,
		demangle.NewDemangler("simple", false),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		sampledAddrs []uint64
		wantLines    []profile.LocationLine
	}{
		"fibNaive exact address": {
			sampledAddrs: []uint64{0x401126},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "fibNaive",
						SystemName: "fibNaive",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		"fibNaive non exact address": {
			sampledAddrs: []uint64{
				0x40112a,
				0x40112c,
				0x401131,
				0x401132,
				0x401133,
				0x401134,
				0x401138,
				0x40113b,
				0x40113f,
				0x401144,
				0x401147,
				0x40114b,
				0x401150,
				0x401153,
				0x401157,
				0x401158,
				0x401159,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "fibNaive",
						SystemName: "fibNaive",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		"main exact address": {
			sampledAddrs: []uint64{0x40115a},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "main",
						SystemName: "main",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		"main non exact address": {
			sampledAddrs: []uint64{
				0x40115e,
				0x401163,
				0x401168,
				0x40116b,
				0x401170,
				0x401175,
				0x40117a,
				0x40117f,
				0x401184,
				0x401189,
				0x40118d,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "main",
						SystemName: "main",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
	}

	for name, tc := range tests {
		for _, addr := range tc.sampledAddrs {
			tcName := fmt.Sprintf("%s %x", name, addr)
			t.Run(tcName, func(t *testing.T) {
				gotLines, err := lnr.PCToLines(addr)
				if err != nil {
					t.Error(err)
				}

				require.Equal(t, tc.wantLines, gotLines)
			})
		}
	}
}

func TestSymtabLinerPIE(t *testing.T) {
	// The sampled program was compiled as follows:
	// gcc -o fib-nopie main.c
	filename := "testdata/fib"
	f, err := elf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})

	const (
		mmapOffset = 0x1000
		mmapStart  = 0x5646e2188000
	)
	lnr, err := Symbols(
		log.NewNopLogger(),
		filename,
		f,
		mmapOffset,
		mmapStart,
		demangle.NewDemangler("simple", false),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		sampledAddrs []uint64
		wantLines    []profile.LocationLine
	}{
		"fibNaive exact address": {
			sampledAddrs: []uint64{0x5646e2188149},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "fibNaive",
						SystemName: "fibNaive",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		"fibNaive non exact address": {
			sampledAddrs: []uint64{
				0x5646e218814d,
				0x5646e218814e,
				0x5646e2188151,
				0x5646e2188152,
				0x5646e2188156,
				0x5646e218815a,
				0x5646e218815f,
				0x5646e2188161,
				0x5646e2188166,
				0x5646e2188168,
				0x5646e218816c,
				0x5646e2188170,
				0x5646e2188173,
				0x5646e2188178,
				0x5646e218817b,
				0x5646e218817f,
				0x5646e2188183,
				0x5646e2188186,
				0x5646e218818b,
				0x5646e218818e,
				0x5646e2188192,
				0x5646e2188193,
			},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "fibNaive",
						SystemName: "fibNaive",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
		"main exact address": {
			sampledAddrs: []uint64{0x5646e21881b4},
			wantLines: []profile.LocationLine{
				{
					Function: &metastorev1alpha1.Function{
						Name:       "main",
						SystemName: "main",
						Filename:   "?",
					},
					Line: 0,
				},
			},
		},
	}

	for name, tc := range tests {
		for _, addr := range tc.sampledAddrs {
			tcName := fmt.Sprintf("%s %x", name, addr)
			t.Run(tcName, func(t *testing.T) {
				gotLines, err := lnr.PCToLines(addr)
				if err != nil {
					t.Error(err)
				}

				require.Equal(t, tc.wantLines, gotLines)
			})
		}
	}
}
