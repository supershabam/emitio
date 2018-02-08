package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"go.uber.org/zap"
)

type Heatmap struct {
	Window     time.Duration
	Start      time.Time
	Histograms []Histogram
}

func (h Heatmap) Min() float64 {
	min := math.MaxFloat64
	for _, h := range h.Histograms {
		for k := range h {
			if math.IsInf(k, 0) {
				continue
			}
			if k < min {
				min = k
			}
		}
	}
	return min
}

func (h Heatmap) Max() float64 {
	max := -math.MaxFloat64
	for _, h := range h.Histograms {
		for k := range h {
			if math.IsInf(k, 0) {
				continue
			}
			if k > max {
				max = k
			}
		}
	}
	return max
}

func (h Heatmap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"min":        h.Min(),
		"max":        h.Max(),
		"window":     h.Window.Seconds(),
		"start":      h.Start.UnixNano() / 1e6,
		"histograms": h.Histograms,
	})
}

type Histogram map[float64]uint64

func (h Histogram) MarshalJSON() ([]byte, error) {
	m := []interface{}{}
	type Entry struct {
		LE    float64
		Count uint64
	}
	entries := []Entry{}
	for le, cnt := range h {
		entries = append(entries, Entry{
			LE:    le,
			Count: cnt,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LE < entries[j].LE
	})
	for _, e := range entries {
		if math.IsInf(e.LE, 1) {
			m = append(m, map[string]interface{}{
				"le": "+Inf",
				"c":  e.Count,
			})
			continue
		}
		m = append(m, map[string]interface{}{
			"le": fmt.Sprintf("%e", e.LE),
			"c":  e.Count,
		})
	}
	return json.Marshal(m)
}

func generateHistogram(rng *rand.Rand, buckets []float64, double bool) Histogram {
	count := rng.Int31n(6000)
	h := Histogram{}
	for i := 0; i < int(count); i++ {
		desiredStdDev := 40.0
		desiredMean := 220.0
		if rng.Intn(900) == 0 {
			desiredStdDev = 900
		}
		if double {
			if rng.Intn(4) == 0 {
				desiredMean = 440
			}
		}
		sample := rng.NormFloat64()*desiredStdDev + desiredMean
		for _, b := range buckets {
			if sample <= b {
				h[b]++
			}
		}
	}
	return h
}

func main() {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Content-Type", "application/json")
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		buckets := []float64{
			50,
			100,
			150,
			200,
			250,
			300,
			350,
			400,
			450,
			500,
			550,
			600,
			650,
			700,
		}
		h := Heatmap{
			Window:     time.Minute * 5,
			Start:      time.Now().Add(time.Minute * 5 * 5),
			Histograms: []Histogram{},
		}
		count := 5 + rng.Intn(50)
		double := false
		for i := 0; i < count; i++ {
			if rng.Intn(5) == 0 {
				double = !double
			}
			h.Histograms = append(h.Histograms, generateHistogram(rng, buckets, double))
		}
		err := json.NewEncoder(w).Encode(h)
		if err != nil {
			zap.L().Error("error writing response", zap.Error(err))
		}
	})
	addr := ":8080"
	zap.L().Info("listening", zap.String("addr", addr))
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
