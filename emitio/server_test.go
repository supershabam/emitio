package emitio

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransform(t *testing.T) {
	tr := &mockt{}
	ctx := context.TODO()
	acc := "hi"
	in := []string{}
	acc, out, err := tr.Transform(ctx, acc, in)
	require.Nil(t, err)
	spew.Dump(out)
	assert.Equal(t, "accumulator!", acc)
}
