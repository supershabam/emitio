package pkg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supershabam/emitio/emitio/pkg/transformers"
)

func TestTransform(t *testing.T) {
	tr, err := transformers.NewJS(`
	function transform(acc, lines) {
		var a = JSON.parse(acc)
		var output = []
		for (var i = 0; i < lines.length; i++) {
			a.count++
			output.push(lines[i])
		}
		return [JSON.stringify(a), output]
	}
`)
	require.Nil(t, err)
	ctx := context.TODO()
	acc := `{"count":0}`
	in := []string{
		"{\"a\":\"2018-01-15T12:07:24.186726127-08:00\",\"r\":\"sldfkjsdjklfhi\\n\",\"s\":1}",
		"{\"a\":\"2018-01-15T12:12:32.977232909-08:00\",\"r\":\"sldfkjsdjklfhi\\n\",\"s\":2}",
	}
	acc, out, err := tr.Transform(ctx, acc, in)
	require.Nil(t, err)
	assert.Equal(t, `{"count":2}`, acc)
	assert.Equal(t, 2, len(out))
	acc, out, err = tr.Transform(ctx, acc, out)
	assert.Nil(t, err)
}
