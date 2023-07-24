package parcacol

import (
	"fmt"
	"testing"

	"github.com/apache/arrow/go/v13/arrow/memory"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestBuildArrowLocations(t *testing.T) {
	stacktraces := []*pb.Stacktrace{{
		LocationIds: []string{"1"},
	}, {
		LocationIds: []string{"2"},
	}}
	locations := []*profile.Location{{
		ID:      "1",
		Address: 0x1,
		Mapping: &pb.Mapping{
			Id:      "1",
			BuildId: "1",
		},
		Lines: []profile.LocationLine{{
			Line: 1,
			Function: &pb.Function{
				Id:   "1",
				Name: "main",
			},
		}},
	}, {
		ID:      "2",
		Address: 0x1,
		Mapping: &pb.Mapping{
			Id:      "2",
			BuildId: "2",
		},
	}}
	locationIndex := map[string]int{"1": 0, "2": 1}

	r := buildArrowLocations(memory.DefaultAllocator, stacktraces, locations, locationIndex)
	fmt.Println(r)
}
