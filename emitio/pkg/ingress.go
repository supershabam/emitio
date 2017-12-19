package pkg

import (
	"context"
	"errors"
	"net/url"
)

type Wait func() error

type Message struct {
	Origin map[string]string
	What   map[string]interface{}
}

type Ingresser interface {
	Ingress(ctx context.Context) (<-chan Message, Wait)
}

func ParseIngress(uri string) (Ingresser, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "syslog+udp":
		i := &UDP{
			network: "udp",
			addr:    u.Host,
			uri:     uri,
		}
		return i, nil
	}
	return nil, errors.New("unknown ingress")
}
