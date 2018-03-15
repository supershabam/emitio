package m

import (
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	b, err := ioutil.ReadFile("./testdata/input.nljson")
	require.Nil(t, err)
	var m Memory
	err = m.UnmarshalJSON(b)
	require.Nil(t, err)
	spew.Dump(m)
	require.True(t, false)
}
