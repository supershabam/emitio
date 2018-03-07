package boson

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadscots() (*Database, error) {
	b, err := ioutil.ReadFile("./testdata/scotts.json")
	if err != nil {
		return nil, err
	}
	type scotts []struct {
		ID      float64 `json:"ID"`
		Date    string  `json:"Date"`
		Title   string  `json:"Title"`
		Sponsor string  `json:"Sponsor"`
	}
	var s scotts
	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	sort.Slice(s, func(i, j int) bool {
		it, err := time.Parse("2006-01-02T15:04:05", s[i].Date)
		if err != nil {
			panic(err)
		}
		jt, err := time.Parse("2006-01-02T15:04:05", s[j].Date)
		if err != nil {
			panic(err)
		}
		return it.Before(jt)
	})
	d := &Database{
		index:          make([]time.Time, 0, len(s)),
		float64Columns: map[string]Float64Column{},
		stringColumns:  map[string]StringColumn{},
	}
	ids := Float64Column{
		Index:  make([]uint32, 0, len(s)),
		Values: make([]float64, 0, len(s)),
	}
	titles := StringColumn{
		Index:  make([]uint32, 0, len(s)),
		Values: make([]string, 0, len(s)),
	}
	sponsors := StringColumn{
		Index:  make([]uint32, 0, len(s)),
		Values: make([]string, 0, len(s)),
	}
	for idx, e := range s {
		t, err := time.Parse("2006-01-02T15:04:05", e.Date)
		if err != nil {
			return nil, err
		}
		d.index = append(d.index, t)
		ids.Index = append(ids.Index, uint32(idx))
		titles.Index = append(titles.Index, uint32(idx))
		sponsors.Index = append(sponsors.Index, uint32(idx))
		ids.Values = append(ids.Values, e.ID)
		titles.Values = append(titles.Values, e.Title)
		sponsors.Values = append(sponsors.Values, e.Sponsor)
	}
	d.float64Columns["ID"] = ids
	d.stringColumns["Title"] = titles
	d.stringColumns["Sponsor"] = sponsors
	return d, nil
}

func BenchmarkColumn(b *testing.B) {
	ctx := context.Background()
	f1 := makeStringColumnFilter("src", "Title", func(id, column uint32, value string) bool {
		return column != math.MaxUint32 && strings.Contains(value, "a")
	})
	f2 := makeStringColumnFilter("src", "Sponsor", func(id, column uint32, value string) bool {
		return column != math.MaxUint32 && strings.Contains(value, "Bob")
	})
	f := sequence(f1, f2)
	d, err := loadscots()
	require.Nil(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		in, _ := between(ctx, d, time.Now().Add(-time.Hour*24*365*2), time.Now())
		in, _ = f(ctx, d, in)
		_, err := sum(ctx, in, d, "src", "ID")
		require.Nil(b, err)
	}
}

func TestColumn(t *testing.T) {
	ctx := context.Background()
	f1 := makeStringColumnFilter("src", "Title", func(id, column uint32, value string) bool {
		return column != math.MaxUint32 && strings.Contains(value, "a")
	})
	f2 := makeStringColumnFilter("src", "Sponsor", func(id, column uint32, value string) bool {
		return column != math.MaxUint32 && strings.Contains(value, "Bob")
	})
	f := sequence(f1, f2)
	d, err := loadscots()
	require.Nil(t, err)
	count := 0
	in, _ := between(ctx, d, time.Now().Add(-time.Hour*24*365*2), time.Now())
	nextCh, wait := f(ctx, d, in)
	for next := range nextCh {
		count += len(next)
	}
	err = wait()
	require.Nil(t, err)
	fmt.Printf("count=%d\n", count)
	assert.True(t, false)
}

func TestColumnSum(t *testing.T) {
	ctx := context.Background()
	f1 := makeStringColumnFilter("src", "Title", func(id, column uint32, value string) bool {
		return column != math.MaxUint32
	})
	f2 := makeStringColumnFilter("src", "Sponsor", func(id, column uint32, value string) bool {
		return column != math.MaxUint32
	})
	f := sequence(f1, f2)
	d, err := loadscots()
	require.Nil(t, err)
	t0 := time.Now()
	in, _ := between(ctx, d, time.Now().Add(-time.Hour*24*365*2), time.Now())
	in, _ = f(ctx, d, in)
	v, err := sum(ctx, in, d, "src", "ID")
	require.Nil(t, err)
	fmt.Printf("duration=%s\n", time.Since(t0))
	fmt.Printf("sum=%f\n", v)
	assert.True(t, false)
}
