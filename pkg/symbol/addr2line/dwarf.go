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

package addr2line

import (
	"debug/elf"
	"fmt"
	"runtime/debug"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

// DwarfLiner is a symbolizer that uses DWARF debug info to symbolize addresses.
type DwarfLiner struct {
	logger log.Logger

	dbgFile elfutils.DebugInfoFile
}

// DWARF creates a new DwarfLiner.
func DWARF(logger log.Logger, f *elf.File, demangler *demangle.Demangler) (*DwarfLiner, error) {
	dbgFile, err := elfutils.NewDebugInfoFile(f, demangler)
	if err != nil {
		return nil, err
	}

	return &DwarfLiner{
		logger:  log.With(logger, "liner", "dwarf"),
		dbgFile: dbgFile,
	}, nil
}

// PCToLines returns the resolved source lines for a program counter (memory address).
func (dl *DwarfLiner) PCToLines(addr uint64) (lines []profile.LocationLine, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered stack trace:\n", string(debug.Stack()))
			err = fmt.Errorf("recovering from panic in DWARF add2line: %v", r)
		}
	}()

	lines, err = dl.dbgFile.SourceLines(addr)
	if err != nil {
		level.Debug(dl.logger).Log("msg", "failed to symbolize location", "addr", fmt.Sprintf("0x%x", addr), "err", err)
		return nil, err
	}
	return lines, nil
}
