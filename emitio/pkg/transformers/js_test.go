package transformers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJS(t *testing.T) {
	js, err := NewJS(`function transform(acc, lines) {
	return ["hi", ["there"]]
}`)
	require.Nil(t, err)
	acc, lines, err := js.Transform("hi", []string{"one", "two"})
	require.Nil(t, err)
	assert.Equal(t, "hi", acc)
	assert.Equal(t, []string{"one", "two"}, lines)
}
