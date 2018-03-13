package v1

import (
	"context"
	"net/http"
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

}

func WithHistogrammer(h Histogrammer) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.h = h
		return nil
	}
}
