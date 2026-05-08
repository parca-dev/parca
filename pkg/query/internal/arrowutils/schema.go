package arrowutils

import (
	"fmt"
	"sort"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// EnsureSameSchema ensures that all the records have the same schema. In cases
// where the schema is not equal, virtual null columns are inserted in the
// records with the missing column. When we have static schemas in the execution
// engine, steps like these should be unnecessary.
func EnsureSameSchema(records []arrow.Record) ([]arrow.Record, error) {
	if len(records) < 2 {
		return records, nil
	}

	lastSchema := records[0].Schema()
	needSchemaRecalculation := false
	for i := range records {
		if !records[i].Schema().Equal(lastSchema) {
			needSchemaRecalculation = true
			break
		}
	}
	if !needSchemaRecalculation {
		return records, nil
	}

	columns := make(map[string]arrow.Field)
	for _, r := range records {
		for j := 0; j < r.Schema().NumFields(); j++ {
			field := r.Schema().Field(j)
			if _, ok := columns[field.Name]; !ok {
				columns[field.Name] = field
			}
		}
	}

	columnNames := make([]string, 0, len(columns))
	for name := range columns {
		columnNames = append(columnNames, name)
	}
	sort.Strings(columnNames)

	mergedFields := make([]arrow.Field, 0, len(columnNames))
	for _, name := range columnNames {
		mergedFields = append(mergedFields, columns[name])
	}
	mergedSchema := arrow.NewSchema(mergedFields, nil)

	mergedRecords := make([]arrow.Record, len(records))
	var replacedRecords []arrow.Record

	for i := range records {
		recordSchema := records[i].Schema()
		if mergedSchema.Equal(recordSchema) {
			mergedRecords[i] = records[i]
			continue
		}

		mergedColumns := make([]arrow.Array, 0, len(mergedFields))
		recordNumRows := records[i].NumRows()
		for j := 0; j < mergedSchema.NumFields(); j++ {
			field := mergedSchema.Field(j)
			if otherFields := recordSchema.FieldIndices(field.Name); otherFields != nil {
				if len(otherFields) > 1 {
					fieldsFound, _ := recordSchema.FieldsByName(field.Name)
					return nil, fmt.Errorf(
						"found multiple fields %v for name %s",
						fieldsFound,
						field.Name,
					)
				}
				mergedColumns = append(mergedColumns, records[i].Column(otherFields[0]))
			} else {
				// Note that this VirtualNullArray will be read from, but the
				// merged output will be a physical null array, so there is no
				// virtual->physical conversion necessary before we return data.
				mergedColumns = append(mergedColumns, MakeVirtualNullArray(field.Type, int(recordNumRows)))
			}
		}

		replacedRecords = append(replacedRecords, records[i])
		mergedRecords[i] = array.NewRecord(mergedSchema, mergedColumns, recordNumRows)
	}

	for _, r := range replacedRecords {
		r.Release()
	}

	return mergedRecords, nil
}
