package v1

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"text/template"
	"time"
)

type MySQL struct {
	db *sql.DB
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
			WHEN {{$b.Field}} <= {{$b.LE}} THEN "{{$b.LE}}"
			{{end}}
			ELSE "+inf"
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
	fmt.Printf("%s\n", buf.Bytes())
	r, err := m.db.QueryContext(ctx, string(buf.Bytes()))
	if err != nil {
		return err
	}
	for r.Next() {
		var (
			le    string
			count int64
		)
		valuePtrs := []interface{}{&count, &le}
		for _ = range breakdowns {
			var i string
			valuePtrs = append(valuePtrs, &i)
		}
		err := r.Scan(valuePtrs...)
		if err != nil {
			return err
		}
		fmt.Println(le)
		fmt.Println(count)
		for i := 2; i < len(valuePtrs); i++ {
			if s, ok := valuePtrs[i].(*string); ok {
				fmt.Println(*s)
			}
		}
	}
	return nil
}
