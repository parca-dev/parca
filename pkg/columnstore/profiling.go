package columnstore

import (
	"context"
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/prometheus/model/labels"
)

func ParcaProfilingTableSchema() Schema {
	return NewSchema(
		[]ColumnDefinition{{
			Name:     "sample_type",
			Type:     StringType,
			Encoding: PlainEncoding,
		}, {
			Name:     "sample_unit",
			Type:     StringType,
			Encoding: PlainEncoding,
		}, {
			Name:     "period_type",
			Type:     StringType,
			Encoding: PlainEncoding,
		}, {
			Name:     "period_unit",
			Type:     StringType,
			Encoding: PlainEncoding,
		}, {
			Name:     "labels",
			Type:     StringType,
			Encoding: PlainEncoding,
			Dynamic:  true,
		}, {
			Name: "stacktrace",
			// This should be a UUID, but we don't have a UUID type yet. For
			// now, we'll just use a string. UUIDs might also be best
			// represented as a Uint128 internally.
			Type:     List(UUIDType),
			Encoding: PlainEncoding,
		}, {
			Name:     "timestamp",
			Type:     Int64Type,
			Encoding: PlainEncoding,
			// TODO
			//}, {
			//	Name:     "pprof_labels",
			//	Type:     StringType,
			//	Encoding: PlainEncoding,
			//}, {
			//	Name:     "pprof_num_labels",
			//	Type:     Int64Type,
			//	Encoding: PlainEncoding,
		}, {
			Name:     "duration",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "period",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "value",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}},
		8192, // 2^13
	)
}

func InsertProfileIntoTable(ctx context.Context, logger log.Logger, table *Table, ls labels.Labels, prof *parcaprofile.FlatProfile) (int, error) {
	// TODO all of this should be done in the flat profile
	// extraction in the first place. Also this `__name__` hack is
	// only here for backward compatibility while we finish up the
	// columnstore. This can be removed once the migration is
	// complete and the old storage is removed.
	labels := make([]DynamicColumnValue, 0, len(ls))
	found := false
	for _, l := range ls {
		if l.Name == "__name__" {
			found = true
			labels = append(labels, DynamicColumnValue{
				Name:  "__name__",
				Value: l.Value + "_" + prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
			})
			continue
		}
		labels = append(labels, DynamicColumnValue{
			Name:  l.Name,
			Value: l.Value,
		})
	}
	if !found {
		labels = append(labels, DynamicColumnValue{
			Name:  "__name__",
			Value: prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
		})
	}

	rows := make([]*SampleRow, 0, len(prof.FlatSamples))
	for _, s := range prof.FlatSamples {
		rows = append(rows, &SampleRow{
			Stacktrace: metastoreLocationsToSampleStacktrace(s.Location),
			Value:      s.Value,
		})
	}

	level.Debug(logger).Log("msg", "writing sample", "label_set", ls.String(), "timestamp", prof.Meta.Timestamp)

	SortSampleRows(rows)
	return len(rows), table.Insert(makeRows(prof, labels, rows))
}

func makeRows(prof *parcaprofile.FlatProfile, labels []DynamicColumnValue, rows []*SampleRow) []Row {
	res := make([]Row, len(rows))
	for i, r := range rows {
		res[i] = Row{
			Values: []interface{}{
				prof.Meta.SampleType.Type,
				prof.Meta.SampleType.Unit,
				prof.Meta.PeriodType.Type,
				prof.Meta.PeriodType.Unit,
				labels,
				r.Stacktrace,
				prof.Meta.Timestamp,
				prof.Meta.Duration,
				prof.Meta.Period,
				r.Value,
			},
		}
	}

	return res
}

func metastoreLocationsToSampleStacktrace(locs []*metastore.Location) []UUID {
	length := len(locs) - 1
	stacktrace := make([]UUID, length+1)
	for i := range locs {
		cUUID := UUID(locs[length-i].ID)
		stacktrace[i] = cUUID
	}

	return stacktrace
}

type SampleRow struct {
	// Array of Location IDs.
	Stacktrace []UUID

	PprofStringLabels  map[string]string
	PprofNumLabels     map[string]int64
	PprofNumLabelUnits map[string]string

	Value int64
}

func SortSampleRows(samples []*SampleRow) {
	sort.Slice(samples, func(i, j int) bool {
		// TODO need to take labels into account
		return UUIDType.ListLess(samples[i].Stacktrace, samples[j].Stacktrace)
	})
}
