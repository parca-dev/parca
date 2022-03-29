package parcacol

import (
	"context"
	"fmt"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/profile"
)

func ArrowRecordToStacktraceSamples(
	ctx context.Context,
	metaStore metastore.ProfileMetaStore,
	ar arrow.Record,
	valueColumnName string,
) (*profile.StacktraceSamples, error) {
	// sample is an intermediate representation used before
	// we actually have the profile.Sample assembled from the metastore.
	type sample struct {
		locationIDs [][]byte
		value       int64
	}

	schema := ar.Schema()
	indices := schema.FieldIndices("stacktrace")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one stacktrace column, got %d", len(indices))
	}
	stacktraceColumn := ar.Column(indices[0]).(*array.String)

	indices = schema.FieldIndices("sum(value)")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one value column, got %d", len(indices))
	}
	valueColumn := ar.Column(indices[0]).(*array.Int64)

	locationUUIDSeen := map[string]struct{}{}
	locationUUIDs := [][]byte{}
	rows := int(ar.NumRows())
	samples := make([]sample, rows)
	for i := 0; i < rows; i++ {
		s := sample{
			value: valueColumn.Value(i),
		}

		uuids := stacktraceColumn.Value(i)
		if len(uuids)%16 != 0 {
			return nil, fmt.Errorf("expected stacktrace uuids to be multiple of 16 bytes")
		}

		// We split the uuids into 16 byte pieces which are exactly one uuid.
		for i := 0; i < len(uuids); i += 16 {
			u := []byte(uuids[i : i+16])
			s.locationIDs = append(s.locationIDs, u)

			if _, seen := locationUUIDSeen[string(u)]; !seen {
				locationUUIDSeen[string(u)] = struct{}{}
				locationUUIDs = append(locationUUIDs, u)
			}
		}

		samples[i] = s
	}

	// Get the full locations for the location UUIDs
	locationsMap, err := metastore.GetLocationsByIDs(ctx, metaStore, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	stackSamples := make([]*profile.Sample, 0, len(samples))
	for _, s := range samples {
		stackSample := &profile.Sample{
			Value:    s.value,
			Location: make([]*metastore.Location, 0, len(s.locationIDs)),
		}

		// LocationIDs are stored in the opposite order than the flamegraph
		// builder expects, so we need to iterate over them in reverse.
		for i := len(s.locationIDs) - 1; i >= 0; i-- {
			locID := s.locationIDs[i]
			stackSample.Location = append(stackSample.Location, locationsMap[string(locID)])
		}

		stackSamples = append(stackSamples, stackSample)
	}

	return &profile.StacktraceSamples{
		Samples: stackSamples,
	}, nil
}
