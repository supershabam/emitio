package v1

import (
	"context"
	"time"

	"github.com/supershabam/gohistogram"
)

// Histogrammer is able to produce histograms
type Histogrammer interface {
	Histogram(
		ctx context.Context,
		service string,
		rate float64,
		start, end time.Time,
		field string,
		predicates []Predicate,
	) (gohistogram.Histogram, error)
}
