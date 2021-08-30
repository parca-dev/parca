package parca

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-kit/log"
	pb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func benchmarkSetup(ctx context.Context, b *testing.B) (pb.ProfileStoreClient, <-chan struct{}) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Run(ctx, logger, reg, &Flags{ConfigPath: "testdata/parca.yaml", Port: ":9090"})
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
	return client, done
}

func Benchmark_Parca_WriteRaw(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, done := benchmarkSetup(ctx, b)

	f, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	// Benchamrk section
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.WriteRaw(ctx, &pb.WriteRawRequest{
			Series: []*pb.RawProfileSeries{
				{
					Labels: &pb.LabelSet{
						Labels: []*pb.Label{
							{
								Name:  "test",
								Value: b.Name(),
							},
						},
					},
					Samples: []*pb.RawSample{
						{
							RawProfile: f,
						},
					},
				},
			},
		})
		require.NoError(b, err)
	}
	b.StopTimer()

	cancel()
	<-done
}
