package v1

import (
	"math/rand"
	"time"

	"github.com/VividCortex/gohistogram"
	"github.com/scritchley/orc"
)

type ORC struct {
	r *orc.Reader
}

type Predicate struct {
	Field string
	Match func(interface{}) bool
}

func (o *ORC) Histogram(sample float64, start, end time.Time, field string, predicates []Predicate) (gohistogram.Histogram, error) {
	fields := []string{field}
	for _, p := range predicates {
		fields = append(fields, p.Field)
	}
	c := o.r.Select(fields...)
	h := gohistogram.NewHistogram(40)
	for c.Stripes() {
	Next:
		for c.Next() {
			if rand.Float64() > sample {
				continue
			}
			r := c.Row()
			for idx, p := range predicates {
				v, _ := r[idx+1].(string)
				if !p.Match(v) {
					continue Next
				}
			}
			f, _ := r[0].(int64)
			h.Add(float64(f))
		}
	}
	return h, nil
}
