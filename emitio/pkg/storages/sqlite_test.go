package storages

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/supershabam/emitio/emitio/pkg"
)

func TestBlock(t *testing.T) {
	ctx := context.Background()
	path, err := ioutil.TempDir("", "emitio")
	require.Nil(t, err)
	file := fmt.Sprintf("%s/db.sq3", path)
	b, err := newBlock(ctx, file, 24)
	require.Nil(t, err)
	t0 := time.Now()
	err = b.write(ctx, []pkg.Record{
		{
			At:   t0,
			Blob: []byte("greetings"),
		},
	})
	require.Nil(t, err)
	reqs, err := b.read(ctx, 0, math.MaxInt64, 10)
	require.Nil(t, err)
	require.Len(t, reqs, 1)
	require.Equal(t, int64(24), reqs[0].Seq)
	b, err = openBlock(ctx, file)
	require.Nil(t, err)
	require.Equal(t, int64(1), b.count)
	require.Equal(t, t0.UnixNano(), b.lastAt.UnixNano())
	require.Equal(t, int64(24), b.lastSeq)
	reqs, err = b.read(ctx, 0, math.MaxInt64, 10)
	require.Nil(t, err)
	require.Len(t, reqs, 1)
	require.Equal(t, int64(24), reqs[0].Seq)
}

func TestSQLite(t *testing.T) {
	ctx := context.Background()
	s, err := NewSQLite(ctx)
	require.Nil(t, err)
	t0 := time.Now()
	err = s.Write(ctx, "one", []pkg.Record{
		{
			At:   t0,
			Blob: []byte("greetings"),
		},
	})
	require.Nil(t, err)
	err = s.Write(ctx, "two", []pkg.Record{
		{
			At:   t0,
			Blob: []byte("a"),
		},
		{
			At:   t0,
			Blob: []byte("b"),
		},
	})
	require.Nil(t, err)
	records := []pkg.SeqRecord{}
	ch, wait := s.Read(ctx, "one", 0, math.MaxInt64, 1e3)
	for batch := range ch {
		records = append(records, batch...)
	}
	err = wait()
	require.Nil(t, err)
	require.Len(t, records, 1)
	records = []pkg.SeqRecord{}
	ch, wait = s.Read(ctx, "two", 0, math.MaxInt64, 1e3)
	for batch := range ch {
		records = append(records, batch...)
	}
	err = wait()
	require.Nil(t, err)
	require.Len(t, records, 2)
	records = []pkg.SeqRecord{}
	ch, wait = s.Read(ctx, "nonexistent", 0, math.MaxInt64, 1e3)
	for batch := range ch {
		records = append(records, batch...)
	}
	err = wait()
	require.Nil(t, err)
	require.Len(t, records, 0)
}
