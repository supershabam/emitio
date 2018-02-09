package transformers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJS(t *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)
	tt := []struct {
		name     string
		program  string
		inacc    []string
		inlines  [][]string
		outacc   []string
		outlines [][]string
		outerr   []error
		err      error
	}{
		{
			name: "passthrough",
			program: `
function transform(acc, lines) {
	return [acc, lines]
}`,
			inacc: []string{
				"acc",
				"",
				"世界",
			},
			inlines: [][]string{
				{"one", "two"},
				{},
				{"世界世界"},
			},
			outacc: []string{
				"acc",
				"",
				"世界",
			},
			outlines: [][]string{
				{"one", "two"},
				{},
				{"世界世界"},
			},
			outerr: []error{nil, nil, nil},
			err:    nil,
		},
		{
			name: "double",
			program: `
function transform(acc, lines) {
	return [acc + acc, [].concat(lines).concat(lines)]
}`,
			inacc: []string{
				"acc",
				"",
				"世界",
			},
			inlines: [][]string{
				{"one", "two"},
				{},
				{"世界世界"},
			},
			outacc: []string{
				"accacc",
				"",
				"世界世界",
			},
			outlines: [][]string{
				{"one", "two", "one", "two"},
				{},
				{"世界世界", "世界世界"},
			},
			outerr: []error{nil, nil, nil},
			err:    nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			js, err := NewJS(tc.program)
			if tc.err != nil {
				require.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.err.Error())
				return
			}
			for idx := range tc.inacc {
				acc, lines, err := js.Transform(context.Background(), tc.inacc[idx], tc.inlines[idx])
				if err != nil {
					require.NotNil(t, tc.outerr[idx], "got error %s when not expecting idx=%d", err.Error(), idx)
					assert.Contains(t, err.Error(), tc.outerr[idx].Error())
					continue
				}
				assert.Equal(t, tc.outacc[idx], acc)
				assert.Equal(t, tc.outlines[idx], lines)
			}
		})
	}
}
