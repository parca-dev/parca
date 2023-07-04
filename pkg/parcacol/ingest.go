// Copyright 2022-2023 The Parca Authors
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/klauspost/compress/gzip"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/segmentio/parquet-go"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

var ExperimentalArrow bool

var ErrMissingNameLabel = errors.New("missing __name__ label")

type Table interface {
	Schema() *dynparquet.Schema
	Insert(context.Context, []byte) (tx uint64, err error)
	InsertRecord(context.Context, arrow.Record) (tx uint64, err error)
}

type NormalizedIngester struct {
	logger     log.Logger
	table      Table
	schema     *dynparquet.Schema
	bufferPool *sync.Pool

	allLabelNames         []string
	allPprofLabelNames    []string
	allPprofNumLabelNames []string
}

func NewNormalizedIngester(
	logger log.Logger,
	table Table,
	schema *dynparquet.Schema,
	bufferPool *sync.Pool,
	allLabelNames []string,
	allPprofLabelNames []string,
	allPprofNumLabelNames []string,
) NormalizedIngester {
	return NormalizedIngester{
		logger:     logger,
		table:      table,
		schema:     schema,
		bufferPool: bufferPool,

		allLabelNames:         allLabelNames,
		allPprofLabelNames:    allPprofLabelNames,
		allPprofNumLabelNames: allPprofNumLabelNames,
	}
}

type Series struct {
	Labels  map[string]string
	Samples [][]*profile.NormalizedProfile
}

func (ing NormalizedIngester) Ingest(ctx context.Context, series []Series) error {
	pBuf, err := ing.schema.GetBuffer(map[string][]string{
		ColumnLabels:         ing.allLabelNames,
		ColumnPprofLabels:    ing.allPprofLabelNames,
		ColumnPprofNumLabels: ing.allPprofNumLabelNames,
	})
	if err != nil {
		return err
	}
	defer ing.schema.PutBuffer(pBuf)

	var r parquet.Row
	for _, s := range series {
		for _, normalizedProfiles := range s.Samples {
			for _, p := range normalizedProfiles {
				if len(p.Samples) == 0 {
					ls := labels.FromMap(s.Labels)
					level.Debug(ing.logger).Log("msg", "no samples found in profile, dropping it", "name", p.Meta.Name, "sample_type", p.Meta.SampleType.Type, "sample_unit", p.Meta.SampleType.Unit, "labels", ls)
					continue
				}

				for _, profileSample := range p.Samples {
					r = SampleToParquetRow(
						ing.schema,
						r[:0],
						ing.allLabelNames,
						ing.allPprofLabelNames,
						ing.allPprofNumLabelNames,
						s.Labels,
						p.Meta,
						profileSample,
					)
					_, err := pBuf.WriteRows([]parquet.Row{r})
					if err != nil {
						return err
					}
				}
			}
		}
	}

	pBuf.Sort()

	// Experimental feature that ingests profiles as arrow records.
	if ExperimentalArrow {
		// Read sorted rows into an arrow record
		records, err := ParquetBufToArrowRecord(ctx, pBuf.Buffer, 0)
		if err != nil {
			return err
		}
		defer func() {
			for _, record := range records {
				record.Release()
			}
		}()

		for _, record := range records {
			if record.NumRows() == 0 {
				return nil
			}

			if _, err := ing.table.InsertRecord(ctx, record); err != nil {
				return err
			}
		}
		return nil
	}

	buf := ing.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer ing.bufferPool.Put(buf)

	if err := ing.schema.SerializeBuffer(buf, pBuf.Buffer); err != nil {
		return err
	}

	if _, err := ing.table.Insert(ctx, buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// NormalizedIngest normalizes and persists pprof samples
// (mappings, functions, locations, stack traces).
// Note, normalization is used in broad terms (think db normalization),
// it doesn't necessarily mean address normalization (PIE).
func NormalizedIngest(
	ctx context.Context,
	req *profilestorepb.WriteRawRequest,
	logger log.Logger,
	table Table,
	schema *dynparquet.Schema,
	metastore metastorepb.MetastoreServiceClient,
	bufferPool *sync.Pool,
	enableAddressNormalization bool,
) error {
	normalizer := NewNormalizer(metastore, enableAddressNormalization)
	normalizedRequest, err := NormalizeWriteRawRequest(ctx, normalizer, req)
	if err != nil {
		return err
	}

	if err := NewNormalizedIngester(
		logger,
		table,
		schema,
		bufferPool,
		normalizedRequest.AllLabelNames,
		normalizedRequest.AllPprofLabelNames,
		normalizedRequest.AllPprofNumLabelNames,
	).Ingest(ctx, normalizedRequest.Series); err != nil {
		return status.Errorf(codes.Internal, "failed to create ingester: %v", err)
	}

	return nil
}

type NormalizedWriteRawRequest struct {
	Series                []Series
	AllLabelNames         []string
	AllPprofLabelNames    []string
	AllPprofNumLabelNames []string
}

type Normalizer interface {
	NormalizePprof(ctx context.Context, name string, takenLabelNames map[string]string, p *pprofpb.Profile, normalizedAddress bool) ([]*profile.NormalizedProfile, error)
}

// NormalizeWriteRawRequest normalizes the profiles
// (mappings, functions, locations, stack traces) to prepare for ingestion.
// It also validates label names of profiles' series,
// decompresses the samples, unmarshals and validates them.
func NormalizeWriteRawRequest(ctx context.Context, normalizer Normalizer, req *profilestorepb.WriteRawRequest) (NormalizedWriteRawRequest, error) {
	allLabelNames := make(map[string]struct{})
	allPprofLabelNames := make(map[string]struct{})
	allPprofNumLabelNames := make(map[string]struct{})

	series := make([]Series, 0, len(req.Series))
	for _, rawSeries := range req.Series {
		ls := make(map[string]string, len(rawSeries.Labels.Labels))
		name := ""
		for _, l := range rawSeries.Labels.Labels {
			if l.Name == model.MetricNameLabel {
				name = l.Value
				continue
			}

			if valid := model.LabelName(l.Name).IsValid(); !valid {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "invalid label name: %v", l.Name)
			}

			if _, ok := ls[l.Name]; ok {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "duplicate label name: %v", l.Name)
			}

			ls[l.Name] = l.Value
			allLabelNames[l.Name] = struct{}{}
		}

		if name == "" {
			return NormalizedWriteRawRequest{}, status.Error(codes.InvalidArgument, ErrMissingNameLabel.Error())
		}

		samples := make([][]*profile.NormalizedProfile, 0, len(rawSeries.Samples))
		for _, sample := range rawSeries.Samples {
			if len(sample.RawProfile) >= 2 && sample.RawProfile[0] == 0x1f && sample.RawProfile[1] == 0x8b {
				gz, err := gzip.NewReader(bytes.NewBuffer(sample.RawProfile))
				if err == nil {
					sample.RawProfile, err = io.ReadAll(gz)
				}
				if err != nil {
					return NormalizedWriteRawRequest{}, fmt.Errorf("decompressing profile: %v", err)
				}

				if err := gz.Close(); err != nil {
					return NormalizedWriteRawRequest{}, fmt.Errorf("close gzip reader: %v", err)
				}
			}

			p := &pprofpb.Profile{}
			if err := p.UnmarshalVT(sample.RawProfile); err != nil {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
			}

			if err := ValidatePprofProfile(p); err != nil {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
			}

			LabelNamesFromSamples(
				ls,
				p.StringTable,
				p.Sample,
				allPprofLabelNames,
				allPprofNumLabelNames,
			)

			normalizedProfiles, err := normalizer.NormalizePprof(ctx, name, ls, p, req.Normalized)
			if err != nil {
				return NormalizedWriteRawRequest{}, fmt.Errorf("normalize profile: %w", err)
			}

			samples = append(samples, normalizedProfiles)
		}

		series = append(series, Series{
			Labels:  ls,
			Samples: samples,
		})
	}

	return NormalizedWriteRawRequest{
		Series:                series,
		AllLabelNames:         sortedKeys(allLabelNames),
		AllPprofLabelNames:    sortedKeys(allPprofLabelNames),
		AllPprofNumLabelNames: sortedKeys(allPprofNumLabelNames),
	}, nil
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	out := maps.Keys(m)
	sort.Strings(out)
	return out
}
