package boson

import (
	"context"
	"fmt"
	"math"
	"sync"
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

type Wait func() error

func sequence(filters ...filter) filter {
	return func(ctx context.Context, d *Database, in <-chan []uint32) (<-chan []uint32, Wait) {
		var wait func() error
		for _, f := range filters {
			in, wait = f(ctx, d, in)
		}
		return in, wait
	}
}

type filter func(ctx context.Context, d *Database, in <-chan []uint32) (<-chan []uint32, Wait)

type Database struct {
	index          []time.Time
	float64Columns map[string]Float64Column
	stringColumns  map[string]StringColumn
	l              sync.Mutex
}

func (d *Database) Float64Columns(source, field string, ids []uint32) ([]uint32, []float64, error) {
	var columns []uint32
	var values []float64
	d.l.Lock()
	l, ok := d.float64Columns[field]
	d.l.Unlock()
	if !ok {
		fmt.Printf("tried to find field=%s but did not\n", field)
		return columns, values, nil
	}
	columns, values = make([]uint32, 0, len(ids)), make([]float64, 0, len(ids))
	idx := 0
	cdx := 0
	for idx < len(ids) && cdx < len(l.Index) {
		id, col := ids[idx], l.Index[cdx]
		if id == col {
			columns = append(columns, col)
			values = append(values, l.Values[cdx])
			cdx++
			idx++
			continue
		}
		if id < col {
			idx++
			continue
		}
		cdx++
	}
	return columns, values, nil
}

func (d *Database) StringColumns(source, field string, ids []uint32) ([]uint32, []string, error) {
	var columns []uint32
	var values []string
	d.l.Lock()
	l, ok := d.stringColumns[field]
	d.l.Unlock()
	if !ok {
		return columns, values, nil
	}
	columns, values = make([]uint32, 0, len(ids)), make([]string, 0, len(ids))
	idx := 0
	cdx := 0
	for idx < len(ids) && cdx < len(l.Index) {
		id, col := ids[idx], l.Index[cdx]
		if id == col {
			columns = append(columns, col)
			values = append(values, l.Values[cdx])
			cdx++
			idx++
			continue
		}
		if id < col {
			idx++
			continue
		}
		cdx++
	}
	return columns, values, nil
}

func between(ctx context.Context, d *Database, start, end time.Time) (<-chan []uint32, Wait) {
	ch := make(chan []uint32)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		batch := make([]uint32, 0, 512)
		flush := func() bool {
			if len(batch) == 0 {
				return true
			}
			select {
			case <-ctx.Done():
				return false
			case ch <- batch:
				batch = make([]uint32, 0, 512)
				return true
			}
		}
		for idx := 0; idx < len(d.index); idx++ {
			at := d.index[idx]
			if at.Before(start) {
				continue
			}
			if at.Equal(end) || at.After(end) {
				flush()
				return nil
			}
			batch = append(batch, uint32(idx))
			if len(batch) == cap(batch) {
				ok := flush()
				if !ok {
					return nil
				}
			}
		}
		flush()
		return nil
	})
	return ch, eg.Wait
}

func makeFloat64ColumnFilter(source, field string, match func(id, column uint32, value float64) bool) filter {
	return func(ctx context.Context, d *Database, in <-chan []uint32) (<-chan []uint32, Wait) {
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
					columns, values, err := d.Float64Columns(source, field, ids)
					if err != nil {
						return err
					}
					for idx, cdx := 0, 0; idx < len(ids) && cdx < len(columns); idx++ {
						id := ids[idx]
						column := columns[cdx]
						if id != column {
							if match(id, math.MaxUint32, 0) {
								next = append(next, id)
							}
							continue
						}
						if match(id, column, values[cdx]) {
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

func makeStringColumnFilter(source, field string, match func(id, column uint32, value string) bool) filter {
	return func(ctx context.Context, d *Database, in <-chan []uint32) (<-chan []uint32, Wait) {
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
					columns, values, err := d.StringColumns(source, field, ids)
					if err != nil {
						return err
					}
					for idx, cdx := 0, 0; idx < len(ids) && cdx < len(columns); idx++ {
						id := ids[idx]
						column := columns[cdx]
						if id != column {
							if match(id, math.MaxUint32, "") {
								next = append(next, id)
							}
							continue
						}
						if match(id, column, values[cdx]) {
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
