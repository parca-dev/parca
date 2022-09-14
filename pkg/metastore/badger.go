// Copyright 2022 The Parca Authors
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
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

// BadgerMetastore is an implementation of the metastore using the badger KV
// store.
type BadgerMetastore struct {
	tracer trace.Tracer
	logger log.Logger

	db *badger.DB

	pb.UnimplementedMetastoreServiceServer
}

type BadgerLogger struct {
	Logger log.Logger
}

func (l *BadgerLogger) Errorf(f string, v ...interface{}) {
	level.Error(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Warningf(f string, v ...interface{}) {
	level.Warn(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Infof(f string, v ...interface{}) {
	level.Info(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Debugf(f string, v ...interface{}) {
	level.Debug(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

var _ pb.MetastoreServiceServer = &BadgerMetastore{}

// NewBadgerMetastore returns a new BadgerMetastore with using in-memory badger
// instance.
func NewBadgerMetastore(
	logger log.Logger,
	reg prometheus.Registerer,
	tracer trace.Tracer,
	db *badger.DB,
) *BadgerMetastore {
	return &BadgerMetastore{
		db:     db,
		tracer: tracer,
		logger: logger,
	}
}

func (m *BadgerMetastore) Mappings(ctx context.Context, r *pb.MappingsRequest) (*pb.MappingsResponse, error) {
	res := &pb.MappingsResponse{
		Mappings: make([]*pb.Mapping, 0, len(r.MappingIds)),
	}

	mappingKeys := make([][]byte, 0, len(r.MappingIds))
	for _, id := range r.MappingIds {
		mappingKeys = append(mappingKeys, []byte(MakeMappingKeyWithID(id)))
	}

	err := m.db.View(func(txn *badger.Txn) error {
		for _, mappingKey := range mappingKeys {
			item, err := txn.Get(mappingKey)
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				mapping := &pb.Mapping{}
				err := mapping.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Mappings = append(res.Mappings, mapping)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}

func (m *BadgerMetastore) GetOrCreateMappings(ctx context.Context, r *pb.GetOrCreateMappingsRequest) (*pb.GetOrCreateMappingsResponse, error) {
	res := &pb.GetOrCreateMappingsResponse{
		Mappings: make([]*pb.Mapping, 0, len(r.Mappings)),
	}

	mappingKeys := make([]string, 0, len(r.Mappings))
	for _, id := range r.Mappings {
		mappingKeys = append(mappingKeys, MakeMappingKey(id))
	}

	err := m.db.Update(func(txn *badger.Txn) error {
		for i, mappingKey := range mappingKeys {
			item, err := txn.Get([]byte(mappingKey))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			if err == badger.ErrKeyNotFound {
				mapping := r.Mappings[i]
				mapping.Id = MappingIDFromKey(mappingKey)
				b, err := mapping.MarshalVT()
				if err != nil {
					return err
				}
				if err := txn.Set([]byte(mappingKey), b); err != nil {
					return err
				}
				res.Mappings = append(res.Mappings, mapping)
				continue
			}

			err = item.Value(func(val []byte) error {
				mapping := &pb.Mapping{}
				err := mapping.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Mappings = append(res.Mappings, mapping)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}

func (m *BadgerMetastore) Functions(ctx context.Context, r *pb.FunctionsRequest) (*pb.FunctionsResponse, error) {
	res := &pb.FunctionsResponse{
		Functions: make([]*pb.Function, 0, len(r.FunctionIds)),
	}

	functionKeys := make([][]byte, 0, len(r.FunctionIds))
	for _, id := range r.FunctionIds {
		functionKeys = append(functionKeys, []byte(MakeFunctionKeyWithID(id)))
	}

	err := m.db.View(func(txn *badger.Txn) error {
		for _, functionKey := range functionKeys {
			item, err := txn.Get(functionKey)
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				function := &pb.Function{}
				err := function.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Functions = append(res.Functions, function)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}

func (m *BadgerMetastore) GetOrCreateFunctions(ctx context.Context, r *pb.GetOrCreateFunctionsRequest) (*pb.GetOrCreateFunctionsResponse, error) {
	res := &pb.GetOrCreateFunctionsResponse{
		Functions: make([]*pb.Function, 0, len(r.Functions)),
	}

	functionKeys := make([]string, 0, len(r.Functions))
	for _, function := range r.Functions {
		functionKeys = append(functionKeys, MakeFunctionKey(function))
	}

	err := m.db.Update(func(txn *badger.Txn) error {
		for i, functionKey := range functionKeys {
			item, err := txn.Get([]byte(functionKey))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			if err == badger.ErrKeyNotFound {
				function := r.Functions[i]
				function.Id = FunctionIDFromKey(functionKey)
				b, err := function.MarshalVT()
				if err != nil {
					return err
				}
				if err := txn.Set([]byte(functionKey), b); err != nil {
					return err
				}
				res.Functions = append(res.Functions, function)
				continue
			}

			err = item.Value(func(val []byte) error {
				function := &pb.Function{}
				err := function.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Functions = append(res.Functions, function)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}

func (m *BadgerMetastore) Locations(ctx context.Context, r *pb.LocationsRequest) (*pb.LocationsResponse, error) {
	res := &pb.LocationsResponse{
		Locations: make([]*pb.Location, 0, len(r.LocationIds)),
	}

	locationKeys := make([][]byte, 0, len(r.LocationIds))
	for _, id := range r.LocationIds {
		locationKeys = append(locationKeys, []byte(MakeLocationKeyWithID(id)))
	}

	err := m.db.View(func(txn *badger.Txn) error {
		var err error
		res.Locations, err = m.locations(ctx, txn, res.Locations, locationKeys)
		return err
	})

	return res, err
}

func (m *BadgerMetastore) locations(ctx context.Context, txn *badger.Txn, locations []*pb.Location, locationKeys [][]byte) ([]*pb.Location, error) {
	for _, locationKey := range locationKeys {
		item, err := txn.Get(locationKey)
		if err != nil {
			return nil, err
		}

		err = item.Value(func(val []byte) error {
			location := &pb.Location{}
			err := location.UnmarshalVT(val)
			if err != nil {
				return err
			}

			locations = append(locations, location)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return locations, nil
}

func (m *BadgerMetastore) GetOrCreateLocations(ctx context.Context, r *pb.GetOrCreateLocationsRequest) (*pb.GetOrCreateLocationsResponse, error) {
	res := &pb.GetOrCreateLocationsResponse{
		Locations: make([]*pb.Location, 0, len(r.Locations)),
	}

	locationKeys := make([]string, 0, len(r.Locations))
	for _, location := range r.Locations {
		locationKeys = append(locationKeys, MakeLocationKey(location))
	}

	err := m.db.Update(func(txn *badger.Txn) error {
		for i, locationKey := range locationKeys {
			item, err := txn.Get([]byte(locationKey))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			if err == badger.ErrKeyNotFound {
				location := r.Locations[i]
				location.Id = LocationIDFromKey(locationKey)
				b, err := location.MarshalVT()
				if err != nil {
					return err
				}
				if err := txn.Set([]byte(locationKey), b); err != nil {
					return err
				}
				res.Locations = append(res.Locations, location)

				if location.MappingId != "" && location.Address != 0 && len(location.Lines) == 0 {
					unsymbolizableKey := MakeUnsymbolizedLocationKeyWithID(location.Id)
					if err := txn.Set([]byte(unsymbolizableKey), []byte{}); err != nil {
						return err
					}
					continue
				}

				continue
			}

			err = item.Value(func(val []byte) error {
				location := &pb.Location{}
				err := location.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Locations = append(res.Locations, location)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}

func (m *BadgerMetastore) UnsymbolizedLocations(ctx context.Context, r *pb.UnsymbolizedLocationsRequest) (*pb.UnsymbolizedLocationsResponse, error) {
	var locations []*pb.Location

	maxKey := ""
	err := m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		locationKeys := [][]byte{}
		prefix := []byte(UnsymbolizedLocationLinesKeyPrefix)
		if len(r.MinKey) > 0 {
			it.Seek([]byte(r.MinKey))
			if !it.ValidForPrefix(prefix) {
				// No keys.
				return nil
			}
			// Need to skip the first one as the min is not supposed to be
			// included.
			it.Next()
		} else {
			it.Seek(prefix)
		}
		for it.ValidForPrefix(prefix) {
			maxKey = string(it.Item().Key())
			key := MakeLocationKeyWithID(LocationIDFromUnsymbolizedKey(maxKey))
			locationKeys = append(locationKeys, []byte(key))
			if uint32(len(locationKeys)) == r.Limit {
				break
			}
			it.Next()
		}

		locations = make([]*pb.Location, 0, len(locationKeys))
		var err error
		locations, err = m.locations(ctx, txn, locations, locationKeys)

		return err
	})
	if err != nil {
		return nil, err
	}

	return &pb.UnsymbolizedLocationsResponse{
		Locations: locations,
		MaxKey:    maxKey,
	}, nil
}

func (m *BadgerMetastore) CreateLocationLines(ctx context.Context, r *pb.CreateLocationLinesRequest) (*pb.CreateLocationLinesResponse, error) {
	err := m.db.Update(func(txn *badger.Txn) error {
		for _, location := range r.Locations {
			b, err := location.MarshalVT()
			if err != nil {
				return err
			}
			if err := txn.Set([]byte(MakeLocationKeyWithID(location.Id)), b); err != nil {
				return err
			}

			if err := txn.Delete([]byte(MakeUnsymbolizedLocationKeyWithID(location.Id))); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.CreateLocationLinesResponse{}, nil
}

func (m *BadgerMetastore) GetOrCreateStacktraces(ctx context.Context, r *pb.GetOrCreateStacktracesRequest) (*pb.GetOrCreateStacktracesResponse, error) {
	res := &pb.GetOrCreateStacktracesResponse{
		Stacktraces: make([]*pb.Stacktrace, 0, len(r.Stacktraces)),
	}

	stacktraceKeys := make([]string, 0, len(r.Stacktraces))
	for _, stacktrace := range r.Stacktraces {
		stacktraceKeys = append(stacktraceKeys, MakeStacktraceKey(stacktrace))
	}

	const maxRetries = 2
	var result retryableGetOrCreateStacktraces
	var err error

	level.Debug(m.logger).Log("msg", "GetOrCreateStacktraces", "stacktrace_keys_len", len(r.Stacktraces))
	for i := 0; i < maxRetries; i++ {
		result, err = m.retryableGetOrCreateStacktraces(r, stacktraceKeys)
		if err != nil {
			return res, err
		}

		res.Stacktraces = append(res.Stacktraces, result.stackTraces...)
		if len(result.retryWith) == 0 {
			break
		}
		stacktraceKeys = result.retryWith
		level.Debug(m.logger).Log("msg", "retrying GetOrCreateStacktraces", "stacktrace_keys_len", len(stacktraceKeys))
	}

	if len(result.retryWith) != 0 {
		level.Debug(m.logger).Log("msg", "failed to GetOrCreateStacktraces all stacktraces", "stacktrace_keys_len", len(result.retryWith))
		return res, fmt.Errorf("partial commit of stacktraces: %w", badger.ErrTxnTooBig)
	}
	return res, err
}

type retryableGetOrCreateStacktraces struct {
	stackTraces []*pb.Stacktrace
	retryWith   []string
}

func (m *BadgerMetastore) retryableGetOrCreateStacktraces(r *pb.GetOrCreateStacktracesRequest, stacktraceKeys []string) (retryableGetOrCreateStacktraces, error) {
	result := retryableGetOrCreateStacktraces{}
	err := m.db.Update(func(txn *badger.Txn) error {
		for i, stacktraceKey := range stacktraceKeys {
			item, err := txn.Get([]byte(stacktraceKey))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			if err == badger.ErrKeyNotFound {
				stacktrace := r.Stacktraces[i]
				stacktrace.Id = StacktraceIDFromKey(stacktraceKey)
				b, err := stacktrace.MarshalVT()
				if err != nil {
					return err
				}
				err = txn.Set([]byte(stacktraceKey), b)
				if err != nil && err != badger.ErrTxnTooBig {
					return err
				}

				if err == badger.ErrTxnTooBig {
					// force the calling function to commit the transaction
					// we will go ahead and retry the operation
					// from where we left off
					result.retryWith = stacktraceKeys[i:]
					return nil
				}

				result.stackTraces = append(result.stackTraces, stacktrace)
				continue
			}

			err = item.Value(func(val []byte) error {
				stacktrace := &pb.Stacktrace{}
				err := stacktrace.UnmarshalVT(val)
				if err != nil {
					return err
				}

				result.stackTraces = append(result.stackTraces, stacktrace)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	return result, err
}

func (m *BadgerMetastore) Stacktraces(ctx context.Context, r *pb.StacktracesRequest) (*pb.StacktracesResponse, error) {
	res := &pb.StacktracesResponse{
		Stacktraces: make([]*pb.Stacktrace, 0, len(r.StacktraceIds)),
	}

	stacktraceKeys := make([][]byte, 0, len(r.StacktraceIds))
	for _, id := range r.StacktraceIds {
		stacktraceKeys = append(stacktraceKeys, []byte(MakeStacktraceKeyWithID(id)))
	}

	err := m.db.View(func(txn *badger.Txn) error {
		for _, stacktraceKey := range stacktraceKeys {
			item, err := txn.Get(stacktraceKey)
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				stacktrace := &pb.Stacktrace{}
				err := stacktrace.UnmarshalVT(val)
				if err != nil {
					return err
				}

				res.Stacktraces = append(res.Stacktraces, stacktrace)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return res, err
}
