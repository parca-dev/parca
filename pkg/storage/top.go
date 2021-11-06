package storage

import (
	"context"
	"errors"
	"fmt"
	"sort"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"go.opentelemetry.io/otel/trace"
)

// GenerateTop creates a list of top nodes by flat value
func GenerateTop(ctx context.Context, tracer trace.Tracer, locations Locations, p InstantProfile) (*pb.Top, error) {
	fgCtx, fgSpan := tracer.Start(ctx, "generate-top")
	defer fgSpan.End()

	_, copySpan := tracer.Start(fgCtx, "copy-profile-tree")
	meta := p.ProfileMeta()
	pt := CopyInstantProfileTree(p.ProfileTree())
	copySpan.End()

	locs, err := getLocations(fgCtx, tracer, locations, pt)
	if err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}

	_, buildSpan := tracer.Start(fgCtx, "build-top")
	defer buildSpan.End()
	it := pt.Iterator()

	if !it.HasMore() || !it.NextChild() {
		return nil, nil
	}

	n := it.At()
	loc := n.LocationID()
	if loc != uint64(0) {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	rootNode := &pb.FlamegraphNode{}

	flamegraph := &pb.Flamegraph{
		Root: &pb.FlamegraphRootNode{},
		Unit: meta.SampleType.Unit,
	}
	top := &pb.Top{
		List: []*pb.TopNode{},
	}

	flamegraphStack := TreeStack{{node: rootNode}}
	steppedInto := it.StepInto()
	if !steppedInto {
		return top, nil
	}
	flamegraph.Height = int32(1)

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()
			id := child.LocationID()
			l, found := locs[id]
			if !found {
				return nil, fmt.Errorf("could not find location with ID %d", id)
			}

			outerMost, innerMost := locationToTreeNodes(l, 0, 0)

			flamegraphStack.Peek().node.Children = append(flamegraphStack.Peek().node.Children, outerMost)
			flamegraphStack.Push(&TreeStackEntry{
				node: innerMost,
			})
			if int32(len(flamegraphStack)) > flamegraph.Height {
				flamegraph.Height = int32(len(flamegraphStack))
			}

			for _, n := range child.FlatValues() {
				if n.Value == 0 {
					continue
				}
				for _, entry := range flamegraphStack {
					entry.node.Cumulative += n.Value
				}
				innerMost.Flat += n.Value
			}
			for _, n := range child.FlatDiffValues() {
				if n.Value == 0 {
					continue
				}
				for _, entry := range flamegraphStack {
					entry.node.Diff += n.Value
				}
			}

			it.StepInto()
			continue
		}

		it.StepUp()
		entry, _ := flamegraphStack.Pop()
		if entry != nil {
			top.List = append(top.List, &pb.TopNode{
				Meta:       (*pb.TopNodeMeta)(entry.node.Meta),
				Cumulative: entry.node.Cumulative,
				Flat:       entry.node.Flat,
			})
		}
	}

	// TODO
	top.Total = int32(rootNode.Cumulative)
	top.Reported = int32(rootNode.Cumulative)
	sort.Slice(top.List, func(i, j int) bool {
		return top.List[i].Flat > top.List[j].Flat
	})

	// TODO: aggregate by function?
	return top, nil
}
