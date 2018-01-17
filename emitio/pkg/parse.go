package pkg

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

// ParseStorage initializes a badger database from a uri.
func ParseStorage(rawuri string) (*badger.DB, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}
	if u.Scheme != "file" {
		return nil, fmt.Errorf("unknown storage scheme: %s", u.Scheme)
	}
	opts := badger.DefaultOptions
	opts.Dir = u.Path
	opts.ValueDir = u.Path
	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "badger open")
	}
	return db, nil
}

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
