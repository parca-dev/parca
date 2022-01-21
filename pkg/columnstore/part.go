package columnstore

type Row struct {
	Values []interface{}
}

type Part struct {
	schema      Schema
	columns     []Column
	Cardinality int
}

func NewPart(schema Schema, rows []Row) (*Part, error) {
	p := &Part{
		schema:      schema,
		Cardinality: len(rows),
	}
	p.columns = make([]Column, len(schema.Columns))

	for i, c := range schema.Columns {
		p.columns[i] = NewColumn(c)
		app, err := p.columns[i].Appender()
		if err != nil {
			return nil, err
		}

		for j := range rows {
			err := app.AppendAt(j, rows[j].Values[i])
			if err != nil {
				return nil, err
			}
		}
	}

	return p, nil
}

func (p *Part) Iterator() *PartIterator {
	columnIterators := make([]Iterator, len(p.columns))

	for i, c := range p.columns {
		columnIterators[i] = c.Iterator(p.Cardinality)
	}

	return &PartIterator{
		columnIterators: columnIterators,
	}
}

type PartIterator struct {
	columnIterators []Iterator
}

func (pi *PartIterator) Next() bool {
	for _, ci := range pi.columnIterators {
		if !ci.Next() {
			return false
		}
	}

	return true
}

func (pi *PartIterator) Values() []interface{} {
	res := make([]interface{}, len(pi.columnIterators))

	for i, ci := range pi.columnIterators {
		res[i] = ci.Value()
	}

	return res
}

func (pi *PartIterator) Err() error {
	for _, ci := range pi.columnIterators {
		if err := ci.Err(); err != nil {
			return err
		}
	}

	return nil
}
