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
	"errors"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"

	pprofproto "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	"github.com/parca-dev/parca/pkg/profile"
)

type Table interface {
	Schema() *dynparquet.Schema
	InsertBuffer(context.Context, *dynparquet.Buffer) (tx uint64, err error)
}

type Ingester struct {
	logger     log.Logger
	table      Table
	normalizer *Normalizer
	schema     *dynparquet.Schema
}

func NewIngester(logger log.Logger, normalizer *Normalizer, table Table) *Ingester {
	return &Ingester{
		logger:     logger,
		normalizer: normalizer,
		table:      table,
		schema:     Schema(),
	}
}

var ErrMissingNameLabel = errors.New("missing __name__ label")

func separateNameFromLabels(ls labels.Labels) (string, labels.Labels, error) {
	out := make(labels.Labels, 0, len(ls))
	name := ""
	for _, l := range ls {
		if l.Name == "__name__" {
			name = l.Value
		} else {
			out = append(out, l)
		}
	}

	if name == "" {
		return "", nil, ErrMissingNameLabel
	}

	return name, out, nil
}

func (ing Ingester) Ingest(ctx context.Context, ls labels.Labels, p *pprofproto.Profile, normalized bool) error {
	name, ls, err := separateNameFromLabels(ls)
	if err != nil {
		return err
	}

	normalizedProfiles, err := ing.normalizer.NormalizePprof(ctx, name, p, normalized)
	if err != nil {
		return err
	}

	for _, p := range normalizedProfiles {
		if len(p.Samples) == 0 {
			level.Debug(ing.logger).Log("msg", "no samples found in profile, dropping it", "name", p.Meta.Name, "sample_type", p.Meta.SampleType.Type, "sample_unit", p.Meta.SampleType.Unit, "labels", ls)
			continue
		}

		if err := ing.IngestProfile(ctx, ls, p); err != nil {
			return err
		}
	}

	return nil
}

func (ing Ingester) IngestProfile(ctx context.Context, ls labels.Labels, p *profile.NormalizedProfile) error {

	buffer, err := NormalizedProfileToParquetBuffer(ing.schema, ls, p)
	if err != nil {
		return fmt.Errorf("failed to convert samples to buffer: %w", err)
	}

	buffer.Sort()

	// This is necessary because sorting a buffer makes concurrent reading not
	// safe as the internal pages are cyclically sorted at read time. Cloning
	// executes the cyclic sort once and makes the resulting buffer safe for
	// concurrent reading as it no longer has to perform the cyclic sorting at
	// read time. This should probably be improved in the parquet library.
	buffer, err = buffer.Clone()
	if err != nil {
		return err
	}

	_, err = ing.table.InsertBuffer(ctx, buffer)
	if err != nil {
		return fmt.Errorf("failed to insert buffer: %w", err)
	}

	return nil
}
