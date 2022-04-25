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

package parcacol

import (
	"context"
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"

	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

type Table interface {
	Schema() *dynparquet.Schema
	InsertBuffer(buf *dynparquet.Buffer) (tx uint64, err error)
}

func InsertProfileIntoTable(ctx context.Context, logger log.Logger, table Table, ls labels.Labels, prof *parcaprofile.Profile) (int, error) {
	buf, err := FlatProfileToBuffer(logger, ls, table.Schema(), prof)
	if err != nil {
		return 0, err
	}

	_, err = table.InsertBuffer(buf)
	return len(prof.FlatSamples), err
}

func FlatProfileToBuffer(logger log.Logger, ls labels.Labels, schema *dynparquet.Schema, prof *parcaprofile.Profile) (*dynparquet.Buffer, error) {
	// TODO all of this should be done in the flat profile
	// extraction in the first place. Also this `__name__` hack is
	// only here for backward compatibility while we finish up the
	// columnstore. This can be removed once the migration is
	// complete and the old storage is removed.
	lbls := make(labels.Labels, 0, len(ls)+1)
	found := false
	for _, l := range ls {
		if l.Name == "__name__" {
			found = true
			lbls = append(lbls, labels.Label{
				Name:  "__name__",
				Value: l.Value + "_" + prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
			})
			continue
		}
		lbls = append(lbls, labels.Label{
			Name:  l.Name,
			Value: l.Value,
		})
	}
	if !found {
		lbls = append(lbls, labels.Label{
			Name:  "__name__",
			Value: prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
		})
	}
	sort.Sort(lbls)

	rows := make(Samples, 0, len(prof.FlatSamples))
	for _, s := range prof.FlatSamples {
		pprofLabels := make(map[string]string, len(s.Label))
		for name, values := range s.Label {
			if len(values) != 1 {
				panic("expected exactly one value per pprof label")
			}
			pprofLabels[name] = values[0]
		}
		pprofNumLabels := make(map[string]int64, len(s.NumLabel))
		for name, values := range s.NumLabel {
			if len(values) != 1 {
				panic("expected exactly one value per pprof num label")
			}
			pprofNumLabels[name] = values[0]
		}

		rows = append(rows, Sample{
			SampleType:     prof.Meta.SampleType.Type,
			SampleUnit:     prof.Meta.SampleType.Unit,
			PeriodType:     prof.Meta.PeriodType.Type,
			PeriodUnit:     prof.Meta.PeriodType.Unit,
			PprofLabels:    pprofLabels,
			PprofNumLabels: pprofNumLabels,
			Labels:         lbls,
			Stacktrace:     extractLocationIDs(s.Location),
			Timestamp:      prof.Meta.Timestamp,
			Duration:       prof.Meta.Duration,
			Period:         prof.Meta.Period,
			Value:          s.Value,
		})
	}

	level.Debug(logger).Log("msg", "writing sample", "label_set", ls.String(), "timestamp", prof.Meta.Timestamp)

	buf, err := rows.ToBuffer(schema)
	if err != nil {
		return nil, err
	}

	buf.Sort()

	// This is necessary because sorting a buffer makes concurrent reading not
	// safe as the internal pages are cyclically sorted at read time. Cloning
	// executes the cyclic sort once and makes the resulting buffer safe for
	// concurrent reading as it no longer has to perform the cyclic sorting at
	// read time. This should probably be improved in the parquet library.
	buf, err = buf.Clone()
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func extractLocationIDs(locs []*metastore.Location) []byte {
	b := make([]byte, len(locs)*16) // UUID are 16 bytes thus multiply by 16
	index := 0
	for i := len(locs) - 1; i >= 0; i-- {
		copy(b[index:index+16], locs[i].ID[:])
		index += 16
	}
	return b
}
