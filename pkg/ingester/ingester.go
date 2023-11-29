package ingester

import (
	"context"

	"github.com/parca-dev/parca/pkg/normalizer"
)

type Ingester interface {
	Ingest(ctx context.Context, req normalizer.NormalizedWriteRawRequest) error
	Close() error
}
