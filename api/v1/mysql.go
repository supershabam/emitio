package v1

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type MySQL struct {
	db *sql.DB
}

type Histogram struct {
	Breakdown []string
	Buckets   []*Bucket
}

func (h *Histogram) print() {
	fmt.Printf("breakdown: %s\n", strings.Join(h.Breakdown, ","))
	for _, b := range h.Buckets {
		fmt.Printf("le%f		%d\n", b.LE, b.Count)
	}
}

type Bucket struct {
	LE    float64
	Count int64
}

func (m *MySQL) Heatmap(ctx context.Context,
	service string,
	field string,
	sampleRate float64,
	window Window,
	filters []Filter,
	breakdowns []Field,
	buckets []float64,
) error {
	tmpl := `SELECT count(*), le
	{{range $idx, $b := .Breakdowns}}
	  , b_{{$idx}}
	{{end}}
	FROM (
	SELECT
		CASE
			{{range $b := .Buckets}}
			WHEN {{$b.Field}} <= {{$b.LE}} THEN '{{$b.LE}}'
			{{end}}
			ELSE '+inf'
		END AS le
		{{range $idx, $b := .Breakdowns}}
		, {{$b}} AS b_{{$idx}}
		{{end}}
	FROM hot
	WHERE 1=1
	{{range $p := .Predicates}}
	AND {{$p.Field}} {{$p.Op}} {{$p.Value}}
	{{end}}
	) as buckets
	GROUP BY {{.GroupBy}}`
	t, err := template.New("query").Parse(tmpl)
	if err != nil {
		return err
	}
	type bucket struct {
		Field string
		LE    float64
	}
	type predicate struct {
		Field string
		Op    string
		Value string
	}
	type vars struct {
		Buckets    []bucket
		Predicates []predicate
		Breakdowns []string
		GroupBy    string
	}
	v := vars{
		Buckets: []bucket{},
		Predicates: []predicate{
			{
				Field: "service",
				Op:    "=",
				Value: `"` + service + `"`,
			},
			{
				Field: `data->"` + field + `"`,
				Op:    "IS NOT",
				Value: "NULL",
			},
			{
				Field: "at",
				Op:    ">=",
				Value: "'" + window.Start.UTC().Format(time.RFC3339) + "'",
			},
			{
				Field: "at",
				Op:    "<",
				Value: "'" + window.End.UTC().Format(time.RFC3339) + "'",
			},
		},
		Breakdowns: []string{},
		GroupBy:    "2",
	}
	for _, filter := range filters {
		v.Predicates = append(v.Predicates, predicate{
			Field: `data->"` + filter.JSONPath + `"`,
			Op:    "=",
			Value: filter.Equals,
		})
	}
	for _, b := range buckets {
		v.Buckets = append(v.Buckets, bucket{
			Field: `data->"` + field + `"`,
			LE:    b,
		})
	}
	for _, b := range breakdowns {
		v.Breakdowns = append(v.Breakdowns, `data->"`+b.JSONPath+`"`)
		v.GroupBy = v.GroupBy + "," + fmt.Sprintf("%d", len(v.Breakdowns)+2)
	}
	buf := bytes.NewBuffer([]byte{})
	err = t.Execute(buf, v)
	if err != nil {
		return err
	}
	r, err := m.db.QueryContext(ctx, string(buf.Bytes()))
	if err != nil {
		return err
	}
	histograms := map[string]Histogram{}
	for r.Next() {
		var (
			leStr string
			count int64
		)
		valuePtrs := []interface{}{&count, &leStr}
		for _ = range breakdowns {
			var i string
			valuePtrs = append(valuePtrs, &i)
		}
		err := r.Scan(valuePtrs...)
		if err != nil {
			return err
		}
		bd := []string{}
		for i := 2; i < len(valuePtrs); i++ {
			if s, ok := valuePtrs[i].(*string); ok {
				bd = append(bd, *s)
			}
		}
		key := strings.Join(bd, ",")
		h := histograms[key]
		if len(h.Buckets) == 0 {
			for _, b := range buckets {
				h.Buckets = append(h.Buckets, &Bucket{
					LE:    b,
					Count: 0,
				})
			}
		}
		if h.Breakdown == nil {
			h.Breakdown = bd
		}
		le, err := strconv.ParseFloat(leStr, 64)
		if err != nil {
			return err
		}
		for _, b := range h.Buckets {
			if le <= b.LE {
				b.Count = b.Count + count
			}
		}
		histograms[key] = h
	}
	for _, h := range histograms {
		h.print()
	}
	return nil
}
