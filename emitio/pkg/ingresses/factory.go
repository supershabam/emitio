package ingresses

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/emitio/pkg"
)

// MakeIngress parses a rawuri to initialize an ingress
func MakeIngress(rawuri string) (pkg.Ingresser, error) {
	uri, err := url.Parse(rawuri)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}
	// allow scheme hints, e.g., syslog+udp://
	// the last element in the "+"-separated string is the type of ingress
	// to create.
	parts := strings.Split(uri.Scheme, "+")
	switch parts[len(parts)-1] {
	case "udp":
		return &UDP{uri}, nil
	}
	return nil, fmt.Errorf("unhandled ingress scheme: %s", uri.Scheme)
}
