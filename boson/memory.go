package boson

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type element struct {
	At     time.Time
	Fields []Field
}

func (e element) Get(name string) interface{} {
	for _, f := range e.Fields {
		if f.Name != name {
			continue
		}
		return f.Value
	}
	return nil
}

type Field struct {
	Name  string
	Value interface{}
}

type Memory struct {
	elements []element
}

func (m *Memory) Query(ctx context.Context, s Select, f From, w Where, g GroupBy, d Downsampling, b Between) (<-chan []Result, Wait) {
	out := make(chan []Result)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(out)
		results := []element{}
	Loop:
		for _, e := range m.elements {
			if e.At.Before(b.Start) {
				continue
			}
			if e.At.After(b.End) || b.End.Equal(e.At) {
				continue
			}
			for _, wc := range w {
				v, err := wc.SelectExp.Transform(e.Get(wc.SelectExp.Field))
				if err != nil {
					return err
				}
				ok, err := wc.Match(v)
				if err != nil {
					return err
				}
				if !ok {
					continue Loop
				}
			}
			next := element{
				At:     e.At,
				Fields: []Field{},
			}
			for _, sf := range s {
				v, err := sf.Transform(e.Get(sf.Field))
				if err != nil {
					return err
				}
				next.Fields = append(next.Fields, Field{
					Name:  sf.Field,
					Value: v,
				})
			}
			results = append(results, next)
		}

		return nil
	})
	return out, eg.Wait
}
