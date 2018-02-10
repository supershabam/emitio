package pkg

import (
	"strings"

	"github.com/pkg/errors"
)

func ParseOriginTags(tags []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, tag := range tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("origin tag must be specified by ${tagName}=${tagValue}")
		}
		m[parts[0]] = parts[1]
	}
	return m, nil
}
