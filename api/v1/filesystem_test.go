package v1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilesystem(t *testing.T) {
	fs := &filesystem{
		Root: "./testdata",
	}
	ctx := context.Background()
	err := fs.Histogram(ctx, "output", 0.1, time.Now(), time.Now(), "bytes", []Predicate{})
	assert.Nil(t, err)
	assert.True(t, false)
}
