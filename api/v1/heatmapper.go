package v1

import (
	"context"
	"time"
)

type Window struct {
	Start time.Time
	End   time.Time
}

type Filter struct {
	JSONPath string
	Equals   string
}

type Field struct {
	JSONPath string
}

type Heatmapper interface {
	Heatmap(ctx context.Context,
		service string,
		sampleRate float64,
		window Window,
		filters []Filter,
		breakdowns []Field,
		buckets []float64,
	)
}
