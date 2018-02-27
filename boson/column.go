package boson

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type Float64Column struct {
	Index  []uint32
	Values []float64
}

type StringColumn struct {
	Index  []uint32
	Values []string
}

type Page struct {
	Index          []time.Time
	Float64Columns map[string]Float64Column
	StringColumns  map[string]StringColumn
}

type Predi interface {
	Key() string
	MatchFloat64(float64) bool
	MatchString(string) bool
}

type matchIDs func(acc, local []uint32, values []float64) []uint32

type filter func(ctx context.Context, in <-chan []uint32) (<-chan []uint32, Wait)

func makeFloat64ColumnFilter(source, field string, match func(id, column uint32, value float64) bool) filter {
	return func(ctx context.Context, d Database, in <-chan []uint32) (<-chan []uint32, Wait) {
		ch := make(chan []uint32)
		eg, ctx := errgroup.WithContext(ctx)
		eg.Go(func() error {
			defer close(ch)
			for {
				select {
				case <-ctx.Done():
					return nil
				case ids, active := <-in:
					if !active {
						return nil
					}
					if len(ids) == 0 {
						continue
					}
					next := make([]uint32, 0, len(ids))
					columns, values := d.Float64Columns(source, field, ids)
					for idx, cdx := 0; idx < len(ids) && cdx < len(columns); idx++ {
						id := ids[idx]
						column := columns[cdx]
						if id != column {
							if match(id, 0, 0) {
								next = append(next, id)
							}
							continue
						}
						if match(id, column, value[cdx]) {
							next = append(next, id)
						}
						cdx++
					}
					select {
					case <-ctx.Done():
						return nil
					case ch <- next:
					}
				}
			}
		})
		return ch, eg.Wait
	}
}
