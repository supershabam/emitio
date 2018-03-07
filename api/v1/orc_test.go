package v1

import (
	"fmt"
	"testing"

	"github.com/VividCortex/gohistogram"

	"github.com/scritchley/orc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestORC(t *testing.T) {
	r, err := orc.Open("./testdata/output.orc")
	require.Nil(t, err)
	defer r.Close()
	// Create a new Cursor reading the provided columns.
	c := r.Select("response", "bytes")
	// Iterate over each stripe in the file.
	histograms := map[string]*gohistogram.NumericHistogram{}
	for c.Stripes() {
		// Iterate over each row in the stripe.
		for c.Next() {
			r := c.Row()
			response, _ := r[0].(int64)
			bytes, _ := r[1].(int64)
			key := fmt.Sprintf("%d", response)
			h := histograms[key]
			if h == nil {
				h = gohistogram.NewHistogram(80)
				histograms[key] = h
			}
			h.Add(float64(bytes))
		}
	}
	for k, h := range histograms {
		fmt.Printf("breakdown: %s\n%s\n", k, h.String())
	}
	assert.True(t, false)
}
