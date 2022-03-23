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

	"github.com/google/uuid"
	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func GenerateTopTable(ctx context.Context, metaStore metastore.ProfileMetaStore, p *parcaprofile.StacktraceSamples) (*pb.Top, error) {
	// Iterate over all samples and their locations.
	// Calculate the cumulative value of all locations of all samples.
	// In the end return a *pb.TopNode for each location including all the metadata we have.
	locationsTopNodes := map[uuid.UUID]*pb.TopNode{}
	for _, sample := range p.Samples {
		// If values are zero we can simply ignore the samples.
		// They wouldn't show up on their own in the table anyway.
		if sample.Value == 0 {
			continue
		}

		for i, location := range sample.Location {
			if node, found := locationsTopNodes[location.ID]; found {
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
	aggregatesFunctions := map[string]map[string]*pb.TopNode{}

	for _, n := range top.GetList() {
		if n.GetMeta() == nil {
			// Ignore nodes without metadata.
			continue
		}

		var mappingID []byte
		if n.Meta.GetMapping() != nil {
			mappingID = n.Meta.Mapping.GetId()
		}

		// Initialize maps for given mappingID
		mapping := string(mappingID)
		if aggregatesAddresses[mapping] == nil {
			aggregatesAddresses[mapping] = map[uint64]*pb.TopNode{}
			aggregatesFunctions[mapping] = map[string]*pb.TopNode{}
		}

		if n.Meta.GetFunction() == nil {
			// If there is no function we aggregate by address.
			addr := n.Meta.GetLocation().GetAddress()
			if aggregateNode, exists := aggregatesAddresses[mapping][addr]; exists {
				aggregateNode.Cumulative += n.Cumulative
				aggregateNode.Flat += n.Flat
			} else {
				aggregatesAddresses[mapping][addr] = n
			}
			continue
		}
		// Finally, if there's a function name we aggregated by their name.
		name := n.Meta.Function.GetName()
		if aggregateNode, exists := aggregatesFunctions[mapping][name]; exists {
			aggregateNode.Cumulative += n.Cumulative
			aggregateNode.Flat += n.Flat
		} else {
			aggregatesFunctions[mapping][name] = n
		}
	}

	var count uint64
	for _, addrs := range aggregatesAddresses {
		count += uint64(len(addrs))
	}
	for _, funcs := range aggregatesFunctions {
		count += uint64(len(funcs))
	}

	list := make([]*pb.TopNode, 0, count)
	for _, addrs := range aggregatesAddresses {
		for _, n := range addrs {
			list = append(list, n)
		}
	}
	for _, funcs := range aggregatesFunctions {
		for _, n := range funcs {
			list = append(list, n)
		}
	}

	return &pb.Top{
		List:     list,
		Reported: int32(len(list)),
		Total:    top.GetTotal(),
		Unit:     top.GetUnit(),
	}
}
