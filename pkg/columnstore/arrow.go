package columnstore

import (
	"fmt"

	"github.com/apache/arrow/go/arrow/array"
)

type StaticArrowColumn struct {
	Name string
	Data array.Interface
}

func NewStaticArrowColumn(name string, array array.Interface) *StaticArrowColumn {
	return &StaticArrowColumn{
		Name: name,
		Data: array,
	}
}

func (c *StaticArrowColumn) String() string {
	res := c.Name + ":\n"
	res += fmt.Sprint(c.Data) + "\n"
	return res
}

func (c *StaticArrowColumn) Release() {
	c.Data.Release()
}

type DynamicArrowColumn struct {
	Name               string
	DynamicColumnNames []string
	data               map[string]*StaticArrowColumn
}

func NewDynamicArrowColumn(name string, dynamicColumnNames []string, data map[string]*StaticArrowColumn) *DynamicArrowColumn {
	return &DynamicArrowColumn{
		Name:               name,
		data:               data,
		DynamicColumnNames: dynamicColumnNames,
	}
}

func (c *DynamicArrowColumn) Columns() []*StaticArrowColumn {
	columns := make([]*StaticArrowColumn, len(c.data))
	i := 0
	for _, columnName := range c.DynamicColumnNames {
		columns[i] = c.data[columnName]
		i++
	}
	return columns
}

func (c *DynamicArrowColumn) DynamicColumn(name string) *StaticArrowColumn {
	return c.data[name]
}

func (c *DynamicArrowColumn) Release() {
	for _, column := range c.data {
		column.Release()
	}
}

func (c *DynamicArrowColumn) String() string {
	res := c.Name + "::\n"
	for _, columnName := range c.DynamicColumnNames {
		res += c.data[columnName].String() + "\n"
	}
	return res
}

type ArrowColumn interface {
	Release()
	String() string
}

type ArrowRecord struct {
	columns []ArrowColumn
}

func NewArrowRecord(columns []ArrowColumn) *ArrowRecord {
	return &ArrowRecord{columns: columns}
}

func (r *ArrowRecord) String() string {
	res := ""
	for _, column := range r.columns {
		res += column.String() + "\n"
	}
	return res
}

func (r *ArrowRecord) Release() {
	for _, column := range r.columns {
		column.Release()
	}
}
