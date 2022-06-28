package metastoretest

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
)

type Testing interface {
	require.TestingT
	Helper()
	Name() string
}

func NewTestMetastore(
	t Testing,
	logger log.Logger,
	reg prometheus.Registerer,
	tracer trace.Tracer,
) pb.MetastoreServiceServer {
	t.Helper()
	return metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
	)
}
