package v1

import (
	"context"
	"time"
)

// Histogrammer is able to produce histograms
type Histogrammer interface {
	Histogram(
		ctx context.Context,
		service string,
		rate float64,
		start, end time.Time,
		predicates []Predicate,
	)
}
