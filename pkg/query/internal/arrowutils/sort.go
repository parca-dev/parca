package arrowutils

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"sync"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"golang.org/x/sync/errgroup"

	"github.com/parca-dev/parca/pkg/query/internal/builder"
)

type Direction uint

const (
	Ascending Direction = iota
	Descending
)

func (d Direction) comparison() int {
	switch d {
	case Ascending:
		return -1
	case Descending:
		return 1
	default:
		panic("unexpected direction value " + strconv.Itoa(int(d)) + " only -1 and 1 are allowed")
	}
}

// SortingColumn describes a sorting column on a arrow.Record.
type SortingColumn struct {
	Index      int
	Direction  Direction
	NullsFirst bool
}

// SortRecord sorts given arrow.Record by columns. Returns *array.Int32 of
// indices to sorted rows or record r.
//
// Comparison is made sequentially by each column. When rows are equal in the
// first column we compare the rows om the second column and so on and so forth
// until rows that are not equal are found.
func SortRecord(r arrow.Record, columns []SortingColumn) (*array.Int32, error) {
	if len(columns) == 0 {
		return nil, errors.New("pqarrow/arrowutils: at least one column is needed for sorting")
	}
	ms, err := newMultiColSorter(r, columns)
	if err != nil {
		return nil, err
	}
	defer ms.Release()
	sort.Sort(ms)
	return ms.indices.NewArray().(*array.Int32), nil
}

// Take uses indices which is an array of row index and returns a new record
// that only contains rows specified in indices.
//
// Use compute.WithAllocator to pass a custom memory.Allocator.
func Take(ctx context.Context, r arrow.Record, indices *array.Int32) (arrow.Record, error) {
	// compute.Take doesn't support dictionaries or lists. Use take on r when r
	// does not have these columns.
	var customTake bool
	for i := 0; i < int(r.NumCols()); i++ {
		if r.Column(i).DataType().ID() == arrow.DICTIONARY ||
			r.Column(i).DataType().ID() == arrow.RUN_END_ENCODED ||
			r.Column(i).DataType().ID() == arrow.LIST ||
			r.Column(i).DataType().ID() == arrow.STRUCT {
			customTake = true
			break
		}
	}
	if !customTake {
		res, err := compute.Take(
			ctx,
			compute.TakeOptions{BoundsCheck: true},
			compute.NewDatumWithoutOwning(r),
			compute.NewDatumWithoutOwning(indices),
		)
		if err != nil {
			return nil, err
		}
		return res.(*compute.RecordDatum).Value, nil
	}
	if r.NumCols() == 0 {
		return r, nil
	}

	resArr := make([]arrow.Array, r.NumCols())
	defer func() {
		for _, a := range resArr {
			if a != nil {
				a.Release()
			}
		}
	}()
	var g errgroup.Group
	for i := 0; i < int(r.NumCols()); i++ {
		i := i
		col := r.Column(i)
		switch arr := r.Column(i).(type) {
		case *array.Dictionary:
			g.Go(func() error { return TakeDictColumn(ctx, arr, i, resArr, indices) })
		case *array.RunEndEncoded:
			g.Go(func() error { return TakeRunEndEncodedColumn(ctx, arr, i, resArr, indices) })
		case *array.List:
			g.Go(func() error { return TakeListColumn(ctx, arr, i, resArr, indices) })
		case *array.Struct:
			g.Go(func() error { return TakeStructColumn(ctx, arr, i, resArr, indices) })
		default:
			g.Go(func() error { return TakeColumn(ctx, col, i, resArr, indices) })
		}
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// We checked for at least one column at the beginning of the function.
	expectedLen := resArr[0].Len()
	for _, a := range resArr {
		if a.Len() != expectedLen {
			return nil, fmt.Errorf(
				"pqarrow/arrowutils: expected same length %d for all columns got %d for %s", expectedLen, a.Len(), a.DataType().Name(),
			)
		}
	}
	return array.NewRecord(r.Schema(), resArr, int64(indices.Len())), nil
}

func TakeColumn(ctx context.Context, a arrow.Array, idx int, arr []arrow.Array, indices *array.Int32) error {
	r, err := compute.TakeArray(ctx, a, indices)
	if err != nil {
		return err
	}
	arr[idx] = r
	return nil
}

func TakeDictColumn(ctx context.Context, a *array.Dictionary, idx int, arr []arrow.Array, indices *array.Int32) error {
	switch a.Dictionary().(type) {
	case *array.String, *array.Binary:
		r := array.NewDictionaryBuilderWithDict(
			compute.GetAllocator(ctx), a.DataType().(*arrow.DictionaryType), a.Dictionary(),
		).(*array.BinaryDictionaryBuilder)
		defer r.Release()

		r.Reserve(indices.Len())
		idxBuilder := r.IndexBuilder()
		for _, i := range indices.Int32Values() {
			if a.IsNull(int(i)) {
				r.AppendNull()
				continue
			}
			idxBuilder.Append(a.GetValueIndex(int(i)))
		}

		arr[idx] = r.NewArray()
		return nil
	case *array.FixedSizeBinary:
		r := array.NewDictionaryBuilderWithDict(
			compute.GetAllocator(ctx), a.DataType().(*arrow.DictionaryType), a.Dictionary(),
		).(*array.FixedSizeBinaryDictionaryBuilder)
		defer r.Release()

		r.Reserve(indices.Len())
		idxBuilder := r.IndexBuilder()
		for _, i := range indices.Int32Values() {
			if a.IsNull(int(i)) {
				r.AppendNull()
				continue
			}
			// TODO: Improve this by not copying actual values.
			idxBuilder.Append(a.GetValueIndex(int(i)))
		}

		arr[idx] = r.NewArray()
		return nil
	}

	return nil
}

func TakeRunEndEncodedColumn(ctx context.Context, a *array.RunEndEncoded, idx int, arr []arrow.Array, indices *array.Int32) error {
	expandedIndexBuilder := array.NewInt32Builder(compute.GetAllocator(ctx))
	defer expandedIndexBuilder.Release()

	dict := a.Values().(*array.Dictionary)
	for i := 0; i < a.Len(); i++ {
		if dict.IsNull(a.GetPhysicalIndex(i)) {
			expandedIndexBuilder.AppendNull()
		} else {
			expandedIndexBuilder.Append(int32(dict.GetValueIndex(a.GetPhysicalIndex(i))))
		}
	}
	expandedIndex := expandedIndexBuilder.NewInt32Array()
	defer expandedIndex.Release()

	expandedReorderedArr := make([]arrow.Array, 1)
	if err := TakeColumn(ctx, expandedIndex, 0, expandedReorderedArr, indices); err != nil {
		return err
	}
	expandedReordered := expandedReorderedArr[0].(*array.Int32)
	defer expandedReordered.Release()

	b := array.NewRunEndEncodedBuilder(
		compute.GetAllocator(ctx), a.RunEndsArr().DataType(), a.Values().DataType(),
	)
	defer b.Release()
	b.Reserve(indices.Len())

	dictValues := dict.Dictionary().(*array.String)
	for i := 0; i < expandedReordered.Len(); i++ {
		if expandedReordered.IsNull(i) {
			b.AppendNull()
			continue
		}
		reorderedIndex := expandedReordered.Value(i)
		v := dictValues.Value(int(reorderedIndex))
		if err := b.AppendValueFromString(v); err != nil {
			return err
		}
	}

	arr[idx] = b.NewRunEndEncodedArray()
	return nil
}

func TakeListColumn(ctx context.Context, a *array.List, idx int, arr []arrow.Array, indices *array.Int32) error {
	mem := compute.GetAllocator(ctx)
	r := array.NewBuilder(mem, a.DataType()).(*array.ListBuilder)

	switch valueBuilder := r.ValueBuilder().(type) {
	case *array.BinaryDictionaryBuilder:
		defer valueBuilder.Release()

		listValues := a.ListValues().(*array.Dictionary)
		switch dictV := listValues.Dictionary().(type) {
		case *array.String:
			if err := valueBuilder.InsertStringDictValues(dictV); err != nil {
				return err
			}
		case *array.Binary:
			if err := valueBuilder.InsertDictValues(dictV); err != nil {
				return err
			}
		}
		idxBuilder := valueBuilder.IndexBuilder()

		r.Reserve(indices.Len())
		for _, i := range indices.Int32Values() {
			if a.IsNull(int(i)) {
				r.AppendNull()
				continue
			}

			r.Append(true)
			start, end := a.ValueOffsets(int(i))
			for j := start; j < end; j++ {
				idxBuilder.Append(listValues.GetValueIndex(int(j)))
			}
			// Resize is necessary here for the correct offsets to be appended to
			// the list builder. Otherwise, length will remain at 0.
			valueBuilder.Resize(idxBuilder.Len())
		}

		arr[idx] = r.NewArray()
		return nil
	case *array.StructBuilder:
		defer valueBuilder.Release()

		structArray := a.ListValues().(*array.Struct)

		// expand the indices from the list to each row in the struct.
		structIndicesBuilder := array.NewInt32Builder(mem)
		structIndicesBuilder.Reserve(structArray.Len())
		defer structIndicesBuilder.Release()

		for _, i := range indices.Int32Values() {
			start, end := a.ValueOffsets(int(i))
			for j := start; j < end; j++ {
				structIndicesBuilder.Append(int32(j))
			}
		}
		structIndices := structIndicesBuilder.NewInt32Array()
		defer structIndices.Release()

		arrays := []arrow.Array{structArray}
		err := TakeStructColumn(ctx, structArray, 0, arrays, structIndices)
		if err != nil {
			return err
		}
		defer func() {
			for _, a := range arrays {
				a.Release()
			}
		}()

		newOffsetBuilder := array.NewInt32Builder(mem)
		defer newOffsetBuilder.Release()

		// Build validity bitmap for the list array
		nullBitmapBuilder := array.NewBooleanBuilder(mem)
		defer nullBitmapBuilder.Release()

		newOffsetBuilder.Append(0)
		newOffsetPrevious := int32(0)
		nullCount := 0
		for _, i := range indices.Int32Values() {
			if a.IsNull(int(i)) {
				// If the list is null, repeat the previous offset and set the validity to false
				newOffsetBuilder.Append(newOffsetPrevious)
				nullBitmapBuilder.Append(false)
				nullCount++
				continue
			}

			start, end := a.ValueOffsets(int(i))
			// calculate the length of the current list element and add it to the offsets
			newOffsetPrevious += int32(end - start)
			newOffsetBuilder.Append(newOffsetPrevious)
			nullBitmapBuilder.Append(true)
		}
		newOffsets := newOffsetBuilder.NewInt32Array()
		defer newOffsets.Release()

		// Build validity buffer from the boolean builder
		var validityBuffer *memory.Buffer
		if nullCount > 0 {
			nullBitmap := nullBitmapBuilder.NewBooleanArray()
			defer nullBitmap.Release()
			validityBuffer = nullBitmap.Data().Buffers()[1]
		}

		offsetsBuffer := newOffsets.Data().Buffers()[1]

		data := array.NewData(
			arrow.ListOf(structArray.DataType()),
			indices.Len(),
			[]*memory.Buffer{validityBuffer, offsetsBuffer},
			[]arrow.ArrayData{arrays[0].Data()},
			nullCount,
			0,
		)
		defer data.Release()
		arr[idx] = array.NewListData(data)

		return nil
	default:
		return fmt.Errorf("unexpected value builder type %T for list column", r.ValueBuilder())
	}
}

func TakeStructColumn(ctx context.Context, a *array.Struct, idx int, arr []arrow.Array, indices *array.Int32) error {
	aType := a.Data().DataType().(*arrow.StructType)

	// Immediately, return this struct if it has no fields/columns
	if a.NumField() == 0 {
		// If the original record is released and this is released once more,
		// as usually done, we want to retain it once more.
		a.Retain()
		arr[idx] = a
		return nil
	}

	cols := make([]arrow.Array, a.NumField())
	names := make([]string, a.NumField())
	defer func() {
		for _, col := range cols {
			if col != nil {
				col.Release()
			}
		}
	}()

	for i := 0; i < a.NumField(); i++ {
		names[i] = aType.Field(i).Name

		switch f := a.Field(i).(type) {
		case *array.RunEndEncoded:
			if err := TakeRunEndEncodedColumn(ctx, f, i, cols, indices); err != nil {
				return err
			}
		case *array.Dictionary:
			if err := TakeDictColumn(ctx, f, i, cols, indices); err != nil {
				return err
			}
		case *array.List:
			if err := TakeListColumn(ctx, f, i, cols, indices); err != nil {
				return err
			}
		default:
			err := TakeColumn(ctx, f, i, cols, indices)
			if err != nil {
				return err
			}
		}
	}

	takeStruct, err := array.NewStructArray(cols, names)
	if err != nil {
		return err
	}

	arr[idx] = takeStruct
	return nil
}

type multiColSorter struct {
	indices     *builder.OptInt32Builder
	comparisons []comparator
	directions  []int
	nullsFirst  []bool
}

func newMultiColSorter(
	r arrow.Record,
	columns []SortingColumn,
) (*multiColSorter, error) {
	ms := multiColSorterPool.Get().(*multiColSorter)
	if r.NumRows() <= 1 {
		if r.NumRows() == 1 {
			ms.indices.Append(0)
		}
		return ms, nil
	}
	ms.Reserve(int(r.NumRows()), len(columns))
	for i := range columns {
		ms.directions[i] = columns[i].Direction.comparison()
		ms.nullsFirst[i] = columns[i].NullsFirst
	}
	for i, col := range columns {
		switch e := r.Column(col.Index).(type) {
		case *array.Int16:
			ms.comparisons[i] = newOrderedSorter[int16](e, cmp.Compare)
		case *array.Int32:
			ms.comparisons[i] = newOrderedSorter[int32](e, cmp.Compare)
		case *array.Int64:
			ms.comparisons[i] = newOrderedSorter[int64](e, cmp.Compare)
		case *array.Uint16:
			ms.comparisons[i] = newOrderedSorter[uint16](e, cmp.Compare)
		case *array.Uint32:
			ms.comparisons[i] = newOrderedSorter[uint32](e, cmp.Compare)
		case *array.Uint64:
			ms.comparisons[i] = newOrderedSorter[uint64](e, cmp.Compare)
		case *array.Float64:
			ms.comparisons[i] = newOrderedSorter[float64](e, cmp.Compare)
		case *array.String:
			ms.comparisons[i] = newOrderedSorter[string](e, cmp.Compare)
		case *array.Binary:
			ms.comparisons[i] = newOrderedSorter[[]byte](e, bytes.Compare)
		case *array.Timestamp:
			ms.comparisons[i] = newOrderedSorter[arrow.Timestamp](e, cmp.Compare)
		case *array.Dictionary:
			switch elem := e.Dictionary().(type) {
			case *array.String:
				ms.comparisons[i] = newOrderedSorter[string](
					&stringDictionary{
						dict: e,
						elem: elem,
					},
					cmp.Compare,
				)
			case *array.Binary:
				ms.comparisons[i] = newOrderedSorter[[]byte](
					&binaryDictionary{
						dict: e,
						elem: elem,
					},
					bytes.Compare,
				)
			case *array.FixedSizeBinary:
				ms.comparisons[i] = newOrderedSorter[[]byte](
					&fixedSizeBinaryDictionary{
						dict: e,
						elem: elem,
					},
					bytes.Compare,
				)
			default:
				ms.Release()
				return nil, fmt.Errorf("unsupported dictionary column type for sorting %T for column %s", e, r.Schema().Field(col.Index).Name)
			}
		default:
			ms.Release()
			return nil, fmt.Errorf("unsupported column type for sorting %T for column %s", e, r.Schema().Field(col.Index).Name)
		}
	}
	return ms, nil
}

func (m *multiColSorter) Reserve(rows, columns int) {
	m.indices.Reserve(rows)
	for i := 0; i < rows; i++ {
		m.indices.Set(i, int32(i))
	}
	m.comparisons = slices.Grow(m.comparisons, columns)[:columns]
	m.directions = slices.Grow(m.directions, columns)[:columns]
	m.nullsFirst = slices.Grow(m.nullsFirst, columns)[:columns]
}

func (m *multiColSorter) Reset() {
	m.indices.Reserve(0)
	m.comparisons = m.comparisons[:0]
	m.directions = m.directions[:0]
	m.nullsFirst = m.nullsFirst[:0]
}

func (m *multiColSorter) Release() {
	m.Reset()
	multiColSorterPool.Put(m)
}

var multiColSorterPool = &sync.Pool{
	New: func() any {
		return &multiColSorter{
			indices: builder.NewOptInt32Builder(arrow.PrimitiveTypes.Int32),
		}
	},
}

var _ sort.Interface = (*multiColSorter)(nil)

func (m *multiColSorter) Len() int { return m.indices.Len() }

func (m *multiColSorter) Less(i, j int) bool {
	for idx := range m.comparisons {
		cmp := m.compare(idx, int(m.indices.Value(i)), int(m.indices.Value(j)))
		if cmp != 0 {
			// Use direction to reorder the comparison. Direction determines if the list
			// is in ascending or descending.
			//
			// For instance if comparison between i,j value is -1 and direction is -1
			// this will resolve to true hence the list will be in ascending order. Same
			// principle applies for descending.
			return cmp == m.directions[idx]
		}
		// Try comparing the next column
	}
	return false
}

func (m *multiColSorter) compare(idx, i, j int) int {
	x := m.comparisons[idx]
	if x.IsNull(i) {
		if x.IsNull(j) {
			return 0
		}
		if m.directions[idx] == 1 {
			if m.nullsFirst[idx] {
				return 1
			}
			return -1
		}
		if m.nullsFirst[idx] {
			return -1
		}
		return 1
	}
	if x.IsNull(j) {
		if m.directions[idx] == 1 {
			if m.nullsFirst[idx] {
				return -1
			}
			return 1
		}
		if m.nullsFirst[idx] {
			return 1
		}
		return -1
	}
	return x.Compare(i, j)
}

func (m *multiColSorter) Swap(i, j int) {
	m.indices.Swap(i, j)
}

type comparator interface {
	Compare(i, j int) int
	IsNull(int) bool
}

type orderedArray[T any] interface {
	Value(int) T
	IsNull(int) bool
}

type orderedSorter[T any] struct {
	array   orderedArray[T]
	compare func(T, T) int
}

func newOrderedSorter[T any](a orderedArray[T], compare func(T, T) int) *orderedSorter[T] {
	return &orderedSorter[T]{
		array:   a,
		compare: compare,
	}
}

func (s *orderedSorter[T]) IsNull(i int) bool {
	return s.array.IsNull(i)
}

func (s *orderedSorter[T]) Compare(i, j int) int {
	return s.compare(s.array.Value(i), s.array.Value(j))
}

type stringDictionary struct {
	dict *array.Dictionary
	elem *array.String
}

func (s *stringDictionary) IsNull(i int) bool {
	return s.dict.IsNull(i)
}

func (s *stringDictionary) Value(i int) string {
	return s.elem.Value(s.dict.GetValueIndex(i))
}

type binaryDictionary struct {
	dict *array.Dictionary
	elem *array.Binary
}

func (s *binaryDictionary) IsNull(i int) bool {
	return s.dict.IsNull(i)
}

func (s *binaryDictionary) Value(i int) []byte {
	return s.elem.Value(s.dict.GetValueIndex(i))
}

type fixedSizeBinaryDictionary struct {
	dict *array.Dictionary
	elem *array.FixedSizeBinary
}

func (s *fixedSizeBinaryDictionary) IsNull(i int) bool {
	return s.dict.IsNull(i)
}

func (s *fixedSizeBinaryDictionary) Value(i int) []byte {
	return s.elem.Value(s.dict.GetValueIndex(i))
}
