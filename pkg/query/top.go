package query

import (
	"context"
	"fmt"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateTopTable(ctx context.Context, metaStore metastore.ProfileMetaStore, p profile.InstantProfile) (*pb.Top, error) {
	samples := p.Samples()

	sampleUUIDs := make([][]byte, 0, len(samples))
	for id := range samples {
		sampleUUIDs = append(sampleUUIDs, []byte(id))
	}

	sampleMap, err := metaStore.GetStacktraceByIDs(ctx, sampleUUIDs...)
	if err != nil {
		return nil, err
	}

	locationUUIDSeen := map[string]struct{}{}
	locationUUIDs := [][]byte{}
	for _, s := range sampleMap {
		for _, id := range s.GetLocationIds() {
			if _, seen := locationUUIDSeen[string(id)]; !seen {
				locationUUIDSeen[string(id)] = struct{}{}
				locationUUIDs = append(locationUUIDs, id)
			}
		}
	}

	// Get the full locations for the location UUIDs
	locationsMap, err := metastore.GetLocationsByIDs(ctx, metaStore, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	functionMap := map[string]*pb.TopNode{}
	for sampleUUID, sample := range samples {
		s := sampleMap[sampleUUID]

		location := locationsMap[string(s.GetLocationIds()[0])]

		meta := &pb.TopNodeMeta{
			Mapping: location.Mapping,
			Location: &metastorev1alpha1.Location{
				Id:        []byte(location.ID.String()),
				Address:   location.Address,
				MappingId: location.Mapping.GetId(),
				IsFolded:  location.IsFolded,
			},
		}
		if len(location.Lines) > 0 {
			// TODO: Return or merge multiple lines for samples
			meta.Function = location.Lines[0].Function
			meta.Line = &metastorev1alpha1.Line{
				FunctionId: meta.GetFunction().GetId(),
				Line:       location.Lines[0].Line,
			}
		}

		if sample.Value  == 0 {
			continue
		}

		if f, exists:= functionMap[meta.Function.Name]; exists {
			f.Flat += sample.Value
		} else {
			functionMap[meta.Function.Name] = &pb.TopNode{
				Meta: meta,
				Cumulative: 0,
				Flat: sample.Value,
			}
		}
	}

	list := make([]*pb.TopNode, 0, len(functionMap))
	for _, f := range functionMap {
		list = append(list, f)
	}

	return &pb.Top{
		List:     list,
		Reported: int32(len(list)),
		Total:    int32(len(list)),
	}, nil
}
