package emitio

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supershabam/emitio/emitio/pkg/transformers"
)

func TestTransform(t *testing.T) {
	tr, err := transformers.NewJS(`
	function transform(acc, lines) {
		return ["accumulator!", ["hi"]]
	}
`)
	require.Nil(t, err)
	ctx := context.TODO()
	acc := "hi"
	in := []string{}
	acc, out, err := tr.Transform(ctx, acc, in)
	require.Nil(t, err)
	spew.Dump(out)
	assert.Equal(t, "accumulator!", acc)
	assert.Equal(t, []string{"hi"}, out)
}
