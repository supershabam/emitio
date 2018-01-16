package pkg

import (
	"fmt"
	"net/url"

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
