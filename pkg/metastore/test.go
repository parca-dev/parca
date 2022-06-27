package metastore

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

type Testing interface {
	require.TestingT
	Helper()
}

func NewTestMetastore(
	t Testing,
	logger log.Logger,
	reg prometheus.Registerer,
	tracer trace.Tracer,
) pb.MetastoreServiceServer {
	t.Helper()
	return NewBadgerMetastore(
		logger,
		reg,
		tracer,
	)
}
