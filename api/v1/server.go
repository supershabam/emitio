package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ServerOption func(ctx context.Context, s *Server) error

type Server struct {
	h Histogrammer
}

func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{}
	for _, opt := range opts {
		err := opt(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, err := s.h.Histogram(r.Context(), "output", 0.1, time.Now(), time.Now(), "bytes", []Predicate{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Content-Type", "application/json")
	buckets := h.Cummulative()
	type bucket struct {
		LTE   float64 `json:"lte"`
		Count float64 `json:"count"`
	}
	type response struct {
		Buckets []bucket `json:"buckets"`
	}
	res := response{
		Buckets: make([]bucket, 0, len(buckets)),
	}
	for _, b := range buckets {
		res.Buckets = append(res.Buckets, bucket{
			LTE:   b.LTE,
			Count: b.Count,
		})
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(res)
	if err != nil {
		zap.L().Error("while encoding response", zap.Error(err))
	}
}

func WithHistogrammer(h Histogrammer) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.h = h
		return nil
	}
}
