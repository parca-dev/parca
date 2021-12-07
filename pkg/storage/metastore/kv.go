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

package metastore

import (
	"encoding/binary"
	"fmt"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

const (
	v1prefix = "v1"
)

const (
	//const stacktraceKey = "stacktrace/by-key/"
	stacktraceID = "stacktrace/by-id/"
	locationsKey = "locations/by-key/"
	functionKey  = "functions/by-key/"
	mappingKey   = "mappings/by-key/"
)

var (
	stacktraceIDPrefix = versionedPrefix(v1prefix, stacktraceID)
	locationsKeyPrefix = versionedPrefix(v1prefix, locationsKey)
	functionKeyPrefix  = versionedPrefix(v1prefix, functionKey)
	mappingKeyPrefix   = versionedPrefix(v1prefix, mappingKey)
)

func MakeLocationKey(l *Location) []byte {
	normalizedAddress := l.Address
	if l.Mapping != nil {
		// Normalizes address to handle address space randomization.
		normalizedAddress -= l.Mapping.Start
	}

	linesLength := 0
	if normalizedAddress == 0 {
		// Each line is a 16 byte Function ID + 8 byte line number
		linesLength = len(l.Lines) * (16 + 8)
	}

	buf := make(
		[]byte,
		len(locationsKeyPrefix)+
			// MappingID is 16 bytes
			16+
			// NormalizedAddress is 8 bytes
			8+
			// IsFolded is encoded as 8 bytes
			8+
			linesLength,
	)
	copy(buf, locationsKeyPrefix)
	if l.Mapping != nil {
		copy(buf[len(locationsKeyPrefix):], l.Mapping.Id)
	}
	binary.BigEndian.PutUint64(buf[len(locationsKeyPrefix)+16:], normalizedAddress)
	if l.IsFolded {
		// If IsFolded is false this means automatically that these 8 bytes are
		// 0. This works out well as the key is byte aligned to the nearest 8
		// bytes that way.
		binary.BigEndian.PutUint64(buf[len(locationsKeyPrefix)+8+16:], 1)
	}

	// If the normalized address is 0, then the functions attached to the
	// location are not from a native binary, but instead from a dynamic
	// runtime/language eg. ruby or python. In those cases we have no better
	// uniqueness factor than the actual functions, and since there is no
	// address there is no potential for asynchronously symbolizing.
	if normalizedAddress == 0 {
		for i, line := range l.Lines {
			copy(buf[len(locationsKeyPrefix)+16+8+8+24*i:], line.Function.Id)
			binary.BigEndian.PutUint64(buf[len(locationsKeyPrefix)+16+8+8+24*i+8:], uint64(line.Line))
		}
	}
	return buf
}

func MakeFunctionKey(f *pb.Function) []byte {
	buf := make([]byte, len(functionKeyPrefix)+len(f.Name)+len(f.SystemName)+len(f.Filename)+8)
	copy(buf, functionKeyPrefix)
	binary.BigEndian.PutUint64(buf[len(functionKeyPrefix):], uint64(f.StartLine))
	copy(buf[len(functionKeyPrefix)+8:], f.Name)
	copy(buf[len(functionKeyPrefix)+8+len(f.Name):], f.SystemName)
	copy(buf[len(functionKeyPrefix)+8+len(f.Name)+len(f.SystemName):], f.Filename)

	return buf
}

func MakeMappingKey(m *pb.Mapping) []byte {
	// Normalize addresses to handle address space randomization.
	// Round up to next 4K boundary to avoid minor discrepancies.
	const mapsizeRounding = 0x1000

	size := m.Limit - m.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)

	buildIDOrFile := ""
	switch {
	case m.BuildId != "":
		// BuildID has precedence over file as we can rely on it being more
		// unique.
		buildIDOrFile = m.BuildId
	case m.File != "":
		buildIDOrFile = m.File
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}

	buf := make([]byte, len(mappingKeyPrefix)+len(buildIDOrFile)+16)
	copy(buf, mappingKeyPrefix)
	binary.BigEndian.PutUint64(buf[len(mappingKeyPrefix):], size)
	binary.BigEndian.PutUint64(buf[len(mappingKeyPrefix)+8:], m.Offset)
	copy(buf[len(mappingKeyPrefix)+16:], buildIDOrFile)

	return buf
}

func versionedPrefix(version, prefix string) string {
	return fmt.Sprintf("%s/%s", version, prefix)
}
