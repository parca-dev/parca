// Copyright 2021 The Parca Authors
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
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/pprof/profile"
)

var _ ProfileMetaStore = &sqlMetaStore{}

type sqlMetaStore struct {
	db *sql.DB
}

func (s *sqlMetaStore) migrate() error {
	tables := []string{
		"PRAGMA foreign_keys = ON",
		`CREATE TABLE "mappings" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"mapping_id" 		UINT64,
			"start"           	UINT64,
			"limit"          	UINT64,
			"offset"          	UINT64,
			"file"           	TEXT,
			"build_id"         	TEXT,
			"has_functions"    	BOOLEAN,
			"has_filenames"    	BOOLEAN,
			"has_line_numbers"  BOOLEAN,
			"has_inline_frames" BOOLEAN,
			"size"				UINT64,
			"build_id_or_file"	TEXT
		);`,
		`CREATE INDEX idx_mapping_id ON mappings (mapping_id);`,
		`CREATE INDEX idx_mapping_key ON mappings (size, offset, build_id_or_file);`,
		`CREATE TABLE "functions" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"function_id"	UINT64,
			"name"       	TEXT,
			"system_name" 	TEXT,
			"filename"   	TEXT,
			"start_line"  	INT64
		);`,
		`CREATE INDEX idx_function_id ON functions (function_id);`,
		`CREATE INDEX idx_function_key ON functions (start_line, name, system_name, filename);`,
		`CREATE TABLE "lines" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"function_id"	INTEGER NOT NULL,
			"line" 		  	INT64,
			FOREIGN KEY (function_id) REFERENCES functions (id)
		);`,
		`CREATE TABLE "locations" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"location_id"			UINT64,
			"mapping_id"  			INTEGER,
			"address"  				UINT64,
			"is_folded" 			BOOLEAN,
			"normalized_address"	UINT64,
			"lines"					TEXT,
			FOREIGN KEY (mapping_id) REFERENCES mappings (id)
		);`,
		`CREATE INDEX idx_location_id ON locations (location_id);`,
		`CREATE INDEX idx_location_key ON locations (normalized_address, mapping_id, is_folded, lines);`,
		`CREATE TABLE "location_lines" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"location_id" 	INTEGER NOT NULL,
			"line_id" 		INTEGER NOT NULL,
			FOREIGN KEY(location_id) REFERENCES locations (id),
			FOREIGN KEY(line_id) REFERENCES lines (id)
		);`,
	}
	// TODO(kakkoyun): Additional table between location and mapping? - mapInfo from pprof

	for _, t := range tables {
		statement, err := s.db.Prepare(t)
		if err != nil {
			return err
		}

		if _, err := statement.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlMetaStore) GetLocationByKey(k LocationKey) (*profile.Location, error) {
	var (
		l           profile.Location
		mappingPKey *int
		err         error
	)
	if k.MappingID > 0 {
		err = s.db.QueryRow(
			`SELECT "location_id", "address", "is_folded", "mapping_id"
					FROM "locations" l
					JOIN "mappings" m ON l.mapping_id = m.id
					WHERE l.normalized_address=? AND l.is_folded=? AND l.lines=? AND m.id=? `,
			k.Addr, k.IsFolded, k.Lines, k.MappingID,
		).Scan(&l.ID, &l.Address, &l.IsFolded, &mappingPKey)
	} else {
		err = s.db.QueryRow(
			`SELECT "location_id", "address", "is_folded"
					FROM "locations"
					WHERE normalized_address=? AND mapping_id IS NULL AND is_folded=? AND lines=?`,
			k.Addr, k.IsFolded, k.Lines,
		).Scan(&l.ID, &l.Address, &l.IsFolded)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}

	if mappingPKey != nil {
		mapping, err := s.getMappingByPrimaryKey(*mappingPKey)
		if err != nil {
			return nil, err
		}
		l.Mapping = mapping
	}

	lines, err := s.getLocationLines(l.ID)
	if err != nil {
		return nil, err
	}
	l.Line = lines

	return &l, nil
}

func (s *sqlMetaStore) GetLocationByID(id uint64) (*profile.Location, error) {
	var (
		l           profile.Location
		mappingPKey *int
	)
	err := s.db.QueryRow(
		`SELECT "location_id", "address", "is_folded", "mapping_id"
				FROM "locations"
				WHERE location_id=?`, id,
	).Scan(&l.ID, &l.Address, &l.IsFolded, &mappingPKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}

	if mappingPKey != nil {
		mapping, err := s.getMappingByPrimaryKey(*mappingPKey)
		if err != nil {
			return nil, err
		}
		l.Mapping = mapping
	}

	lines, err := s.getLocationLines(l.ID)
	if err != nil {
		return nil, err
	}
	l.Line = lines

	return &l, nil
}

func (s *sqlMetaStore) CreateLocation(l *profile.Location) error {
	k := MakeLocationKey(l)
	var res sql.Result
	if l.Mapping != nil {
		stmt, err := s.db.Prepare(
			`INSERT INTO "locations" (location_id, address, is_folded, mapping_id, normalized_address, lines)
					values(?,?,?,?,?,?)`,
		)
		if err != nil {
			return err
		}
		defer stmt.Close()

		var mappingID int
		err = s.db.QueryRow(`SELECT "id" FROM "mappings" WHERE mapping_id=?`, l.Mapping.ID).Scan(&mappingID)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrMappingNotFound
			}
			return err
		}

		res, err = stmt.Exec(l.ID, l.Address, l.IsFolded, mappingID, k.Addr, k.Lines)
		if err != nil {
			return err
		}
	} else {
		stmt, err := s.db.Prepare(
			`INSERT INTO "locations" (
                         location_id, address, is_folded, normalized_address, lines
                         ) values(?,?,?,?,?)`,
		)
		if err != nil {
			return err
		}
		defer stmt.Close()

		res, err = stmt.Exec(l.ID, l.Address, l.IsFolded, k.Addr, k.Lines)
		if err != nil {
			return err
		}
	}

	locID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	if err := s.createLines(l.Line, locID); err != nil {
		return err
	}

	return nil
}

func (s *sqlMetaStore) createLines(lines []profile.Line, locID int64) error {
	if len(lines) > 0 {
		q := `INSERT INTO "lines" (line, function_id) VALUES `
		for i, ln := range lines {
			functionID, err := s.getOrCreateFunction(ln.Function)
			if err != nil {
				return err
			}
			q += fmt.Sprintf(`(%s, %s)`,
				strconv.FormatInt(ln.Line, 10),
				strconv.FormatInt(functionID, 10))
			if i != len(lines)-1 {
				q += ", "
			}
		}
		q += ";"
		stmt, err := s.db.Prepare(q)
		if err != nil {
			return err
		}
		defer stmt.Close()

		res, err := stmt.Exec()
		if err != nil {
			return err
		}

		// Assuming ids are auto-incremented, we populate locations_lines going backwards.
		rf, err := res.RowsAffected()
		if err != nil {
			return err
		}
		lastLineID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		q = `INSERT INTO "location_lines" (line_id, location_id) VALUES `
		for i := int64(0); i < rf; i++ {
			q += fmt.Sprintf(`(%s, %s)`,
				strconv.FormatInt(lastLineID-i, 10),
				strconv.FormatInt(locID, 10))
			if i != rf-1 {
				q += ", "
			}
		}
		q += ";"
		stmt, err = s.db.Prepare(q)
		if err != nil {
			return err
		}
		defer stmt.Close()

		res, err = stmt.Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlMetaStore) UpdateLocation(l *profile.Location) error {
	k := MakeLocationKey(l)
	var res sql.Result
	if l.Mapping != nil {
		stmt, err := s.db.Prepare(
			`UPDATE "locations" SET address=?, is_folded=?, mapping_id=?, normalized_address=?, lines=? WHERE location_id=?`,
		)

		if err != nil {
			return err
		}
		defer stmt.Close()

		var mappingID int
		err = s.db.QueryRow(`SELECT "id" FROM "mappings" WHERE mapping_id=?`, l.Mapping.ID).Scan(&mappingID)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrMappingNotFound
			}
			return err
		}

		res, err = stmt.Exec(l.Address, l.IsFolded, mappingID, k.Addr, k.Lines, l.ID)
		if err != nil {
			return err
		}
	} else {
		stmt, err := s.db.Prepare(
			`UPDATE "locations" SET address=?, is_folded=? WHERE location_id=?`,
		)

		if err != nil {
			return err
		}
		defer stmt.Close()

		res, err = stmt.Exec(l.Address, l.IsFolded, l.ID)
		if err != nil {
			return err
		}
	}

	locID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	if err := s.createLines(l.Line, locID); err != nil {
		return err
	}

	return nil
}

func (s *sqlMetaStore) GetLocations() ([]*profile.Location, error) {
	rows, err := s.db.Query(
		`SELECT l."location_id", l."address", l."is_folded", m."mapping_id",
       					m."start", m."limit", m."offset", m."file", m."build_id",
       					m."has_functions", m."has_filenames", m."has_line_numbers", m."has_inline_frames"
				FROM "locations" l
				LEFT JOIN "mappings" m ON l.mapping_id = m.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %w", err)
	}
	defer rows.Close()

	locs := []*profile.Location{}
	for rows.Next() {
		l := &profile.Location{}
		var (
			mappingID       *uint64
			start           *uint64
			limit           *uint64
			offset          *uint64
			file            *string
			buildID         *string
			hasFunctions    *bool
			hasFilenames    *bool
			hasLineNumbers  *bool
			hasInlineFrames *bool
		)
		err := rows.Scan(
			&l.ID, &l.Address, &l.IsFolded,
			&mappingID, &start, &limit, &offset, &file, &buildID,
			&hasFunctions, &hasFilenames, &hasLineNumbers, &hasInlineFrames,
		)
		if err != nil {
			return nil, err
		}
		if mappingID != nil {
			l.Mapping = &profile.Mapping{
				ID:              *mappingID,
				Start:           *start,
				Limit:           *limit,
				Offset:          *offset,
				File:            *file,
				BuildID:         *buildID,
				HasFunctions:    *hasFunctions,
				HasFilenames:    *hasFilenames,
				HasLineNumbers:  *hasLineNumbers,
				HasInlineFrames: *hasInlineFrames,
			}
		}

		lines, err := s.getLocationLines(l.ID)
		if err != nil {
			return nil, err
		}
		l.Line = lines

		locs = append(locs, l)
	}
	return locs, nil
}

func (s *sqlMetaStore) GetUnsymbolizedLocations() ([]*profile.Location, error) {
	rows, err := s.db.Query(
		`SELECT l."location_id", l."address", l."is_folded", m."mapping_id",
       					m."start", m."limit", m."offset", m."file", m."build_id",
       					m."has_functions", m."has_filenames", m."has_line_numbers", m."has_inline_frames"
				FROM "locations" l
				LEFT JOIN "mappings" m ON l.mapping_id = m.id
				LEFT JOIN "location_lines" ll ON l."id" = ll."location_id"
				WHERE ll."location_id" IS NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locs := []*profile.Location{}
	for rows.Next() {
		l := &profile.Location{}
		var (
			mappingID       *uint64
			start           *uint64
			limit           *uint64
			offset          *uint64
			file            *string
			buildID         *string
			hasFunctions    *bool
			hasFilenames    *bool
			hasLineNumbers  *bool
			hasInlineFrames *bool
		)
		err := rows.Scan(
			&l.ID, &l.Address, &l.IsFolded,
			&mappingID, &start, &limit, &offset, &file, &buildID,
			&hasFunctions, &hasFilenames, &hasLineNumbers, &hasInlineFrames,
		)
		if err != nil {
			return nil, err
		}
		if mappingID != nil {
			l.Mapping = &profile.Mapping{
				ID:              *mappingID,
				Start:           *start,
				Limit:           *limit,
				Offset:          *offset,
				File:            *file,
				BuildID:         *buildID,
				HasFunctions:    *hasFunctions,
				HasFilenames:    *hasFilenames,
				HasLineNumbers:  *hasLineNumbers,
				HasInlineFrames: *hasInlineFrames,
			}
		}

		locs = append(locs, l)
	}
	return locs, nil
}

func (s *sqlMetaStore) GetFunctionByKey(k FunctionKey) (*profile.Function, error) {
	var fn profile.Function
	err := s.db.QueryRow(
		`SELECT "function_id", "name", "system_name", "filename", "start_line"
				FROM "functions"
				WHERE start_line=? AND name=? AND system_name=? AND filename=?`,
		k.StartLine, k.Name, k.SystemName, k.FileName,
	).Scan(&fn.ID, &fn.Name, &fn.SystemName, &fn.Filename, &fn.StartLine)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFunctionNotFound
		}
		return nil, err
	}
	return &fn, nil
}

func (s *sqlMetaStore) CreateFunction(f *profile.Function) error {
	_, err := s.createFunction(f)
	return err
}

func (s *sqlMetaStore) createFunction(f *profile.Function) (int64, error) {
	stmt, err := s.db.Prepare(
		`INSERT INTO "functions" (
                         function_id, name, system_name, filename, start_line
                         ) values(?,?,?,?,?)`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(f.ID, f.Name, f.SystemName, f.Filename, f.StartLine)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *sqlMetaStore) GetFunctions() ([]*profile.Function, error) {
	rows, err := s.db.Query(`SELECT "function_id", "name", "system_name", "filename", "start_line" FROM "functions"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	funcs := []*profile.Function{}
	for rows.Next() {
		f := profile.Function{}
		err := rows.Scan(&f.ID, &f.Name, &f.SystemName, &f.Filename, &f.StartLine)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, &f)
	}

	return funcs, nil
}

func (s *sqlMetaStore) GetMappingByKey(k MappingKey) (*profile.Mapping, error) {
	var m profile.Mapping
	err := s.db.QueryRow(
		`SELECT "mapping_id", "start", "limit", "offset", "file", "build_id",
				"has_functions", "has_filenames", "has_line_numbers", "has_inline_frames"
				FROM "mappings"
				WHERE size=? AND offset=? AND build_id_or_file=?`,
		k.Size, k.Offset, k.BuildIDOrFile,
	).Scan(
		&m.ID, &m.Start, &m.Limit, &m.Offset, &m.File, &m.BuildID,
		&m.HasFunctions, &m.HasFilenames, &m.HasLineNumbers, &m.HasInlineFrames,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMappingNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *sqlMetaStore) CreateMapping(m *profile.Mapping) error {
	stmt, err := s.db.Prepare(
		`INSERT INTO "mappings" (
                        "mapping_id", "start", "limit", "offset", "file", "build_id",
                        "has_functions", "has_filenames", "has_line_numbers", "has_inline_frames",
                        "size", "build_id_or_file"
                        ) values(?,?,?,?,?,?,?,?,?,?,?,?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	k := MakeMappingKey(m)
	_, err = stmt.Exec(
		m.ID, m.Start, m.Limit, m.Offset, m.File, m.BuildID,
		m.HasFunctions, m.HasFilenames, m.HasLineNumbers, m.HasInlineFrames,
		k.Size, k.BuildIDOrFile,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *sqlMetaStore) Close() error {
	return s.db.Close()
}

func (s *sqlMetaStore) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (s *sqlMetaStore) getMappingByPrimaryKey(pkey int) (*profile.Mapping, error) {
	var m profile.Mapping
	err := s.db.QueryRow(
		`SELECT "mapping_id", "start", "limit", "offset", "file", "build_id",
				"has_functions", "has_filenames", "has_line_numbers", "has_inline_frames"
				FROM "mappings" WHERE id=?`, pkey,
	).Scan(
		&m.ID, &m.Start, &m.Limit, &m.Offset, &m.File, &m.BuildID,
		&m.HasFunctions, &m.HasFilenames, &m.HasLineNumbers, &m.HasInlineFrames,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMappingNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *sqlMetaStore) getLocationLines(locationID uint64) ([]profile.Line, error) {
	var lines []profile.Line
	rows, err := s.db.Query(
		`SELECT ln."line", fn."function_id", fn."name", fn."system_name", fn."filename", fn."start_line"
				FROM "location_lines" ll
				LEFT JOIN "locations" loc ON ll."location_id" = loc."id"
				LEFT JOIN "lines" ln ON ll."line_id" = ln."id"
				LEFT JOIN "functions" fn ON ln."function_id" = fn."id"
				WHERE loc."location_id"=?`, locationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ln := profile.Line{}
		fn := profile.Function{}
		err := rows.Scan(&ln.Line, &fn.ID, &fn.Name, &fn.SystemName, &fn.Filename, &fn.StartLine)
		if err != nil {
			return nil, err
		}
		ln.Function = &fn
		lines = append(lines, ln)
	}

	// To make tests stable.
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})

	return lines, nil
}

func (s *sqlMetaStore) getOrCreateFunction(f *profile.Function) (int64, error) {
	var functionID int64
	err := s.db.QueryRow(`SELECT "id" FROM "functions" WHERE function_id=?`, f.ID).Scan(&functionID)
	if err != nil {
		if err == sql.ErrNoRows {
			functionID, err = s.createFunction(f)
			if err != nil {
				return 0, err
			}
			return functionID, nil
		}
		return 0, err
	}
	return functionID, nil
}
