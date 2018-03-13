package v1

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/scritchley/orc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeOrc() (*ORC, error) {
	r, err := orc.Open("./testdata/output.orc")
	if err != nil {
		return nil, err
	}
	return &ORC{r}, nil
}

func TestORC(t *testing.T) {
	o, err := makeOrc()
	require.Nil(t, err)
	h, err := o.Histogram(.4, time.Now().Add(-time.Hour*24*365*5), time.Now(), "bytes", []Predicate{
		{
			Field: "request",
			Match: func(s string) bool {
				return strings.Contains(s, "product_1")
			},
		},
	})
	require.Nil(t, err)
	fmt.Printf("%s\n", h.String())
	assert.True(t, false)
}

func BenchmarkORC(b *testing.B) {
	o, err := makeOrc()
	require.Nil(b, err)
	for n := 0; n < b.N; n++ {
		_, err := o.Histogram(0.5, time.Now().Add(-time.Hour*24*7), time.Now(), "bytes", []Predicate{
			{
				Field: "remote_ip",
				Match: func(s string) bool {
					return s == "37.58.92.201"
				},
			},
		})
		require.Nil(b, err)
	}
}
