package query

import (
	"context"
	"fmt"
	"sort"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/go-kit/log"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/columnstore"
	"github.com/parca-dev/parca/pkg/metastore"
	"go.opentelemetry.io/otel/trace"
)

// ColumnQuery is the read api interface for parca
// It implements the proto/query/query.proto APIServer interface
type ColumnQueryAPI struct {
	pb.UnimplementedQueryServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	table     *columnstore.Table
	metaStore metastore.ProfileMetaStore
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
	table *columnstore.Table,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:    logger,
		tracer:    tracer,
		table:     table,
		metaStore: metaStore,
	}
}

// Labels issues a labels request against the storage
func (q *ColumnQueryAPI) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	return &pb.LabelsResponse{
		LabelNames: nil,
	}, nil
}

// Values issues a values request against the storage
func (q *ColumnQueryAPI) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	name := req.LabelName
	vals := []string{}
	seen := map[string]struct{}{}

	pool := memory.NewGoAllocator()
	err := q.table.Iterator(pool, columnstore.Distinct(pool, []columnstore.ArrowFieldMatcher{columnstore.DynamicColumnRef("labels").Column(name).ArrowFieldMatcher()}, func(ar arrow.Record) error {
		defer ar.Release()

		if ar.NumCols() != 1 {
			return fmt.Errorf("expected 1 column, got %d", ar.NumCols())
		}

		col := ar.Column(0)
		stringCol, ok := col.(*array.String)
		if !ok {
			return fmt.Errorf("expected string column, got %T", col)
		}

		for i := 0; i < stringCol.Len(); i++ {
			val := stringCol.Value(i)
			if _, ok := seen[val]; !ok {
				vals = append(vals, val)
				seen[val] = struct{}{}
			}
		}

		return nil
	}).Callback)
	if err != nil {
		return nil, err
	}

	sort.Strings(vals)

	return &pb.ValuesResponse{
		LabelValues: vals,
	}, nil
}
