package pkg

import "context"

// Transformer is able to transform input to different output and may retain state between
// calls by storing state in the accumulator variable.
type Transformer interface {
	// Transform operates like an accumulating flatmap
	Transform(ctx context.Context, accumulator string, input []string) (acc string, output []string, err error)
}
