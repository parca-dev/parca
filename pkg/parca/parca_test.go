package parca

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-kit/log"
	pb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func Benchmark_Parca_WriteRaw(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//logger := log.NewNopLogger()
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Run(ctx, logger, "testdata/parca.yaml", ":9090")
		if !errors.Is(err, context.Canceled) {
			require.NoError(b, err)
		}
	}()

	var conn grpc.ClientConnInterface
	backoff.Retry(func() error {
		var err error
		conn, err = grpc.Dial(":9090", grpc.WithInsecure())
		if err != nil {
			return err
		}

		client := pb.NewProfileStoreClient(conn)
		_, err = client.WriteRaw(ctx, &pb.WriteRawRequest{})
		return err
	}, backoff.NewConstantBackOff(time.Second))

	client := pb.NewProfileStoreClient(conn)

	// Benchamrk section
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.WriteRaw(ctx, &pb.WriteRawRequest{})
		require.NoError(b, err)
	}
	b.StopTimer()

	cancel()
	<-done
}
