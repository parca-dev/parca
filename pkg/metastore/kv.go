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
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

// MakeLocationKey returns the key to be used to store/lookup the location in a
// key-value store.
func MakeLocationKey(l *pb.Location) string {
	return MakeLocationKeyWithID(MakeLocationID(l))
}

// Locations are namespaced by their mapping ID.
// v1/locations/by-key/<hashed-mapping-key>/<hashed-location-key>
const locationsKeyPrefix = "v1/locations/by-key/"

// MakeLocationKeyWithID returns the key to be used to store/lookup a location
// with the provided ID in a key-value store.
func MakeLocationKeyWithID(locationID string) string {
	return locationsKeyPrefix + locationID
}

// Location lines are namespaced by their mapping ID.
// v1/locations-lines/by-key/<hashed-mapping-key>/<hashed-location-key>
const locationLinesKeyPrefix = "v1/location-lines/by-key/"

// MakeLocationLinesKeyWithID returns the key to be used to store/lookup a
// location lines with the provided ID in a key-value store.
func MakeLocationLinesKeyWithID(locationID string) string {
	return locationLinesKeyPrefix + locationID
}

// Unsymbolized locations are namespaced by their mapping ID.
// v1/unsymbolized-locations/by-key/<hashed-mapping-key>/<hashed-location-key>
const unsymbolizedLocationLinesKeyPrefix = "v1/unsymbolized-locations/by-key/"

// MakeUnsymbolizedLocationKeyWithID returns the key to be used to store/lookup
// an unsymbolized location.
func MakeUnsymbolizedLocationKeyWithID(locationID string) string {
	return unsymbolizedLocationLinesKeyPrefix + locationID
}

// LocationIDFromUnsymbolizedKey returns the location ID portion of the provided key.
func LocationIDFromUnsymbolizedKey(key string) string {
	return key[len(unsymbolizedLocationLinesKeyPrefix):]
}

// LocationIDFromKey returns the location ID portion of the provided key.
func LocationIDFromKey(key string) string {
	return key[len(locationsKeyPrefix):]
}

// MakeLocationID returns a key for the location that uniquely identifies the
// location. Locations are uniquely identified by their mapping ID and their
// address and whether the address is folded. If a location address is 0, then
// the lines are expected to be non empty and to be already resolved as they
// cannot be asynchronously symbolized. The lines are then taken into the
// location key.
func MakeLocationID(l *pb.Location) string {
	hash := sha512.New512_256()

	hash.Write([]byte(l.MappingId))
	binary.Write(hash, binary.BigEndian, l.Address)
	if l.IsFolded {
		// If IsFolded is false this means automatically that these 8 bytes are
		// 0. This works out well as the key is byte aligned to the nearest 8
		// bytes that way.
		binary.Write(hash, binary.BigEndian, 1)
	} else {
		binary.Write(hash, binary.BigEndian, 0)
	}

	// If the address is 0, then the functions attached to the
	// location are not from a native binary, but instead from a dynamic
	// runtime/language eg. ruby or python. In those cases we have no better
	// uniqueness factor than the actual functions, and since there is no
	// address there is no potential for asynchronously symbolizing.
	if l.Address == 0 && l.Lines != nil {
		for _, line := range l.Lines.Entries {
			hash.Write([]byte(line.FunctionId))
			binary.Write(hash, binary.BigEndian, line.Line)
		}
	}

	sum := hash.Sum(nil)
	mappingId := l.MappingId
	if mappingId == "" {
		mappingId = "unknown-mapping"
	}
	return mappingId + "/" + base64.URLEncoding.EncodeToString(sum[:])
}

// MakeFunctionKey returns the key to be used to store/lookup the function in a
// key-value store.
func MakeFunctionKey(f *pb.Function) string {
	return MakeFunctionKeyWithID(MakeFunctionID(f))
}

// Functions are namespaced by their filename.
// v1/functions/by-key/<filename-hash>/<hashed-function-key>
const functionKeyPrefix = "v1/functions/by-key/"

// MakeFunctionKeyWithID returns the key to be used to store/lookup a function
// with the provided ID in a key-value store.
func MakeFunctionKeyWithID(functionID string) string {
	return functionKeyPrefix + functionID
}

// FunctionIDFromKey returns the function ID portion of the provided key.
func FunctionIDFromKey(key string) string {
	return key[len(functionKeyPrefix):]
}

// MakeFunctionID returns a key for the function. Functions are uniquely
// identified by their name, filename, starting line number and system name.
func MakeFunctionID(f *pb.Function) string {
	hash := sha512.New512_256()

	binary.Write(hash, binary.BigEndian, f.StartLine)
	hash.Write([]byte(f.Name))
	hash.Write([]byte(f.SystemName))
	hash.Write([]byte(f.Filename))

	sum := hash.Sum(nil)
	if f.Filename == "" {
		return "unknown-filename/" + base64.URLEncoding.EncodeToString(sum[:])
	}

	filenameHash := sha512.Sum512_256([]byte(f.Filename))

	return base64.URLEncoding.EncodeToString(filenameHash[:]) +
		"/" +
		base64.URLEncoding.EncodeToString(sum[:])
}

// Normalize addresses to handle address space randomization.
// Round up to next 4K boundary to avoid minor discrepancies.
const mapsizeRounding = 0x1000

// Mappings are organized by their key directly.
// v1/mappings/by-key/<hashed-mapping-key>
const mappingKeyPrefix = "v1/mappings/by-key/"

// MakeMappingKey returns the key to be used to store/lookup the mapping in a
// key-value store.
func MakeMappingKey(m *pb.Mapping) string {
	return MakeMappingKeyWithID(MakeMappingID(m))
}

// MakeMappingKeyWithID returns the key to be used to store/lookup a mapping
// with the provided ID in a key-value store.
func MakeMappingKeyWithID(mappingID string) string {
	return mappingKeyPrefix + mappingID
}

// MappingIDFromKey returns the mapping ID portion of the provided key.
func MappingIDFromKey(key string) string {
	return key[len(mappingKeyPrefix):]
}

// MakeMappingID returns a key for the mapping. Mappings are uniquely
// identified by their build id (or file if build id is not available), their
// size, and offset.
func MakeMappingID(m *pb.Mapping) string {
	hash := sha512.New512_256()

	size := m.Limit - m.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)

	switch {
	case m.BuildId != "":
		// BuildID has precedence over file as we can rely on it being more
		// unique.
		hash.Write([]byte(m.BuildId))
	case m.File != "":
		hash.Write([]byte(m.File))
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}

	binary.Write(hash, binary.BigEndian, size)
	binary.Write(hash, binary.BigEndian, m.Offset)

	sum := hash.Sum(nil)
	return base64.URLEncoding.EncodeToString(sum[:])
}

// Stacktraces are organized prefixed by their root location and then their full key.
// v1/stacktraces/by-key/<root-location-id>/<hashed-mapping-key>
const stacktraceKeyPrefix = "v1/stacktraces/by-key/"

// MakeStacktraceKey returns the key to be used to store/lookup the mapping in a
// key-value store.
func MakeStacktraceKey(s *pb.Stacktrace) string {
	return MakeStacktraceKeyWithID(MakeStacktraceID(s))
}

// MakeStacktraceKeyWithID returns the key to be used to store/lookup a mapping
// with the provided ID in a key-value store.
func MakeStacktraceKeyWithID(stacktraceID string) string {
	return stacktraceKeyPrefix + stacktraceID
}

// StacktraceIDFromKey returns the mapping ID portion of the provided key.
func StacktraceIDFromKey(key string) string {
	return key[len(stacktraceKeyPrefix):]
}

// MakeStacktraceID returns a key for the stacktrace. Stacktraces are uniquely
// identified by their unique combination and order of locations.
func MakeStacktraceID(s *pb.Stacktrace) string {
	if len(s.LocationIds) == 0 {
		return "empty-stacktrace"
	}

	hash := sha512.New512_256()

	for _, locationID := range s.LocationIds {
		hash.Write([]byte(locationID))
	}

	sum := hash.Sum(nil)
	return string(s.LocationIds[len(s.LocationIds)-1]) + "/" + base64.URLEncoding.EncodeToString(sum[:])
}
