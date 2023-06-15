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

package metastore

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"sync"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

func NewKeyMaker() *KeyMaker {
	return &KeyMaker{
		pool: sync.Pool{New: func() interface{} {
			return &bytes.Buffer{}
		}},
	}
}

// KeyMaker is responsible for creating keys used in BadgerMetastore.
type KeyMaker struct {
	// pool is a pool of buffers.
	pool sync.Pool
}

// MakeLocationKey returns the key to be used to store/lookup the location in a
// key-value store.
func (m *KeyMaker) MakeLocationKey(l *pb.Location) string {
	return MakeLocationKeyWithID(m.MakeLocationID(l))
}

// MakeLocationID returns a key for the location that uniquely identifies the
// location. Locations are uniquely identified by their mapping ID and their
// address and whether the address is folded. If a location address is 0, then
// the lines are expected to be non empty and to be already resolved as they
// cannot be asynchronously symbolized. The lines are then taken into the
// location key.
func (m *KeyMaker) MakeLocationID(l *pb.Location) string {
	hbuf := m.pool.Get().(*bytes.Buffer)
	defer m.pool.Put(hbuf)

	hbuf.Reset()
	hbuf.WriteString(l.MappingId)

	// ibuf is a buffer that is used to encode integers.
	ibuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ibuf, l.Address)
	hbuf.Write(ibuf)

	// If the address is 0, then the functions attached to the
	// location are not from a native binary, but instead from a dynamic
	// runtime/language eg. ruby or python. In those cases we have no better
	// uniqueness factor than the actual functions, and since there is no
	// address there is no potential for asynchronously symbolizing.
	if l.Address == 0 {
		for _, line := range l.Lines {
			hbuf.WriteString(line.FunctionId)

			binary.BigEndian.PutUint64(ibuf, uint64(line.Line))
			hbuf.Write(ibuf)
		}
	}

	sum := sha512.Sum512_256(hbuf.Bytes())
	b := unsafeURLEncode(sum, hbuf)

	mappingId := l.MappingId
	if mappingId == "" {
		mappingId = "unknown-mapping"
	}

	return mappingId + "/" + string(b)
}

// MakeFunctionKey returns the key to be used to store/lookup the function in a
// key-value store.
func (m *KeyMaker) MakeFunctionKey(f *pb.Function) string {
	return MakeFunctionKeyWithID(m.MakeFunctionID(f))
}

// MakeFunctionID returns a key for the function. Functions are uniquely
// identified by their name, filename, starting line number and system name.
func (m *KeyMaker) MakeFunctionID(f *pb.Function) string {
	hbuf := m.pool.Get().(*bytes.Buffer)
	defer m.pool.Put(hbuf)

	// ibuf is a buffer that is used to encode integers.
	ibuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ibuf, uint64(f.StartLine))
	hbuf.Reset()
	hbuf.Write(ibuf)

	hbuf.WriteString(f.Name)
	hbuf.WriteString(f.SystemName)
	hbuf.WriteString(f.Filename)

	sum := sha512.Sum512_256(hbuf.Bytes())
	b := unsafeURLEncode(sum, hbuf)

	if f.Filename == "" {
		return "unknown-filename/" + string(b)
	}

	fbuf := m.pool.Get().(*bytes.Buffer)
	defer m.pool.Put(fbuf)

	fbuf.Reset()
	fbuf.WriteString(f.Filename)
	filenameHash := sha512.Sum512_256(fbuf.Bytes())
	fb := unsafeURLEncode(filenameHash, fbuf)

	return string(fb) + "/" + string(b)
}

// MakeMappingKey returns the key to be used to store/lookup the mapping in a
// key-value store.
func (m *KeyMaker) MakeMappingKey(mp *pb.Mapping) string {
	return MakeMappingKeyWithID(m.MakeMappingID(mp))
}

// MakeMappingID returns a key for the mapping. Mappings are uniquely
// identified by their build id (or file if build id is not available), their
// size, and offset.
func (m *KeyMaker) MakeMappingID(mp *pb.Mapping) string {
	hbuf := m.pool.Get().(*bytes.Buffer)
	defer m.pool.Put(hbuf)

	size := mp.Limit - mp.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)

	hbuf.Reset()
	switch {
	case mp.BuildId != "":
		// BuildID has precedence over file as we can rely on it being more
		// unique.
		hbuf.WriteString(mp.BuildId)
	case mp.File != "":
		hbuf.WriteString(mp.File)
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}

	// ibuf is a buffer that is used to encode integers.
	ibuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ibuf, size)
	hbuf.Write(ibuf)
	binary.BigEndian.PutUint64(ibuf, mp.Offset)
	hbuf.Write(ibuf)

	sum := sha512.Sum512_256(hbuf.Bytes())
	b := unsafeURLEncode(sum, hbuf)

	return string(b)
}

// MakeStacktraceKey returns the key to be used to store/lookup the mapping in a
// key-value store.
func (m *KeyMaker) MakeStacktraceKey(s *pb.Stacktrace) string {
	return MakeStacktraceKeyWithID(m.MakeStacktraceID(s))
}

// MakeStacktraceID returns a key for the stacktrace. Stacktraces are uniquely
// identified by their unique combination and order of locations.
func (m *KeyMaker) MakeStacktraceID(s *pb.Stacktrace) string {
	if len(s.LocationIds) == 0 {
		return "empty-stacktrace"
	}

	hbuf := m.pool.Get().(*bytes.Buffer)
	defer m.pool.Put(hbuf)

	hbuf.Reset()
	for _, locationID := range s.LocationIds {
		hbuf.WriteString(locationID)
	}

	sum := sha512.Sum512_256(hbuf.Bytes())
	b := unsafeURLEncode(sum, hbuf)

	return s.LocationIds[len(s.LocationIds)-1] + "/" + string(b)
}

// unsafeURLEncode base64 encodes the hash sum using the supplied buffer
// to avoid allocations.
// Note, once the buffer is modified in any fashion,
// the returned byte slice will be affected as well because
// it's part of the buffer.
func unsafeURLEncode(sum [32]byte, buf *bytes.Buffer) []byte {
	buf.Reset()

	hashLen := base64.URLEncoding.EncodedLen(len(sum))
	buf.Grow(hashLen)
	b := buf.Bytes()[:hashLen]
	base64.URLEncoding.Encode(b, sum[:])

	return b
}

// Locations are namespaced by their mapping ID
// `v1/locations/by-key/<hashed-mapping-key>/<hashed-location-key>`.
const locationsKeyPrefix = "v1/locations/by-key/"

// MakeLocationKeyWithID returns the key to be used to store/lookup a location
// with the provided ID in a key-value store.
func MakeLocationKeyWithID(locationID string) string {
	return locationsKeyPrefix + locationID
}

// Unsymbolized locations are namespaced by their mapping ID.
// `v1/unsymbolized-locations/by-key/<hashed-mapping-key>/<hashed-location-key>`.
const UnsymbolizedLocationLinesKeyPrefix = "v1/unsymbolized-locations/by-key/"

// MakeUnsymbolizedLocationKeyWithID returns the key to be used to store/lookup
// an unsymbolized location.
func MakeUnsymbolizedLocationKeyWithID(locationID string) string {
	return UnsymbolizedLocationLinesKeyPrefix + locationID
}

// LocationIDFromUnsymbolizedKey returns the location ID portion of the provided key.
func LocationIDFromUnsymbolizedKey(key string) string {
	return key[len(UnsymbolizedLocationLinesKeyPrefix):]
}

// LocationIDFromKey returns the location ID portion of the provided key.
func LocationIDFromKey(key string) string {
	return key[len(locationsKeyPrefix):]
}

// Functions are namespaced by their filename.
// `v1/functions/by-key/<filename-hash>/<hashed-function-key>`.
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

// Normalize addresses to handle address space randomization.
// Round up to next 4K boundary to avoid minor discrepancies.
const mapsizeRounding = 0x1000

// Mappings are organized by their key directly.
// `v1/mappings/by-key/<hashed-mapping-key>`.
const mappingKeyPrefix = "v1/mappings/by-key/"

// MakeMappingKeyWithID returns the key to be used to store/lookup a mapping
// with the provided ID in a key-value store.
func MakeMappingKeyWithID(mappingID string) string {
	return mappingKeyPrefix + mappingID
}

// MappingIDFromKey returns the mapping ID portion of the provided key.
func MappingIDFromKey(key string) string {
	return key[len(mappingKeyPrefix):]
}

// Stacktraces are organized prefixed by their root location and then their full key.
// `v1/stacktraces/by-key/<root-location-id>/<hashed-mapping-key>`.
const stacktraceKeyPrefix = "v1/stacktraces/by-key/"

// MakeStacktraceKeyWithID returns the key to be used to store/lookup a mapping
// with the provided ID in a key-value store.
func MakeStacktraceKeyWithID(stacktraceID string) string {
	return stacktraceKeyPrefix + stacktraceID
}

// StacktraceIDFromKey returns the mapping ID portion of the provided key.
func StacktraceIDFromKey(key string) string {
	return key[len(stacktraceKeyPrefix):]
}
