// Copyright 2020 The Parca Authors
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
	"errors"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
)

var ErrLocationFailedBefore = errors.New("failed to symbolize location, attempts are exhausted")

type DebugInfoFile interface {
	// SourceLines returns the resolved source lines for a given address.
	SourceLines(addr uint64) ([]metastore.LocationLine, error)
}

type DwarfLiner struct {
	logger log.Logger

	dbgFile DebugInfoFile

	attemptThreshold int
	attempts         map[uint64]int
	failed           map[uint64]struct{}
}

// DWARF is a symbolizer that uses DWARF debug info to symbolize addresses.
// TODO(kakkoyun): Introduce functional options for attemptThreshold and demangler.
func DWARF(logger log.Logger, path string, m *pb.Mapping, demangler *demangle.Demangler, attemptThreshold int) (*DwarfLiner, error) {
	dbgFile, err := elfutils.NewDebugInfoFile(path, m, demangler)
	if err != nil {
		return nil, err
	}

	return &DwarfLiner{
		logger:  logger,
		dbgFile: dbgFile,

		attemptThreshold: attemptThreshold,
		attempts:         map[uint64]int{},
		failed:           map[uint64]struct{}{},
	}, nil
}

func (dl *DwarfLiner) PCToLines(addr uint64) (lines []metastore.LocationLine, err error) {
	// Check if we already attempt to symbolize this location and failed.
	if _, failedBefore := dl.failed[addr]; failedBefore {
		level.Debug(dl.logger).Log("msg", "location already had been attempted to be symbolized and failed, skipping")
		return nil, ErrLocationFailedBefore
	}

	defer func() {
		if r := recover(); r != nil {
			err = dl.handleError(addr, fmt.Errorf("recovering from panic in DWARF binary add2line: %v", r))
		}
	}()

	lines, err = dl.dbgFile.SourceLines(addr)
	if err != nil {
		return nil, dl.handleError(addr, err)
	}
	if len(lines) == 0 {
		dl.failed[addr] = struct{}{}
		delete(dl.attempts, addr)
		return nil, errors.New("could not find any frames for given address")
	}

	return lines, nil
}

func (dl *DwarfLiner) handleError(addr uint64, err error) error {
	if prev, ok := dl.attempts[addr]; ok {
		prev++
		if prev >= dl.attemptThreshold {
			dl.failed[addr] = struct{}{}
			delete(dl.attempts, addr)
		} else {
			dl.attempts[addr] = prev
		}
		return err
	}
	// First failed attempt
	dl.attempts[addr] = 1
	return err
}
