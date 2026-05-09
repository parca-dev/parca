// Copyright 2026 The Parca Authors
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

package duckdb

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/dennwc/varint"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	duckdb "github.com/marcboeker/go-duckdb/v2"

	"github.com/parca-dev/parca/pkg/profile"
)

// Ingester writes Arrow profile records to DuckDB via the Appender API.
type Ingester struct {
	logger log.Logger
	client *Client
}

// NewIngester returns an Ingester bound to client.
func NewIngester(logger log.Logger, client *Client) *Ingester {
	return &Ingester{logger: logger, client: client}
}

// Ingest writes record into the configured DuckDB table.
//
// Arrow → DuckDB row mapping:
//   - flat columns (name, sample_type, period, ...) → scalar Appender values
//   - labels.<name> Arrow columns → MAP(VARCHAR, VARCHAR) keyed by <name>
//   - stacktrace LIST<binary> (encoded location bytes) → LIST<STRUCT> by
//     decoding each binary blob into a location struct
func (i *Ingester) Ingest(ctx context.Context, record arrow.RecordBatch) error {
	if record.NumRows() == 0 {
		return nil
	}

	conn, err := i.client.DB().Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire duckdb connection: %w", err)
	}
	defer conn.Close()

	var appender *duckdb.Appender
	if rawErr := conn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("duckdb raw connection is not a driver.Conn (got %T)", driverConn)
		}
		var aerr error
		appender, aerr = duckdb.NewAppenderFromConn(dc, "", i.client.Table())
		return aerr
	}); rawErr != nil {
		return fmt.Errorf("create appender: %w", rawErr)
	}
	defer appender.Close()

	schema := record.Schema()

	nameIdx := findColumnIndex(schema, profile.ColumnName)
	sampleTypeIdx := findColumnIndex(schema, profile.ColumnSampleType)
	sampleUnitIdx := findColumnIndex(schema, profile.ColumnSampleUnit)
	periodTypeIdx := findColumnIndex(schema, profile.ColumnPeriodType)
	periodUnitIdx := findColumnIndex(schema, profile.ColumnPeriodUnit)
	periodIdx := findColumnIndex(schema, profile.ColumnPeriod)
	durationIdx := findColumnIndex(schema, profile.ColumnDuration)
	timestampIdx := findColumnIndex(schema, profile.ColumnTimestamp)
	timeNanosIdx := findColumnIndex(schema, profile.ColumnTimeNanos)
	valueIdx := findColumnIndex(schema, profile.ColumnValue)
	stacktraceIdx := findColumnIndex(schema, profile.ColumnStacktrace)

	labelColumns := make(map[string]int)
	for idx, field := range schema.Fields() {
		if strings.HasPrefix(field.Name, profile.ColumnLabelsPrefix) {
			labelColumns[strings.TrimPrefix(field.Name, profile.ColumnLabelsPrefix)] = idx
		}
	}

	for row := 0; row < int(record.NumRows()); row++ {
		labels := make(duckdb.Map, len(labelColumns))
		for name, colIdx := range labelColumns {
			if v := getStringValue(record, colIdx, row); v != "" {
				labels[name] = v
			}
		}

		st := buildStacktraceList(record, stacktraceIdx, row)

		err := appender.AppendRow(
			getStringValue(record, nameIdx, row),
			getStringValue(record, sampleTypeIdx, row),
			getStringValue(record, sampleUnitIdx, row),
			getStringValue(record, periodTypeIdx, row),
			getStringValue(record, periodUnitIdx, row),
			getInt64Value(record, periodIdx, row),
			getInt64Value(record, durationIdx, row),
			getInt64Value(record, timestampIdx, row),
			getInt64Value(record, timeNanosIdx, row),
			getInt64Value(record, valueIdx, row),
			labels,
			st,
		)
		if err != nil {
			level.Error(i.logger).Log("msg", "duckdb appender row failed", "row", row, "err", err)
			return fmt.Errorf("append row %d: %w", row, err)
		}
	}

	if err := appender.Flush(); err != nil {
		return fmt.Errorf("flush appender: %w", err)
	}
	return nil
}

// buildStacktraceList decodes the encoded location blobs in record's
// stacktrace LIST column at row and produces the slice form the duckdb
// Appender expects for a LIST(STRUCT(...)) column.
func buildStacktraceList(record arrow.RecordBatch, colIdx, row int) []map[string]any {
	if colIdx < 0 {
		return nil
	}
	col := record.Column(colIdx)
	listCol, ok := col.(*array.List)
	if !ok || listCol.IsNull(row) {
		return nil
	}
	start, end := listCol.ValueOffsets(row)
	values := listCol.ListValues()
	dictCol, ok := values.(*array.Dictionary)
	if !ok {
		return nil
	}
	bin, ok := dictCol.Dictionary().(*array.Binary)
	if !ok {
		return nil
	}

	out := make([]map[string]any, 0, end-start)
	for idx := int(start); idx < int(end); idx++ {
		if dictCol.IsNull(idx) {
			continue
		}
		raw := bin.Value(dictCol.GetValueIndex(idx))
		sym, _ := profile.DecodeSymbolizationInfo(raw)
		line := decodeLineInfo(raw)
		out = append(out, map[string]any{
			StFieldAddress:            sym.Addr,
			StFieldMappingStart:       sym.Mapping.StartAddr,
			StFieldMappingLimit:       sym.Mapping.EndAddr,
			StFieldMappingOffset:      sym.Mapping.Offset,
			StFieldMappingFile:        sym.Mapping.File,
			StFieldMappingBuildID:     string(sym.BuildID),
			StFieldLineNumber:         line.LineNumber,
			StFieldFunctionName:       line.FunctionName,
			StFieldFunctionSystemName: line.FunctionSystemName,
			StFieldFunctionFilename:   line.FunctionFilename,
			StFieldFunctionStartLine:  line.FunctionStartLine,
		})
	}
	return out
}

// lineInfo mirrors the bits of the encoded location format we care about.
type lineInfo struct {
	LineNumber         int64
	FunctionStartLine  int64
	FunctionName       string
	FunctionSystemName string
	FunctionFilename   string
}

// decodeLineInfo decodes the line/function portion of a varint-encoded
// location record produced by the symbolizer.
func decodeLineInfo(data []byte) lineInfo {
	info := lineInfo{}

	_, offset := varint.Uvarint(data)

	numLines, n := varint.Uvarint(data[offset:])
	offset += n

	hasMapping := data[offset] == 0x1
	offset++

	if hasMapping {
		// buildID
		length, n := varint.Uvarint(data[offset:])
		offset += n + int(length)
		// filename
		length, n = varint.Uvarint(data[offset:])
		offset += n + int(length)
		// memoryStart
		_, n = varint.Uvarint(data[offset:])
		offset += n
		// memoryLength
		_, n = varint.Uvarint(data[offset:])
		offset += n
		// mappingOffset
		_, n = varint.Uvarint(data[offset:])
		offset += n
	}

	if numLines > 0 {
		ln, n := varint.Uvarint(data[offset:])
		offset += n
		info.LineNumber = int64(ln)

		hasFunction := data[offset] == 0x1
		offset++
		if hasFunction {
			startLine, n := varint.Uvarint(data[offset:])
			offset += n
			info.FunctionStartLine = int64(startLine)

			length, n := varint.Uvarint(data[offset:])
			offset += n
			info.FunctionName = string(data[offset : offset+int(length)])
			offset += int(length)

			length, n = varint.Uvarint(data[offset:])
			offset += n
			info.FunctionSystemName = string(data[offset : offset+int(length)])
			offset += int(length)

			length, n = varint.Uvarint(data[offset:])
			offset += n
			info.FunctionFilename = string(data[offset : offset+int(length)])
		}
	}
	return info
}

func findColumnIndex(schema *arrow.Schema, name string) int {
	idx := schema.FieldIndices(name)
	if len(idx) == 0 {
		return -1
	}
	return idx[0]
}

func getStringValue(record arrow.RecordBatch, colIdx, row int) string {
	if colIdx < 0 {
		return ""
	}
	col := record.Column(colIdx)
	if col.IsNull(row) {
		return ""
	}
	switch c := col.(type) {
	case *array.Dictionary:
		switch d := c.Dictionary().(type) {
		case *array.Binary:
			return string(d.Value(c.GetValueIndex(row)))
		case *array.String:
			return d.Value(c.GetValueIndex(row))
		}
	case *array.String:
		return c.Value(row)
	case *array.Binary:
		return string(c.Value(row))
	}
	return ""
}

func getInt64Value(record arrow.RecordBatch, colIdx, row int) int64 {
	if colIdx < 0 {
		return 0
	}
	col := record.Column(colIdx)
	if col.IsNull(row) {
		return 0
	}
	switch c := col.(type) {
	case *array.Int64:
		return c.Value(row)
	case *array.Dictionary:
		if d, ok := c.Dictionary().(*array.Int64); ok {
			return d.Value(c.GetValueIndex(row))
		}
	}
	return 0
}
