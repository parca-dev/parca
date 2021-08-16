package storage

import (
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestInMemoryMetaStore(t *testing.T) {
	s := NewInMemoryProfileMetaStore()
	l := &profile.Location{
		ID:      uint64(8),
		Address: uint64(42),
	}
	s.CreateLocation(l)
	require.Equal(t, uint64(1), l.ID)
	_, err := s.GetLocationByID(l.ID)
	require.NoError(t, err)
	_, err = s.GetLocationByKey(MakeLocationKey(l))
	require.NoError(t, err)
}
