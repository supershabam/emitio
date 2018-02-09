package pkg

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	ctx := context.Background()
	s, err := NewSQLiteStore(ctx, "memory://")
	require.Nil(t, err)
	require.NotNil(t, s)
	t0 := time.Now()
	s.Write(ctx, "test", []Record{
		{
			At:   time.Unix(0, t0.UnixNano()),
			Blob: []byte("first"),
			Seq:  1,
		},
	})
	records := []Record{}
	ch, wait := s.Read(ctx, "test", 0, math.MaxInt64, 24)
	for batch := range ch {
		records = append(records, batch...)
	}
	err = wait()
	require.Nil(t, err)
	require.Len(t, records, 1)
	require.Equal(t, t0.UnixNano(), records[0].At.UnixNano())
	require.Equal(t, int64(1), records[0].Seq)
	require.Equal(t, []byte("first"), records[0].Blob)
}
