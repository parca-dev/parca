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
	"strconv"
	"time"
	"unsafe"

	"github.com/cenkalti/backoff"
	"github.com/google/pprof/profile"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ ProfileMetaStore = &sqlMetaStore{}

type sqlMetaStore struct {
	db     *sql.DB
	cache  *metaStoreCache
	tracer trace.Tracer
}

func (s *sqlMetaStore) migrate() error {
	// Most of the tables have started their lives as representation of pprof data types.
	// Find detailed information in https://github.com/google/pprof/blob/master/proto/README.md
	tables := []string{
		"PRAGMA foreign_keys = ON",
		`CREATE TABLE "mappings" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"start"           	INT64,
			"limit"          	INT64,
			"offset"          	INT64,
			"file"           	TEXT,
			"build_id"         	TEXT,
			"has_functions"    	BOOLEAN,
			"has_filenames"    	BOOLEAN,
			"has_line_numbers"  BOOLEAN,
			"has_inline_frames" BOOLEAN,
			"size"				INT64,
			"build_id_or_file"	TEXT,
			UNIQUE (size, offset, build_id_or_file)
		);`,
		`CREATE INDEX idx_mapping_key ON mappings (size, offset, build_id_or_file);`,
		`CREATE TABLE "functions" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name"       	TEXT,
			"system_name" 	TEXT,
			"filename"   	TEXT,
			"start_line"  	INT64,
			UNIQUE (name, system_name, filename, start_line)
		);`,
		`CREATE INDEX idx_function_key ON functions (start_line, name, system_name, filename);`,
		`CREATE TABLE "lines" (
			"location_id" INTEGER NOT NULL,
			"function_id" INTEGER NOT NULL,
			"line" 		  INT64,
			FOREIGN KEY (function_id) REFERENCES functions (id),
			FOREIGN KEY (location_id) REFERENCES locations (id),
			UNIQUE (location_id, function_id, line)
		);`,
		`CREATE INDEX idx_line_location ON lines (location_id);`,
		`CREATE TABLE "locations" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"mapping_id"  			INTEGER,
			"address"  				INT64,
			"is_folded" 			BOOLEAN,
			"normalized_address"	INT64,
			"lines"					TEXT,
			FOREIGN KEY (mapping_id) REFERENCES mappings (id),
			UNIQUE (mapping_id, is_folded, normalized_address, lines)
		);`,
		`CREATE INDEX idx_location_key ON locations (normalized_address, mapping_id, is_folded, lines);`,
	}

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

func (s *sqlMetaStore) GetLocationByKey(ctx context.Context, k LocationKey) (*profile.Location, error) {
	res := profile.Location{}

	l, found, err := s.cache.getLocationByKey(ctx, k)
	if err != nil {
		return nil, fmt.Errorf("get location by key from cache: %w", err)
	}
	if !found {
		var (
			id      int64
			address int64
			err     error
		)
		if k.MappingID > 0 {
			err = s.db.QueryRowContext(ctx,
				`SELECT "id", "address"
					FROM "locations" l
					WHERE normalized_address=? 
					  AND is_folded=? 
					  AND lines=? 
					  AND mapping_id=? `,
				int64(k.NormalizedAddress), k.IsFolded, k.Lines, int64(k.MappingID),
			).Scan(&id, &address)
		} else {
			err = s.db.QueryRowContext(ctx,
				`SELECT "id", "address"
					FROM "locations" l
					WHERE normalized_address=? 
					  AND mapping_id IS NULL 
					  AND is_folded=? 
					  AND lines=?`,
				int64(k.NormalizedAddress), k.IsFolded, k.Lines,
			).Scan(&id, &address)
		}
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrLocationNotFound
			}
			return nil, err
		}
		l.ID = uint64(id)
		l.Address = uint64(address)
		l.LocationKey = k

		err = s.cache.setLocationByKey(ctx, k, l)
		if err != nil {
			return nil, fmt.Errorf("set location by key in cache: %w", err)
		}
	}
	res.ID = l.ID
	res.Address = l.Address
	res.IsFolded = l.IsFolded

	if k.MappingID > 0 {
		mapping, err := s.getMappingByID(ctx, int64(k.MappingID))
		if err != nil {
			return nil, fmt.Errorf("get mapping by ID: %w", err)
		}
		res.Mapping = mapping
	}

	linesByLocation, functionIDs, err := s.getLinesByLocationIDs(ctx, l.ID)
	if err != nil {
		return nil, fmt.Errorf("get lines by location ID: %w", err)
	}

	functions, err := s.getFunctionsByIDs(ctx, functionIDs...)
	if err != nil {
		return nil, fmt.Errorf("get functions by IDs: %w", err)
	}

	for _, line := range linesByLocation[l.ID] {
		res.Line = append(res.Line, profile.Line{
			Line:     line.Line,
			Function: functions[line.FunctionID],
		})
	}

	return &res, nil
}

func (s *sqlMetaStore) GetLocationsByIDs(ctx context.Context, ids ...uint64) (
	map[uint64]*profile.Location,
	error,
) {
	ctx, span := s.tracer.Start(ctx, "GetLocationsByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("location-ids-number", len(ids)))

	locs := map[uint64]Location{}

	mappingIDs := []uint64{}
	mappingIDsSeen := map[uint64]struct{}{}

	remainingIds := []uint64{}
	maxRemainingID := uint64(0)
	for _, id := range ids {
		l, found, err := s.cache.getLocationByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get location by ID: %w", err)
		}
		if found {
			locs[l.ID] = l
			if l.MappingID > 0 {
				if _, seen := mappingIDsSeen[l.MappingID]; !seen {
					mappingIDs = append(mappingIDs, l.MappingID)
					mappingIDsSeen[l.MappingID] = struct{}{}
				}
			}
			continue
		}
		if maxRemainingID < id {
			maxRemainingID = id
		}
		remainingIds = append(remainingIds, id)
	}

	if len(remainingIds) > 0 {
		dbctx, dbspan := s.tracer.Start(ctx, "GetLocationsByIDs-SQL-query")
		rows, err := s.db.QueryContext(dbctx, buildLocationsByIDsQuery(maxRemainingID, remainingIds))
		dbspan.End()
		if err != nil {
			return nil, fmt.Errorf("execute SQL query: %w", err)
		}

		defer rows.Close()

		for rows.Next() {
			var (
				l                                 Location = Location{}
				locID, address, normalizedAddress int64
				mappingID                         *int64
			)

			err := rows.Scan(&locID, &mappingID, &address, &l.IsFolded, &normalizedAddress, &l.Lines)
			if err != nil {
				return nil, fmt.Errorf("scan row: %w", err)
			}
			l.ID = uint64(locID)
			if mappingID != nil {
				l.MappingID = uint64(*mappingID)
			}
			l.Address = uint64(address)
			l.NormalizedAddress = uint64(normalizedAddress)
			if _, found := locs[l.ID]; !found {
				err := s.cache.setLocationByID(ctx, l)
				if err != nil {
					return nil, fmt.Errorf("set location cache by ID: %w", err)
				}
				locs[l.ID] = l
				if l.MappingID > 0 {
					if _, seen := mappingIDsSeen[l.MappingID]; !seen {
						mappingIDs = append(mappingIDs, l.MappingID)
						mappingIDsSeen[l.MappingID] = struct{}{}
					}
				}
			}
		}
		err = rows.Err()
		if err != nil {
			return nil, fmt.Errorf("iterate over SQL rows: %w", err)
		}
	}

	mappings, err := s.GetMappingsByIDs(ctx, mappingIDs...)
	if err != nil {
		return nil, fmt.Errorf("get mappings by IDs: %w", err)
	}

	linesByLocation, functionIDs, err := s.getLinesByLocationIDs(ctx, ids...)
	if err != nil {
		return nil, fmt.Errorf("get lines by location IDs: %w", err)
	}

	functions, err := s.getFunctionsByIDs(ctx, functionIDs...)
	if err != nil {
		return nil, fmt.Errorf("get functions by ids: %w", err)
	}

	res := make(map[uint64]*profile.Location, len(locs))
	for locationID, loc := range locs {
		location := &profile.Location{
			ID:       loc.ID,
			Address:  loc.Address,
			IsFolded: loc.IsFolded,
		}
		location.Mapping = mappings[loc.MappingID]
		locationLines := linesByLocation[locationID]
		if len(locationLines) > 0 {
			lines := make([]profile.Line, 0, len(locationLines))
			for _, line := range locationLines {
				function, found := functions[line.FunctionID]
				if found {
					lines = append(lines, profile.Line{
						Line:     line.Line,
						Function: function,
					})
				}
			}
			location.Line = lines
		}
		res[locationID] = location
	}

	return res, nil
}

const (
	locsByIDsQueryStart = `SELECT "id", "mapping_id", "address", "is_folded", "normalized_address", "lines"
				FROM "locations"
				WHERE id IN (`
)

func buildLocationsByIDsQuery(max uint64, ids []uint64) string {
	maxLen := len(strconv.FormatUint(max, 10))

	query := make([]byte, 0,
		// The max value is known, and invididual string can be larger than it.
		len(ids)*maxLen+
			// Add the start of the query.
			len(locsByIDsQueryStart)+
			// len(ids)-1 commas, and a closing bracket is len(ids).
			len(ids),
	)
	query = append(query, locsByIDsQueryStart...)

	for i := range ids {
		if i > 0 {
			query = append(query, comma)
		}
		query = strconv.AppendUint(query, ids[i], 10)
	}

	query = append(query, closingBracket)
	return unsafeString(query)
}

func (s *sqlMetaStore) GetMappingsByIDs(ctx context.Context, ids ...uint64) (map[uint64]*profile.Mapping, error) {
	ctx, span := s.tracer.Start(ctx, "GetMappingsByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("mapping-ids-length", len(ids)))

	res := make(map[uint64]*profile.Mapping, len(ids))

	sIds := ""
	for i, id := range ids {
		if i > 0 {
			sIds += ","
		}
		sIds += strconv.FormatInt(int64(id), 10)
	}

	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(
			`SELECT "id", "start", "limit", "offset", "file", "build_id",
				"has_functions", "has_filenames", "has_line_numbers", "has_inline_frames"
				FROM "mappings" WHERE id IN (%s)`, sIds),
	)
	if err != nil {
		return nil, fmt.Errorf("execute SQL query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			m                        *profile.Mapping = &profile.Mapping{}
			id, start, limit, offset int64
		)
		err := rows.Scan(
			&id, &start, &limit, &offset, &m.File, &m.BuildID,
			&m.HasFunctions, &m.HasFilenames, &m.HasLineNumbers, &m.HasInlineFrames,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrMappingNotFound
			}
			return nil, fmt.Errorf("scan row: %w", err)
		}
		m.ID = uint64(id)
		m.Start = uint64(start)
		m.Limit = uint64(limit)
		m.Offset = uint64(offset)

		if _, found := res[m.ID]; !found {
			res[m.ID] = m
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate over SQL rows: %w", err)
	}

	return res, nil
}

type locationLine struct {
	Line       int64
	FunctionID uint64
}

func (s *sqlMetaStore) getLinesByLocationIDs(ctx context.Context, ids ...uint64) (map[uint64][]locationLine, []uint64, error) {
	ctx, span := s.tracer.Start(ctx, "getLinesByLocationIDs")
	defer span.End()

	functionIDs := []uint64{}
	functionIDsSeen := map[uint64]struct{}{}

	res := make(map[uint64][]locationLine, len(ids))
	remainingIds := []uint64{}
	maxRemainingId := uint64(0)
	for _, id := range ids {
		ll, found, err := s.cache.getLocationLinesByID(ctx, id)
		if err != nil {
			return res, functionIDs, fmt.Errorf("get location lines by ID from cache: %w", err)
		}
		if found {
			for _, l := range ll {
				if _, seen := functionIDsSeen[l.FunctionID]; !seen {
					functionIDs = append(functionIDs, l.FunctionID)
					functionIDsSeen[l.FunctionID] = struct{}{}
				}
			}
			res[id] = ll
			continue
		}
		if maxRemainingId < id {
			maxRemainingId = id
		}
		remainingIds = append(remainingIds, id)
	}
	ids = remainingIds

	if len(ids) == 0 {
		return res, functionIDs, nil
	}

	rows, err := s.db.QueryContext(ctx, buildLinesByLocationIDsQuery(maxRemainingId, ids))
	if err != nil {
		return nil, nil, fmt.Errorf("execute SQL query: %w", err)
	}

	defer rows.Close()

	retrievedLocationLines := make(map[uint64][]locationLine, len(ids))
	for rows.Next() {
		var (
			lId int64
			fId int64
		)
		l := locationLine{}
		err := rows.Scan(
			&lId, &l.Line, &fId,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("scan row:%w", err)
		}
		locationId := uint64(lId)
		l.FunctionID = uint64(fId)

		if _, found := retrievedLocationLines[locationId]; !found {
			retrievedLocationLines[locationId] = []locationLine{}
		}
		retrievedLocationLines[locationId] = append(retrievedLocationLines[locationId], l)

		if _, seen := functionIDsSeen[l.FunctionID]; !seen {
			functionIDs = append(functionIDs, l.FunctionID)
			functionIDsSeen[l.FunctionID] = struct{}{}
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, fmt.Errorf("iterate over SQL rows: %w", err)
	}

	for id, ll := range retrievedLocationLines {
		res[id] = ll
		err = s.cache.setLocationLinesByID(ctx, id, ll)
		if err != nil {
			return res, functionIDs, fmt.Errorf("set location lines by ID in cache: %w", err)
		}
	}

	return res, functionIDs, nil
}

const (
	linesByLocationsIDsQueryStart = `SELECT "location_id", "line", "function_id"
			FROM "lines" WHERE location_id IN (`
	comma          = ','
	closingBracket = ')'
)

func buildLinesByLocationIDsQuery(max uint64, ids []uint64) string {
	maxLen := len(strconv.FormatUint(max, 10))

	query := make([]byte, 0,
		// The max value is known, and invididual string can be larger than it.
		len(ids)*maxLen+
			// Add the start of the query.
			len(linesByLocationsIDsQueryStart)+
			// len(ids)-1 commas, and a closing bracket is len(ids).
			len(ids),
	)
	query = append(query, linesByLocationsIDsQueryStart...)

	for i := range ids {
		if i > 0 {
			query = append(query, comma)
		}
		query = strconv.AppendUint(query, ids[i], 10)
	}

	query = append(query, closingBracket)
	return unsafeString(query)
}

func unsafeString(b []byte) string {
	return *((*string)(unsafe.Pointer(&b)))
}

func (s *sqlMetaStore) getFunctionsByIDs(ctx context.Context, ids ...uint64) (map[uint64]*profile.Function, error) {
	ctx, span := s.tracer.Start(ctx, "getFunctionsByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("functions-ids-length", len(ids)))

	res := make(map[uint64]*profile.Function, len(ids))
	remainingIds := []uint64{}
	for _, id := range ids {
		f, found, err := s.cache.getFunctionByID(ctx, id)
		if err != nil {
			return res, fmt.Errorf("get function by ID from cache: %w", err)
		}
		if found {
			res[id] = &f
			continue
		}
		remainingIds = append(remainingIds, id)
	}
	ids = remainingIds

	if len(ids) == 0 {
		return res, nil
	}

	sIds := ""
	for i, id := range ids {
		if i > 0 {
			sIds += ","
		}
		sIds += strconv.FormatInt(int64(id), 10)
	}

	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(
			`SELECT "id", "name", "system_name", "filename", "start_line"
				FROM "functions" WHERE id IN (%s)`, sIds),
	)
	if err != nil {
		return nil, fmt.Errorf("execute SQL query: %w", err)
	}

	defer rows.Close()

	retrievedFunctions := make(map[uint64]profile.Function, len(ids))
	for rows.Next() {
		var (
			fId int64
			f   profile.Function
		)
		err := rows.Scan(
			&fId, &f.Name, &f.SystemName, &f.Filename, &f.StartLine,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		f.ID = uint64(fId)
		retrievedFunctions[f.ID] = f
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	for id, f := range retrievedFunctions {
		res[id] = &f
		err = s.cache.setFunctionByID(ctx, f)
		if err != nil {
			return res, fmt.Errorf("set function by ID in cache: %w", err)
		}
	}

	return res, nil
}

func (s *sqlMetaStore) CreateLocation(ctx context.Context, l *profile.Location) (uint64, error) {
	k := MakeLocationKey(l)
	var (
		stmt *sql.Stmt
		res  sql.Result
		err  error
		m    *profile.Mapping
	)
	var f func() error
	if l.Mapping != nil {
		// Make sure mapping already exists in the database.
		m, err = s.getMappingByID(ctx, int64(l.Mapping.ID))
		if err != nil {
			return 0, fmt.Errorf("get mapping by id: %w", err)
		}

		stmt, err = s.db.PrepareContext(ctx, `INSERT INTO "locations" (
                         address, is_folded, mapping_id, normalized_address, lines
                         )
					values(?,?,?,?,?)`)
		if err != nil {
			return 0, fmt.Errorf("prepare SQL statement: %w", err)
		}
		defer stmt.Close()

		f = func() error {
			res, err = stmt.ExecContext(ctx, int64(l.Address), l.IsFolded, int64(m.ID), int64(k.NormalizedAddress), k.Lines)
			return err
		}
	} else {
		stmt, err = s.db.PrepareContext(ctx, `INSERT INTO "locations" (
                          address, is_folded, normalized_address, lines
                         ) values(?,?,?,?)`)
		if err != nil {
			return 0, fmt.Errorf("CreateLocation failed: %w", err)
		}
		defer stmt.Close()

		f = func() error {
			res, err = stmt.ExecContext(ctx, int64(l.Address), l.IsFolded, int64(k.NormalizedAddress), k.Lines)
			return err
		}
	}

	if err := backoff.Retry(f, backoff.WithContext(backoff.WithMaxRetries(backoff.NewConstantBackOff(10*time.Millisecond), 3), ctx)); err != nil {
		return 0, fmt.Errorf("backoff SQL statement: %w", err)
	}

	if err != nil {
		return 0, fmt.Errorf("execute SQL statement: %w", err)
	}

	locID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("retrieve last inserted ID: %w", err)
	}

	if err := s.createLines(ctx, l.Line, locID); err != nil {
		return 0, fmt.Errorf("create lines: %w", err)
	}

	return uint64(locID), nil
}

func (s *sqlMetaStore) Symbolize(ctx context.Context, l *profile.Location) error {
	// NOTICE: We assume the given location is already persisted in the database.
	if err := s.createLines(ctx, l.Line, int64(l.ID)); err != nil {
		return fmt.Errorf("create lines: %w", err)
	}

	return nil
}

func (s *sqlMetaStore) GetLocations(ctx context.Context) ([]*profile.Location, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT l."id", l."address", l."is_folded", m."id",
       					m."start", m."limit", m."offset", m."file", m."build_id",
       					m."has_functions", m."has_filenames", m."has_line_numbers", m."has_inline_frames"
				FROM "locations" l
				LEFT JOIN "mappings" m ON l.mapping_id = m.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("GetLocations failed: %w", err)
	}
	defer rows.Close()

	locs := []*profile.Location{}
	for rows.Next() {
		l := &profile.Location{}
		var (
			mappingID       *int64
			start           *int64
			limit           *int64
			offset          *int64
			file            *string
			buildID         *string
			hasFunctions    *bool
			hasFilenames    *bool
			hasLineNumbers  *bool
			hasInlineFrames *bool
			locID           int64
			locAddress      int64
		)
		err := rows.Scan(
			&locID, &locAddress, &l.IsFolded,
			&mappingID, &start, &limit, &offset, &file, &buildID,
			&hasFunctions, &hasFilenames, &hasLineNumbers, &hasInlineFrames,
		)
		if err != nil {
			return nil, fmt.Errorf("GetLocations failed: %w", err)
		}
		l.ID = uint64(locID)
		l.Address = uint64(locAddress)
		if mappingID != nil {
			l.Mapping = &profile.Mapping{
				ID:              uint64(*mappingID),
				Start:           uint64(*start),
				Limit:           uint64(*limit),
				Offset:          uint64(*offset),
				File:            *file,
				BuildID:         *buildID,
				HasFunctions:    *hasFunctions,
				HasFilenames:    *hasFilenames,
				HasLineNumbers:  *hasLineNumbers,
				HasInlineFrames: *hasInlineFrames,
			}
		}

		lines, err := s.getLocationLines(ctx, l.ID)
		if err != nil {
			return nil, fmt.Errorf("GetLocations failed: %w", err)
		}
		l.Line = lines

		locs = append(locs, l)
	}
	return locs, nil
}

func (s *sqlMetaStore) GetSymbolizableLocations(ctx context.Context) ([]*profile.Location, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT l."id", l."address", l."is_folded", m."id",
       					m."start", m."limit", m."offset", m."file", m."build_id",
       					m."has_functions", m."has_filenames", 
       					m."has_line_numbers", m."has_inline_frames"
				FROM "locations" l
				JOIN "mappings" m ON l.mapping_id = m.id
				LEFT JOIN "lines" ln ON l."id" = ln."location_id"
                WHERE l.normalized_address > 0
                  AND ln."line" IS NULL 
                  AND l."id" IS NOT NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("GetSymbolizableLocations failed: %w", err)
	}
	defer rows.Close()

	locs := []*profile.Location{}
	for rows.Next() {
		l := &profile.Location{}
		var (
			mappingID       *int64
			start           *int64
			limit           *int64
			offset          *int64
			file            *string
			buildID         *string
			hasFunctions    *bool
			hasFilenames    *bool
			hasLineNumbers  *bool
			hasInlineFrames *bool
			locID           int64
			locAddress      int64
		)
		err := rows.Scan(
			&locID, &locAddress, &l.IsFolded,
			&mappingID, &start, &limit, &offset, &file, &buildID,
			&hasFunctions, &hasFilenames, &hasLineNumbers, &hasInlineFrames,
		)
		if err != nil {
			return nil, fmt.Errorf("GetSymbolizableLocations failed: %w", err)
		}
		l.ID = uint64(locID)
		l.Address = uint64(locAddress)
		if mappingID != nil {
			l.Mapping = &profile.Mapping{
				ID:              uint64(*mappingID),
				Start:           uint64(*start),
				Limit:           uint64(*limit),
				Offset:          uint64(*offset),
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

func (s *sqlMetaStore) GetFunctionByKey(ctx context.Context, k FunctionKey) (*profile.Function, error) {
	var (
		fn profile.Function
		id int64
	)

	fn, found, err := s.cache.getFunctionByKey(ctx, k)
	if err != nil {
		return nil, fmt.Errorf("get function by key from cache: %w", err)
	}
	if found {
		return &fn, nil
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT "id", "name", "system_name", "filename", "start_line"
				FROM "functions"
				WHERE start_line=? AND name=? AND system_name=? AND filename=?`,
		k.StartLine, k.Name, k.SystemName, k.FileName,
	).Scan(&id, &fn.Name, &fn.SystemName, &fn.Filename, &fn.StartLine); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFunctionNotFound
		}
		return nil, fmt.Errorf("execute SQL statement: %w", err)
	}
	fn.ID = uint64(id)

	err = s.cache.setFunctionByKey(ctx, k, fn)
	if err != nil {
		return nil, fmt.Errorf("set function by key in cache: %w", err)
	}

	return &fn, nil
}

func (s *sqlMetaStore) CreateFunction(ctx context.Context, fn *profile.Function) (uint64, error) {
	var (
		stmt *sql.Stmt
		res  sql.Result
		err  error
	)
	stmt, err = s.db.PrepareContext(ctx,
		`INSERT INTO "functions" (
                         name, system_name, filename, start_line
                         ) values(?,?,?,?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("CreateFunction failed: %w", err)
	}
	defer stmt.Close()

	res, err = stmt.ExecContext(ctx, fn.Name, fn.SystemName, fn.Filename, fn.StartLine)

	if err != nil {
		return 0, fmt.Errorf("CreateFunction failed: %w", err)
	}

	fnID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CreateFunction failed: %w", err)
	}

	return uint64(fnID), nil
}

func (s *sqlMetaStore) GetFunctions(ctx context.Context) ([]*profile.Function, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT "id", "name", "system_name", "filename", "start_line" FROM "functions"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	funcs := []*profile.Function{}
	for rows.Next() {
		f := profile.Function{}
		var id int64
		err := rows.Scan(&id, &f.Name, &f.SystemName, &f.Filename, &f.StartLine)
		if err != nil {
			return nil, fmt.Errorf("GetFunctions failed: %w", err)
		}
		f.ID = uint64(id)
		funcs = append(funcs, &f)
	}

	return funcs, nil
}

func (s *sqlMetaStore) GetMappingByKey(ctx context.Context, k MappingKey) (*profile.Mapping, error) {
	var (
		m                        profile.Mapping
		id, start, limit, offset int64
	)

	m, found, err := s.cache.getMappingByKey(ctx, k)
	if err != nil {
		return nil, err
	}
	if found {
		return &m, nil
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT "id", "start", "limit", "offset", "file", "build_id",
				"has_functions", "has_filenames", "has_line_numbers", "has_inline_frames"
				FROM "mappings"
				WHERE size=? AND offset=? AND build_id_or_file=?`,
		int64(k.Size), int64(k.Offset), k.BuildIDOrFile,
	).Scan(
		&id, &start, &limit, &offset, &m.File, &m.BuildID,
		&m.HasFunctions, &m.HasFilenames, &m.HasLineNumbers, &m.HasInlineFrames,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMappingNotFound
		}
		return nil, fmt.Errorf("GetMappingByKey failed: %w", err)
	}
	m.ID = uint64(id)
	m.Start = uint64(start)
	m.Limit = uint64(limit)
	m.Offset = uint64(offset)

	err = s.cache.setMappingByKey(ctx, k, m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (s *sqlMetaStore) CreateMapping(ctx context.Context, m *profile.Mapping) (uint64, error) {
	var (
		stmt *sql.Stmt
		res  sql.Result
		err  error
	)
	if m.ID == 0 {
		stmt, err = s.db.PrepareContext(ctx,
			`INSERT INTO "mappings" (
                        "start", "limit", "offset", "file", "build_id",
                        "has_functions", "has_filenames", "has_line_numbers", "has_inline_frames",
                        "size", "build_id_or_file"
                        ) values(?,?,?,?,?,?,?,?,?,?,?)`,
		)
		if err != nil {
			return 0, fmt.Errorf("CreateMapping failed: %w", err)
		}
		defer stmt.Close()

		k := MakeMappingKey(m)
		res, err = stmt.ExecContext(ctx,
			int64(m.Start), int64(m.Limit), int64(m.Offset), m.File, m.BuildID,
			m.HasFunctions, m.HasFilenames, m.HasLineNumbers, m.HasInlineFrames,
			int64(k.Size), k.BuildIDOrFile,
		)
	} else {
		stmt, err = s.db.PrepareContext(ctx,
			`INSERT INTO "mappings" (
                        "id", "start", "limit", "offset", "file", "build_id",
                        "has_functions", "has_filenames", "has_line_numbers", "has_inline_frames",
                        "size", "build_id_or_file"
                        ) values(?,?,?,?,?,?,?,?,?,?,?,?)`,
		)
		if err != nil {
			return 0, fmt.Errorf("CreateMapping failed: %w", err)
		}
		defer stmt.Close()

		k := MakeMappingKey(m)
		res, err = stmt.ExecContext(ctx,
			int64(m.ID), int64(m.Start), int64(m.Limit), int64(m.Offset), m.File, m.BuildID,
			m.HasFunctions, m.HasFilenames, m.HasLineNumbers, m.HasInlineFrames,
			int64(k.Size), k.BuildIDOrFile,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("CreateMapping failed: %w", err)
	}

	mID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CreateMapping failed: %w", err)
	}

	return uint64(mID), nil
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

func (s *sqlMetaStore) getMappingByID(ctx context.Context, mid int64) (*profile.Mapping, error) {
	var (
		m                        profile.Mapping
		id, start, limit, offset int64
	)

	m, found, err := s.cache.getMappingByID(ctx, uint64(mid))
	if err != nil {
		return nil, err
	}
	if found {
		return &m, nil
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT "id", "start", "limit", "offset", "file", "build_id",
				"has_functions", "has_filenames", "has_line_numbers", "has_inline_frames"
				FROM "mappings" WHERE id=?`, mid,
	).Scan(
		&id, &start, &limit, &offset, &m.File, &m.BuildID,
		&m.HasFunctions, &m.HasFilenames, &m.HasLineNumbers, &m.HasInlineFrames,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMappingNotFound
		}
		return nil, fmt.Errorf("getMappingByID failed: %w", err)
	}
	m.ID = uint64(id)
	m.Start = uint64(start)
	m.Limit = uint64(limit)
	m.Offset = uint64(offset)

	err = s.cache.setMappingByID(ctx, m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (s *sqlMetaStore) getLocationLines(ctx context.Context, locationID uint64) ([]profile.Line, error) {
	var lines []profile.Line
	rows, err := s.db.QueryContext(ctx,
		`SELECT ln."line", fn."id", fn."name", fn."system_name", fn."filename", fn."start_line"
				FROM "lines" ln
				JOIN "locations" loc ON ln."location_id" = loc."id"
				JOIN "functions" fn ON ln."function_id" = fn."id"
				WHERE loc."id"=? ORDER BY ln."line" ASC`, int64(locationID),
	)
	if err != nil {
		return nil, fmt.Errorf("getLocationLines failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ln := profile.Line{}
		fn := profile.Function{}
		var fnID int64
		err := rows.Scan(&ln.Line, &fnID, &fn.Name, &fn.SystemName, &fn.Filename, &fn.StartLine)
		if err != nil {
			return nil, fmt.Errorf("getLocationLines failed: %w", err)
		}
		fn.ID = uint64(fnID)
		ln.Function = &fn
		lines = append(lines, ln)
	}

	return lines, nil
}

func (s *sqlMetaStore) getOrCreateFunction(ctx context.Context, f *profile.Function) (uint64, error) {
	fn, err := s.GetFunctionByKey(ctx, MakeFunctionKey(f))
	if err == nil {
		return fn.ID, nil
	}
	if err != nil && err != ErrFunctionNotFound {
		return 0, err
	}

	fnID, err := s.CreateFunction(ctx, f)
	if err != nil {
		return 0, err
	}
	return fnID, nil
}

func (s *sqlMetaStore) createLines(ctx context.Context, lines []profile.Line, locID int64) error {
	if len(lines) > 0 {
		q := `INSERT INTO "lines" (location_id, line, function_id) VALUES `
		ll := make([]locationLine, 0, len(lines))
		for i, ln := range lines {
			fnID, err := s.getOrCreateFunction(ctx, ln.Function)
			if err != nil {
				return err
			}
			q += fmt.Sprintf(`(%s, %s, %s)`,
				strconv.FormatInt(locID, 10),
				strconv.FormatInt(ln.Line, 10),
				strconv.FormatInt(int64(fnID), 10))
			if i != len(lines)-1 {
				q += ", "
			}
			ll = append(ll, locationLine{
				Line:       ln.Line,
				FunctionID: fnID,
			})
		}
		q += ";"
		stmt, err := s.db.PrepareContext(ctx, q)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx)
		if err != nil {
			return err
		}

		err = s.cache.setLocationLinesByID(ctx, uint64(locID), ll)
		if err != nil {
			return err
		}
	}
	return nil
}
