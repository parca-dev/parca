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

package query

import (
	"context"
	"sort"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func GenerateTopTable(ctx context.Context, p *parcaprofile.Profile) (*pb.Top, error) {
	// Iterate over all samples and their locations.
	// Calculate the cumulative value of all locations of all samples.
	// In the end return a *pb.TopNode for each location including all the metadata we have.
	locationsTopNodes := map[string]*pb.TopNode{}
	for _, sample := range p.Samples {
		for i, location := range sample.Locations {
			if node, found := locationsTopNodes[location.ID]; found {
				node.Cumulative += sample.Value
				node.Diff += sample.DiffValue

				if i == 0 {
					node.Flat += sample.Value
				}
			} else {
				node := &pb.TopNode{
					Cumulative: sample.Value,
					Diff:       sample.DiffValue,
					Meta: &pb.TopNodeMeta{
						Mapping: location.Mapping,
						Location: &metastorev1alpha1.Location{
							Id:        location.ID,
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
						FunctionId: location.Lines[0].Function.Id,
						Line:       location.Lines[0].Line,
					}
				}
				if i == 0 {
					node.Flat = sample.Value
					node.Diff = sample.DiffValue
				}
				locationsTopNodes[location.ID] = node
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
		Unit:     p.Meta.SampleType.Unit,
	}

	return aggregateTopByFunction(top), nil
}

func aggregateTopByFunction(top *pb.Top) *pb.Top {
	aggregatesAddresses := map[string]map[uint64]*pb.TopNode{}
	aggregatesFunctions := map[string]*pb.TopNode{}

	for _, n := range top.List {
		if n.Meta == nil {
			// Ignore nodes without metadata.
			continue
		}

		var mappingID string
		if n.Meta.Mapping != nil {
			mappingID = n.Meta.Mapping.Id
		}

		// Initialize maps for given mappingID
		mapping := string(mappingID)
		if aggregatesAddresses[mapping] == nil {
			aggregatesAddresses[mapping] = map[uint64]*pb.TopNode{}
		}

		if n.Meta.GetFunction() == nil {
			// If there is no function we aggregate by address.
			addr := n.Meta.Location.Address
			if aggregateNode, exists := aggregatesAddresses[mapping][addr]; exists {
				aggregateNode.Cumulative += n.Cumulative
				aggregateNode.Diff += n.Diff
				aggregateNode.Flat += n.Flat
			} else {
				aggregatesAddresses[mapping][addr] = n
			}
			continue
		}
		// Finally, if there's a function name we aggregate by their name.
		name := n.Meta.Function.Name
		if aggregateNode, exists := aggregatesFunctions[name]; exists {
			aggregateNode.Cumulative += n.Cumulative
			aggregateNode.Diff += n.Diff
			aggregateNode.Flat += n.Flat
			if aggregateNode.Meta.Mapping != nil && n.Meta.Mapping != nil && aggregateNode.Meta.Mapping.Id != n.Meta.Mapping.Id {
				aggregateNode.Meta.Mapping = &metastorev1alpha1.Mapping{}
			}
			aggregateNode.Meta.Line = nil
		} else {
			aggregatesFunctions[name] = n
		}
	}

	var count uint64
	for _, addrs := range aggregatesAddresses {
		count += uint64(len(addrs))
	}
	count += uint64(len(aggregatesFunctions))

	list := make([]*pb.TopNode, 0, count)
	for _, addrs := range aggregatesAddresses {
		for _, n := range addrs {
			list = append(list, n)
		}
	}
	for _, n := range aggregatesFunctions {
		list = append(list, n)
	}

	// Sort the list
	sort.Slice(list, func(i, j int) bool {
		// If flat value is equal, sort by address, ascending
		if list[i].Flat == list[j].Flat {
			return list[i].Meta.Location.Address < list[j].Meta.Location.Address
		}

		// Sort by flat value, descending
		return list[i].Flat > list[j].Flat
	})

	return &pb.Top{
		List:     list,
		Reported: int32(len(list)),
		Total:    top.GetTotal(),
		Unit:     top.GetUnit(),
	}
}
