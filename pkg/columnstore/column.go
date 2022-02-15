package columnstore

import (
	"sort"
)

type Appender interface {
	AppendAt(index int, value interface{}) error
	AppendValuesAt(index int, values interface{}) error
}

type Iterator interface {
	Next() bool
	IsNull() bool
	Value() interface{}
	Err() error
}

type Iterable interface {
	Iterator(maxIterations int) Iterator
}

func NewAppendOnceColumn(def ColumnDefinition) (Iterable, Appender, error) {
	if def.Dynamic {
		c := NewDynamicColumn(def)
		app, err := c.Appender()
		return c, app, err
	}

	c := NewStaticColumn(def)
	app, err := c.Appender()
	return c, app, err
}

type StaticColumn struct {
	def  ColumnDefinition
	data Encoding
}

func NewStaticColumn(def ColumnDefinition) *StaticColumn {
	return &StaticColumn{
		def:  def,
		data: def.Encoding.New(),
	}
}

func (c *StaticColumn) Appender() (Appender, error) {
	return c.def.Type.NewAppender(c.data), nil
}

func (c *StaticColumn) Iterator(maxIterations int) Iterator {
	return c.def.Type.NewIterator(c.data.Iterator(maxIterations))
}

type DynamicColumn struct {
	def ColumnDefinition

	data           map[string]Encoding
	dynamicColumns []string
}

func NewDynamicColumn(def ColumnDefinition) *DynamicColumn {
	return &DynamicColumn{
		def:  def,
		data: map[string]Encoding{},
	}
}

func (c *DynamicColumn) Appender() (Appender, error) {
	return &DynamicAppender{column: c}, nil
}

func (c *DynamicColumn) Iterator(maxIterations int) Iterator {
	its := make([]EncodingIterator, len(c.dynamicColumns))
	cols := make([]string, len(c.dynamicColumns))

	for i, d := range c.dynamicColumns {
		its[i] = c.data[d].Iterator(maxIterations)
		cols[i] = d
	}

	return &DynamicIterator{iterators: its, dynamicColumnNames: cols}
}

type DynamicAppender struct {
	column *DynamicColumn
}

type DynamicColumnValue struct {
	Name  string
	Value interface{}
}

func (a *DynamicAppender) AppendAt(index int, v interface{}) error {
	return a.DynamicAppendAt(index, v.([]DynamicColumnValue))
}

func (a *DynamicAppender) AppendValuesAt(index int, vs interface{}) error {
	dynamicValues := vs.([][]DynamicColumnValue)
	for i, v := range dynamicValues {
		if err := a.DynamicAppendAt(index+i, v); err != nil {
			return err
		}
	}
	return nil
}

func (a *DynamicAppender) DynamicAppendAt(index int, v []DynamicColumnValue) error {
	for _, d := range v {
		if _, ok := a.column.data[d.Name]; !ok {
			a.column.data[d.Name] = a.column.def.Encoding.New()
			a.column.dynamicColumns = append(a.column.dynamicColumns, d.Name)
			sort.Strings(a.column.dynamicColumns)
		}
	}

	for _, d := range v {
		a.column.def.Type.NewAppender(a.column.data[d.Name]).AppendAt(index, d.Value)
	}

	return nil
}

type DynamicIterator struct {
	iterators          []EncodingIterator
	dynamicColumnNames []string
}

func (i *DynamicIterator) Next() bool {
	for _, it := range i.iterators {
		if !it.Next() {
			return false
		}
	}

	return true
}

func (i *DynamicIterator) Value() interface{} {
	res := make([]DynamicColumnValue, 0, len(i.iterators))

	for j, it := range i.iterators {
		if it.IsNull() {
			continue
		}

		res = append(res, DynamicColumnValue{Name: i.dynamicColumnNames[j], Value: it.Value()})
	}

	return res
}

func (i *DynamicIterator) IsNull() bool {
	for _, it := range i.iterators {
		if !it.IsNull() {
			return false
		}
	}

	return true
}

func (i *DynamicIterator) Err() error {
	for _, it := range i.iterators {
		if err := it.Err(); err != nil {
			return err
		}
	}

	return nil
}
