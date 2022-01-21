package columnstore

import (
	"fmt"
	"sort"
)

type Appender interface {
	AppendAt(index int, values interface{}) error
}

type Iterator interface {
	Next() bool
	Value() interface{}
	Err() error
}

type Column interface {
	Appender() (Appender, error)
	Iterator() Iterator
	String() string
}

func NewColumn(def ColumnDefinition) Column {
	if def.Dynamic {
		return NewDynamicColumn(def)
	}

	return NewStaticColumn(def)
}

type StaticColumn struct {
	def   ColumnDefinition
	data  Encoding
	count int
}

func NewStaticColumn(def ColumnDefinition) *StaticColumn {
	return &StaticColumn{
		def:  def,
		data: def.Encoding.New(),
	}
}

type staticColumnAppender struct {
	column *StaticColumn
	app    Appender
}

func (a *staticColumnAppender) AppendAt(index int, values interface{}) error {
	err := a.app.AppendAt(index, values)
	if err != nil {
		return err
	}

	a.column.count++
	return nil
}

func (c *StaticColumn) Appender() (Appender, error) {
	return c.def.Type.NewAppender(&staticColumnAppender{column: c, app: c.data}), nil
}

func (c *StaticColumn) Iterator() Iterator {
	return c.def.Type.NewIterator(c.data.Iterator(c.count))
}

func (c *StaticColumn) String() string {
	res := c.def.String()

	it := c.Iterator()
	for it.Next() {
		res += "\n" + fmt.Sprint(it.Value())
	}
	if it.Err() != nil {
		res += "\nerror: " + it.Err().Error()
	}

	return res
}

type DynamicColumn struct {
	def ColumnDefinition

	data           map[string]Encoding
	dynamicColumns []string

	count int
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

func (c *DynamicColumn) Iterator() Iterator {
	its := make([]EncodingIterator, len(c.dynamicColumns))
	cols := make([]string, len(c.dynamicColumns))

	for i, d := range c.dynamicColumns {
		its[i] = c.data[d].Iterator(c.count)
		cols[i] = d
	}

	return &DynamicIterator{iterators: its, dynamicColumnNames: cols}
}

func (c *DynamicColumn) String() string {
	res := c.def.String()

	for _, d := range c.dynamicColumns {
		res += "\n" + "dynamicColumn: " + d

		it := c.data[d].Iterator(c.count)
		for it.Next() {
			res += "\n" + fmt.Sprint(it.Value())
		}
	}

	return res
}

type DynamicAppender struct {
	column *DynamicColumn
}

type DynamicColumnValue struct {
	Name  string
	Value interface{}
}

func (a *DynamicAppender) AppendAt(index int, v interface{}) error {
	err := a.DynamicAppendAt(index, v.([]DynamicColumnValue))
	if err != nil {
		return err
	}

	a.column.count++
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
		v := it.Value()
		res = append(res, DynamicColumnValue{Name: i.dynamicColumnNames[j], Value: v})
	}

	return res
}

func (i *DynamicIterator) Err() error {
	for _, it := range i.iterators {
		if err := it.Err(); err != nil {
			return err
		}
	}

	return nil
}
