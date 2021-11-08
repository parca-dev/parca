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
	"bytes"
	"context"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

// UUIDGenerator returns new UUIDs.
type UUIDGenerator interface {
	New() uuid.UUID
}

// RandomUUIDGenerator returns a new random UUID.
type RandomUUIDGenerator struct{}

// New returns a new UUID.
func (g *RandomUUIDGenerator) New() uuid.UUID {
	return uuid.New()
}

// NewRandomUUIDGenerator returns a new random UUID generator.
func NewRandomUUIDGenerator() UUIDGenerator {
	return &RandomUUIDGenerator{}
}

// BadgerMetastore is an implementation of the metastore using the badger KV
// store.
type BadgerMetastore struct {
	tracer trace.Tracer

	db *badger.DB

	uuidGenerator UUIDGenerator
}

// NewBadgerMetastore returns a new BadgerMetastore with using in-memory badger
// instance.
func NewBadgerMetastore(
	reg prometheus.Registerer,
	tracer trace.Tracer,
	uuidGenerator UUIDGenerator,
) *BadgerMetastore {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}

	return &BadgerMetastore{
		db:            db,
		tracer:        tracer,
		uuidGenerator: uuidGenerator,
	}
}

// Close closes the badger store.
func (m *BadgerMetastore) Close() error {
	return m.db.Close()
}

// Ping returns an error if the metastore is not available.
func (m *BadgerMetastore) Ping() error {
	return nil
}

// GetMappingsByIDs returns the mappings for the given IDs.
func (m *BadgerMetastore) GetMappingsByIDs(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]*Mapping, error) {
	mappings := map[uuid.UUID]*Mapping{}
	err := m.db.View(func(txn *badger.Txn) error {
		for _, id := range ids {
			item, err := txn.Get(append([]byte("mappings/by-id/"), id[:]...))
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				ma := &pb.Mapping{}
				err := proto.Unmarshal(val, ma)
				if err != nil {
					return err
				}
				id, err := uuid.FromBytes(ma.Id)
				if err != nil {
					return err
				}

				mappings[id] = &Mapping{
					ID:              id,
					Start:           ma.Start,
					Limit:           ma.Limit,
					Offset:          ma.Offset,
					File:            ma.File,
					BuildID:         ma.BuildId,
					HasFunctions:    ma.HasFunctions,
					HasFilenames:    ma.HasFilenames,
					HasLineNumbers:  ma.HasLineNumbers,
					HasInlineFrames: ma.HasInlineFrames,
				}

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return mappings, err
}

// GetMappingByKey returns the mapping for the given key.
func (m *BadgerMetastore) GetMappingByKey(ctx context.Context, k MappingKey) (*Mapping, error) {
	ma := &pb.Mapping{}
	err := m.db.View(func(txn *badger.Txn) error {
		var err error
		item, err := txn.Get(k.Bytes())
		if err == badger.ErrKeyNotFound {
			return ErrMappingNotFound
		}
		if err != nil {
			return err
		}

		var mappingID uuid.UUID
		err = item.Value(func(val []byte) error {
			return mappingID.UnmarshalBinary(val)
		})
		if err != nil {
			return err
		}

		item, err = txn.Get(append([]byte("mappings/by-id/"), mappingID[:]...))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, ma)
		})
	})
	if err != nil {
		return nil, err
	}

	id, err := uuid.FromBytes(ma.Id)
	if err != nil {
		return nil, err
	}

	return &Mapping{
		ID:              id,
		Start:           ma.Start,
		Limit:           ma.Limit,
		Offset:          ma.Offset,
		File:            ma.File,
		BuildID:         ma.BuildId,
		HasFunctions:    ma.HasFunctions,
		HasFilenames:    ma.HasFilenames,
		HasLineNumbers:  ma.HasLineNumbers,
		HasInlineFrames: ma.HasInlineFrames,
	}, err
}

// CreateMapping creates a new mapping in the database.
func (m *BadgerMetastore) CreateMapping(ctx context.Context, mapping *Mapping) (uuid.UUID, error) {
	mappingID := m.uuidGenerator.New()
	ma := &pb.Mapping{
		Id:              mappingID[:],
		Start:           mapping.Start,
		Limit:           mapping.Limit,
		Offset:          mapping.Offset,
		File:            mapping.File,
		BuildId:         mapping.BuildID,
		HasFunctions:    mapping.HasFunctions,
		HasFilenames:    mapping.HasFilenames,
		HasLineNumbers:  mapping.HasLineNumbers,
		HasInlineFrames: mapping.HasInlineFrames,
	}

	buf, err := proto.Marshal(ma)
	if err != nil {
		return uuid.Nil, err
	}

	err = m.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(mapping.Key().Bytes(), mappingID[:])
		if err != nil {
			return err
		}

		return txn.Set(append([]byte("mappings/by-id/"), mappingID[:]...), buf)
	})

	return mappingID, err
}

// CreateFunction creates a new function in the database.
func (m *BadgerMetastore) CreateFunction(ctx context.Context, function *Function) (uuid.UUID, error) {
	functionID := m.uuidGenerator.New()
	f := &pb.Function{
		Id:         functionID[:],
		StartLine:  function.FunctionKey.StartLine,
		Name:       function.FunctionKey.Name,
		SystemName: function.FunctionKey.SystemName,
		Filename:   function.FunctionKey.Filename,
	}

	buf, err := proto.Marshal(f)
	if err != nil {
		return uuid.Nil, err
	}

	err = m.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(function.Key().Bytes(), f.Id)
		if err != nil {
			return err
		}

		return txn.Set(append([]byte("functions/by-id/"), f.Id...), buf)
	})

	return functionID, err
}

// GetFunctionByKey returns the function for the given key.
func (m *BadgerMetastore) GetFunctionByKey(ctx context.Context, k FunctionKey) (*Function, error) {
	f := &pb.Function{}
	err := m.db.View(func(txn *badger.Txn) error {
		var err error
		item, err := txn.Get(k.Bytes())
		if err == badger.ErrKeyNotFound {
			return ErrFunctionNotFound
		}
		if err != nil {
			return fmt.Errorf("get function by key from store: %w", err)
		}

		var functionID uuid.UUID
		err = item.Value(func(val []byte) error {
			return functionID.UnmarshalBinary(val)
		})
		if err != nil {
			return err
		}

		item, err = txn.Get(append([]byte("functions/by-id/"), functionID[:]...))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, f)
		})
	})
	if err != nil {
		return nil, err
	}

	id, err := uuid.FromBytes(f.Id)
	if err != nil {
		return nil, fmt.Errorf("parse function ID (%v): %w", f, err)
	}

	return &Function{
		ID: id,
		FunctionKey: FunctionKey{
			StartLine:  f.StartLine,
			Name:       f.Name,
			SystemName: f.SystemName,
			Filename:   f.Filename,
		},
	}, nil
}

// GetFunctions returns all functions in the database.
func (m *BadgerMetastore) GetFunctions(ctx context.Context) ([]*Function, error) {
	var functions []*Function
	err := m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := []byte("functions/by-id/")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				f := &pb.Function{}
				err := proto.Unmarshal(val, f)
				if err != nil {
					return err
				}
				id, err := uuid.FromBytes(f.Id)
				if err != nil {
					return err
				}
				functions = append(functions, &Function{
					ID: id,
					FunctionKey: FunctionKey{
						StartLine:  f.StartLine,
						Name:       f.Name,
						SystemName: f.SystemName,
						Filename:   f.Filename,
					},
				})
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return functions, err
}

// GetFunctionByID returns the function for the given ID.
func (m *BadgerMetastore) GetFunctionsByIDs(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]*Function, error) {
	functions := map[uuid.UUID]*Function{}
	err := m.db.View(func(txn *badger.Txn) error {
		for _, id := range ids {
			item, err := txn.Get(append([]byte("functions/by-id/"), id[:]...))
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				f := &pb.Function{}
				err := proto.Unmarshal(val, f)
				if err != nil {
					return err
				}
				id, err := uuid.FromBytes(f.Id)
				if err != nil {
					return err
				}

				functions[id] = &Function{
					ID: id,
					FunctionKey: FunctionKey{
						StartLine:  f.StartLine,
						Name:       f.Name,
						SystemName: f.SystemName,
						Filename:   f.Filename,
					},
				}

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return functions, err
}

// CreateLocationLines writes a set of lines related to a location to the database.
func (m *BadgerMetastore) CreateLocationLines(ctx context.Context, locID uuid.UUID, lines []LocationLine) error {
	l := &pb.LocationLines{
		Id:    locID[:],
		Lines: make([]*pb.Line, 0, len(lines)),
	}

	for _, line := range lines {
		l.Lines = append(l.Lines, &pb.Line{
			Line:       line.Line,
			FunctionId: line.Function.ID[:],
		})
	}

	buf, err := proto.Marshal(l)
	if err != nil {
		return err
	}

	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Set(append([]byte("locations-lines/"), locID[:]...), buf)
	})
}

// GetLinesByLocationIDs returns the lines for the given location IDs.
func (m *BadgerMetastore) GetLinesByLocationIDs(ctx context.Context, ids ...uuid.UUID) (
	map[uuid.UUID][]Line,
	[]uuid.UUID,
	error,
) {
	linesByLocation := map[uuid.UUID][]Line{}
	functionsSeen := map[uuid.UUID]struct{}{}
	functionsIDs := []uuid.UUID{}
	err := m.db.View(func(txn *badger.Txn) error {
		for _, id := range ids {
			item, err := txn.Get(append([]byte("locations-lines/"), id[:]...))
			if err == badger.ErrKeyNotFound {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to get location lines for ID %s: %w", id.String(), err)
			}

			err = item.Value(func(val []byte) error {
				l := &pb.LocationLines{}
				err := proto.Unmarshal(val, l)
				if err != nil {
					return fmt.Errorf("failed to unmarshal location lines for ID %s: %w", id.String(), err)
				}

				lines := make([]Line, 0, len(l.Lines))
				for _, line := range l.Lines {
					functionID, err := uuid.FromBytes(line.FunctionId)
					if err != nil {
						return fmt.Errorf("function ID from bytes: %w", err)
					}

					if _, ok := functionsSeen[functionID]; !ok {
						functionsIDs = append(functionsIDs, functionID)
						functionsSeen[functionID] = struct{}{}
					}

					lines = append(lines, Line{
						Line:       line.Line,
						FunctionID: functionID,
					})
				}

				linesByLocation[id] = lines
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return linesByLocation, functionsIDs, nil
}

func (m *BadgerMetastore) GetLocationByKey(ctx context.Context, k LocationKey) (SerializedLocation, error) {
	l := &pb.Location{}
	err := m.db.View(func(txn *badger.Txn) error {
		var err error
		item, err := txn.Get(k.Bytes())
		if err == badger.ErrKeyNotFound {
			return ErrLocationNotFound
		}
		if err != nil {
			return err
		}

		var locationID uuid.UUID
		err = item.Value(func(val []byte) error {
			return locationID.UnmarshalBinary(val)
		})
		if err != nil {
			return err
		}

		item, err = txn.Get(append([]byte("locations/by-id/"), locationID[:]...))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, l)
		})
	})
	if err != nil {
		return SerializedLocation{}, err
	}

	id, err := uuid.FromBytes(l.Id)
	if err != nil {
		return SerializedLocation{}, err
	}

	var mappingID uuid.UUID
	if len(l.MappingId) > 0 && !bytes.Equal(l.MappingId, uuid.Nil[:]) {
		mappingID, err = uuid.FromBytes(l.MappingId)
		if err != nil {
			return SerializedLocation{}, err
		}
	}

	return SerializedLocation{
		ID:        id,
		Address:   l.Address,
		MappingID: mappingID,
		IsFolded:  l.IsFolded,
	}, nil
}

func (m *BadgerMetastore) GetLocationsByIDs(ctx context.Context, ids ...uuid.UUID) (
	map[uuid.UUID]SerializedLocation,
	[]uuid.UUID,
	error,
) {
	locations := map[uuid.UUID]SerializedLocation{}
	mappingsSeen := map[uuid.UUID]struct{}{}
	mappingIDs := []uuid.UUID{}
	err := m.db.View(func(txn *badger.Txn) error {
		for _, id := range ids {
			item, err := txn.Get(append([]byte("locations/by-id/"), id[:]...))
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				l := &pb.Location{}
				err := proto.Unmarshal(val, l)
				if err != nil {
					return err
				}

				var mappingID uuid.UUID
				if len(l.MappingId) > 0 && !bytes.Equal(l.MappingId, uuid.Nil[:]) {
					mappingID, err = uuid.FromBytes(l.MappingId)
					if err != nil {
						return fmt.Errorf("mapping ID from bytes: %w", err)
					}

					if _, ok := mappingsSeen[mappingID]; !ok {
						mappingIDs = append(mappingIDs, mappingID)
						mappingsSeen[mappingID] = struct{}{}
					}
				}

				locations[id] = SerializedLocation{
					ID:        id,
					Address:   l.Address,
					MappingID: mappingID,
					IsFolded:  l.IsFolded,
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return locations, mappingIDs, nil
}

func (m *BadgerMetastore) CreateLocation(ctx context.Context, l *Location) (uuid.UUID, error) {
	id := m.uuidGenerator.New()
	loc := &pb.Location{
		Id:       id[:],
		Address:  l.Address,
		IsFolded: l.IsFolded,
	}

	if l.Mapping != nil {
		loc.MappingId = l.Mapping.ID[:]
	}

	buf, err := proto.Marshal(loc)
	if err != nil {
		return uuid.Nil, err
	}

	err = m.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(l.Key().Bytes(), id[:])
		if err != nil {
			return err
		}

		if l.Address != uint64(0) && l.Mapping != nil && len(l.Lines) == 0 {
			err := txn.Set(append([]byte("locations-unsymbolized/by-id/"), id[:]...), id[:])
			if err != nil {
				return err
			}
		}

		return txn.Set(append([]byte("locations/by-id/"), id[:]...), buf)
	})
	if err != nil {
		return uuid.Nil, err
	}

	if len(l.Lines) > 0 {
		return id, m.CreateLocationLines(ctx, id, l.Lines)
	}

	return id, nil
}

func (m *BadgerMetastore) GetSymbolizableLocations(ctx context.Context) ([]SerializedLocation, []uuid.UUID, error) {
	ids := []uuid.UUID{}
	err := m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := []byte("locations-unsymbolized/by-id/")
		prefixLen := len(prefix)

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			id, err := uuid.FromBytes(item.Key()[prefixLen:])
			if err != nil {
				return err
			}

			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	locsByIDs, mappingIDs, err := m.GetLocationsByIDs(ctx, ids...)
	if err != nil {
		return nil, nil, err
	}

	locs := make([]SerializedLocation, 0, len(locsByIDs))
	for _, loc := range locsByIDs {
		locs = append(locs, loc)
	}

	return locs, mappingIDs, nil
}

func (m *BadgerMetastore) GetLocations(ctx context.Context) ([]SerializedLocation, []uuid.UUID, error) {
	return m.getLocations(ctx, []byte("locations/by-id/"))
}

func (m *BadgerMetastore) getLocations(ctx context.Context, prefix []byte) ([]SerializedLocation, []uuid.UUID, error) {
	locations := []SerializedLocation{}
	mappingsSeen := map[uuid.UUID]struct{}{}
	mappingIDs := []uuid.UUID{}
	err := m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				l := &pb.Location{}
				err := proto.Unmarshal(val, l)
				if err != nil {
					return err
				}
				id, err := uuid.FromBytes(l.Id)
				if err != nil {
					return err
				}

				var mappingID uuid.UUID
				if len(l.MappingId) > 0 && !bytes.Equal(l.MappingId, uuid.Nil[:]) {
					mappingID, err = uuid.FromBytes(l.MappingId)
					if err != nil {
						return fmt.Errorf("mapping ID from bytes: %w", err)
					}

					if _, ok := mappingsSeen[mappingID]; !ok {
						mappingIDs = append(mappingIDs, mappingID)
						mappingsSeen[mappingID] = struct{}{}
					}
				}

				locations = append(locations, SerializedLocation{
					ID:        id,
					Address:   l.Address,
					MappingID: mappingID,
					IsFolded:  l.IsFolded,
				})
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return locations, mappingIDs, err
}

func (m *BadgerMetastore) Symbolize(ctx context.Context, l *Location) error {
	var err error
	for _, l := range l.Lines {
		l.Function.ID, err = m.getOrCreateFunction(ctx, l.Function)
		if err != nil {
			return fmt.Errorf("get or create function: %w", err)
		}
	}

	if err := m.CreateLocationLines(ctx, l.ID, l.Lines); err != nil {
		return fmt.Errorf("create lines: %w", err)
	}

	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(append([]byte("locations-unsymbolized/by-id/"), l.ID[:]...))
	})
}

func (m *BadgerMetastore) getOrCreateFunction(ctx context.Context, f *Function) (uuid.UUID, error) {
	fn, err := m.GetFunctionByKey(ctx, MakeFunctionKey(f))
	if err == nil {
		return fn.ID, nil
	}
	if err != nil && err != ErrFunctionNotFound {
		return uuid.Nil, fmt.Errorf("get function by key: %w", err)
	}

	id, err := m.CreateFunction(ctx, f)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create function: %w", err)
	}

	return id, nil
}
