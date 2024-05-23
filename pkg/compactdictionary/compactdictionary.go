// Copyright 2024 The Parca Authors
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

package compactdictionary

import (
	"fmt"
	stdmath "math"

	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/array"
	"github.com/apache/arrow/go/v16/arrow/memory"
)

type Releasable interface {
	Release()
}

// compactDictionary copies only the needed values from the old dictionary to the new dictionary.
// Once all needed values are copied, it updates the indices referencing those values in their new place.
func CompactDictionary(mem memory.Allocator, arr *array.Dictionary) (*array.Dictionary, error) {
	// indices := arr.Indices().(*array.Int32)
	releasers := make([]Releasable, 0, 3)
	releasers = append(releasers, arr)
	defer func() {
		for _, r := range releasers {
			r.Release()
		}
	}()

	newLen := 0
	keepValues := make([]int, arr.Dictionary().Len())

	var indices interface{}

	switch arr.Indices().(type) {
	case *array.Int32:
		indices = arr.Indices().(*array.Int32)

		for i := 0; i < indices.(*array.Int32).Len(); i++ {
			if arr.IsValid(i) {
				if keepValues[indices.(*array.Int32).Value(i)] == 0 {
					// keep track of how many values we need to keep to reserve the space upfront
					newLen++
				}
				keepValues[indices.(*array.Int32).Value(i)]++
			}
		}
	case *array.Uint32:
		indices = arr.Indices().(*array.Uint32)

		for i := 0; i < indices.(*array.Uint32).Len(); i++ {
		if arr.IsValid(i) {
			if keepValues[indices.(*array.Uint32).Value(i)] == 0 {
				// keep track of how many values we need to keep to reserve the space upfront
				newLen++
			}
		}
	}
	default:
		return nil, fmt.Errorf("unsupported indices type %T", arr.Indices())
	}

	// This maps the previous index (at the key/index in this slice) to the new index (at the value of the slice).
	newValueIndices := make([]int, arr.Dictionary().Len())

	var valueBuilder array.Builder
	switch dict := arr.Dictionary().(type) {
	case *array.String:
		stringBuilder := array.NewStringBuilder(mem)
		stringBuilder.Reserve(newLen)
		numBytes := 0
		for i, count := range keepValues {
			if count == 0 {
				continue
			}
			numBytes += len(dict.Value(i))
		}
		stringBuilder.ReserveData(numBytes)
		for i, count := range keepValues {
			if count == 0 {
				continue
			}
			newValueIndices[i] = stringBuilder.Len()
			stringBuilder.Append(dict.Value(i))
		}
		valueBuilder = stringBuilder
		releasers = append(releasers, stringBuilder)
	case *array.Binary:
		binaryBuilder := array.NewBinaryBuilder(mem, arrow.BinaryTypes.Binary)
		binaryBuilder.Reserve(newLen)
		numBytes := 0
		for i, count := range keepValues {
			if count == 0 {
				continue
			}
			numBytes += dict.ValueLen(i)
		}
		binaryBuilder.ReserveData(numBytes)
		for i, count := range keepValues {
			if count == 0 {
				continue
			}
			newValueIndices[i] = binaryBuilder.Len()
			binaryBuilder.Append(dict.Value(i))
		}
		valueBuilder = binaryBuilder
		releasers = append(releasers, binaryBuilder)
	default:
		return nil, fmt.Errorf("unsupported dictionary type %T", arr.Dictionary())
	}

	// we know how many values we need to keep, so we can reserve the space upfront
	var indexBuilder array.Builder
	if newLen < stdmath.MaxUint8 {
		indexBuilder = array.NewUint8Builder(mem)
	} else if newLen < stdmath.MaxUint16 {
		indexBuilder = array.NewUint16Builder(mem)
	} else if newLen < stdmath.MaxUint32 {
		indexBuilder = array.NewUint32Builder(mem)
	} else {
		indexBuilder = array.NewUint64Builder(mem)
	}

	switch arr.Indices().(type) {
	case *array.Int32:
		indexBuilder.Reserve(indices.(*array.Int32).Len())
	case *array.Uint32:
		indexBuilder.Reserve(indices.(*array.Uint32).Len())
	default:
		return nil, fmt.Errorf("unsupported indices type %T", arr.Indices())
	}

	releasers = append(releasers, indexBuilder)

	switch arr.Indices().(type) {
	case *array.Int32:
		for i := 0; i < indices.(*array.Int32).Len(); i++ {
		if arr.IsNull(i) {
			indexBuilder.AppendNull()
			continue
		}
		oldValueIndex := indices.(*array.Int32).Value(i)
		newValueIndex := newValueIndices[oldValueIndex]

		switch b := indexBuilder.(type) {
		case *array.Uint8Builder:
			b.Append(uint8(newValueIndex))
		case *array.Uint16Builder:
			b.Append(uint16(newValueIndex))
		case *array.Uint32Builder:
			b.Append(uint32(newValueIndex))
		case *array.Uint64Builder:
			b.Append(uint64(newValueIndex))
		}
	}
	case *array.Uint32:
		for i := 0; i < indices.(*array.Uint32).Len(); i++ {
			if arr.IsNull(i) {
				indexBuilder.AppendNull()
				continue
			}
			oldValueIndex := indices.(*array.Int32).Value(i)
			newValueIndex := newValueIndices[oldValueIndex]

			switch b := indexBuilder.(type) {
			case *array.Uint8Builder:
				b.Append(uint8(newValueIndex))
			case *array.Uint16Builder:
				b.Append(uint16(newValueIndex))
			case *array.Uint32Builder:
				b.Append(uint32(newValueIndex))
			case *array.Uint64Builder:
				b.Append(uint64(newValueIndex))
			}
		}
	case *array.Uint32:
		for i := 0; i < indices.(*array.Uint32).Len(); i++ {
			if arr.IsNull(i) {
				indexBuilder.AppendNull()
				continue
			}
			oldValueIndex := indices.(*array.Uint32).Value(i)
			newValueIndex := newValueIndices[oldValueIndex]

			switch b := indexBuilder.(type) {
			case *array.Uint8Builder:
				b.Append(uint8(newValueIndex))
			case *array.Uint16Builder:
				b.Append(uint16(newValueIndex))
			case *array.Uint32Builder:
				b.Append(uint32(newValueIndex))
			case *array.Uint64Builder:
				b.Append(uint64(newValueIndex))
			}
		}
	default:
		return nil, fmt.Errorf("unsupported indices type %T", arr.Indices())
	}

	index := indexBuilder.NewArray()
	values := valueBuilder.NewArray()

	releasers = append(releasers, index, values)

	return array.NewDictionaryArray(
		&arrow.DictionaryType{IndexType: index.DataType(), ValueType: valueBuilder.Type()},
		index,
		values,
	), nil
}
