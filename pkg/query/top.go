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

	// Iterate over all samples and their locations.
	// Calculate the cumulative value of all locations of all samples.
	// In the end return a *pb.TopNode for each location including all the metadata we have.
	locationsTopNodes := map[string]*pb.TopNode{}
	for sampleUUID, sample := range samples {
		// If values are zero we can simply ignore the samples.
		// They wouldn't show up on their own in the table anyway.
		if sample.Value == 0 {
			continue
		}

		s := sampleMap[sampleUUID]
		for i, id := range s.GetLocationIds() {
			location := locationsMap[string(id)]
			if node, found := locationsTopNodes[string(id)]; found {
				node.Cumulative += sample.Value
			} else {
				node := &pb.TopNode{
					Cumulative: sample.Value,
					Meta: &pb.TopNodeMeta{
						Mapping: location.Mapping,
						Location: &metastorev1alpha1.Location{
							Id:        []byte(location.ID.String()),
							Address:   location.Address,
							MappingId: location.Mapping.GetId(),
							IsFolded:  location.IsFolded,
						},
					},
				}
				if len(location.Lines) > 0 {
					// TODO: Return or merge multiple lines for samples
					node.Meta.Function = location.Lines[0].Function
					node.Meta.Line = &metastorev1alpha1.Line{
						FunctionId: location.Lines[0].Function.GetId(),
						Line:       location.Lines[0].Line,
					}
				}
				if i == 0 {
					node.Flat = sample.Value
				}
				locationsTopNodes[string(id)] = node
			}
		}
	}

	list := make([]*pb.TopNode, 0, len(locationsTopNodes))
	for _, f := range locationsTopNodes {
		list = append(list, f)
	}

	top := &pb.Top{
		List:     list,
		Reported: int32(len(list)),
		Total:    int32(len(list)),
		Unit:     p.ProfileMeta().SampleType.Unit,
	}

	return aggregateTopByFunction(top), nil
}

func aggregateTopByFunction(top *pb.Top) *pb.Top {
	// TODO: Maybe nest by mappingID?
	aggregatesAddresses := map[uint64]*pb.TopNode{}
	aggregatesFunctions := map[string]*pb.TopNode{}

	for _, n := range top.GetList() {
		if n.GetMeta() == nil {
			// Ignore nodes without metadata.
			continue
		}
		if n.Meta.GetFunction() == nil {
			// If there is no function we aggregate by address.
			addr := n.Meta.GetLocation().GetAddress()
			if aggregateNode, exists := aggregatesAddresses[addr]; exists {
				aggregateNode.Cumulative += n.Cumulative
				aggregateNode.Flat += n.Flat
			} else {
				aggregatesAddresses[addr] = n
			}
			continue
		}
		// Finally, if there's a function name we aggregated by their name.
		name := n.Meta.Function.GetName()
		if aggregateNode, exists := aggregatesFunctions[name]; exists {
			aggregateNode.Cumulative += n.Cumulative
			aggregateNode.Flat += n.Flat
		} else {
			aggregatesFunctions[name] = n
		}

	}

	list := make([]*pb.TopNode, 0, len(aggregatesAddresses)+len(aggregatesFunctions))
	for _, n := range aggregatesAddresses {
		list = append(list, n)
	}
	for _, n := range aggregatesFunctions {
		list = append(list, n)
	}

	return &pb.Top{
		List:     list,
		Reported: top.GetReported(),
		Total:    top.GetTotal(),
		Unit:     top.GetUnit(),
	}
}
