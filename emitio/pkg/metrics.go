package pkg

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type metricKey int

const (
	ingressMessageCount metricKey = iota
)

func InitContextMetrics(ctx context.Context, registry prometheus.Registerer) (context.Context, error) {
	var c prometheus.Collector
	c = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "emitio",
			Subsystem: "ingress",
			Name:      "message_count",
			Help:      "count of messages received by an ingress",
		},
		[]string{"ingress"},
	)
	err := registry.Register(c)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, ingressMessageCount, c)
	return ctx, nil
}

func IncIngressMessage(ctx context.Context, uri string) {
	v, ok := ctx.Value(ingressMessageCount).(*prometheus.CounterVec)
	if !ok {
		zap.L().Debug("no collector found for inc ingress message")
		return
	}
	zap.L().Debug("incremented ingress message", zap.String("uri", uri))
	v.WithLabelValues(uri).Add(1)
}
