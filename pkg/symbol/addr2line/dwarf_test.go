// Copyright 2022 The Parca Authors
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
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

func TestDwarfSymbolizer(t *testing.T) {
	logger := log.NewNopLogger()
	demangler := demangle.NewDemangler("simple", true)
	elfFile, err := elf.Open("testdata/basic-cpp-no-fp-with-debuginfo")
	if err != nil {
		panic("failure opening elf file")
	}
	defer elfFile.Close()

	dwarf, err := DWARF(logger, elfFile, demangler)
	if err != nil {
		panic("failure reading DWARF file")
	}
	gotLines, err := dwarf.PCToLines(0x401125)
	if err != nil {
		panic("failure reading lines")
	}

	require.Equal(t, &metastorev1alpha1.Function{
		Name:     "top2",
		Filename: "src/basic-cpp.cpp",
	}, gotLines[0].Function)
}
