package columnstore

type Row struct {
	Values []interface{}
}

type Part struct {
	schema  Schema
	columns []Column
}

func NewPart(schema Schema, rows []Row) (*Part, error) {
	p := &Part{schema: schema}
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

func (p *Part) String() string {
	res := ""
	for _, c := range p.columns {
		res += c.String() + "\n\n"
	}

	return res
}
