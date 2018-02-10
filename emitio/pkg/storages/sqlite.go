package storages

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/supershabam/emitio/emitio/pkg"
)

// TODO run garbage collection to purge old dbs

type SQLite struct {
	blocksets map[string]*blockset
	l         sync.Mutex
	resolver  resolver
}

type SQLiteOption func(ctx context.Context, s *SQLite) error

func WithResolverConfig(rawuri string) SQLiteOption {
	return func(ctx context.Context, s *SQLite) error {
		u, err := url.Parse(rawuri)
		if err != nil {
			return errors.Wrap(err, "url parse")
		}
		switch u.Scheme {
		case "memory":
			s.resolver = &memoryResolver{}
			return nil
		case "file":
			s.resolver = &fileResolver{
				base: u.Path,
			}
			return nil
		default:
			return fmt.Errorf("unhandled scheme=%s for sqlite resolver", u.Scheme)
		}
	}
}

func NewSQLite(ctx context.Context, opts ...SQLiteOption) (*SQLite, error) {
	s := &SQLite{}
	for _, opt := range opts {
		err := opt(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	blocksets, err := s.resolver.Blocksets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new sqlite creating blockets")
	}
	s.blocksets = blocksets
	return s, nil
}

// Close will call close on all currently open db handles. All reads/writes to SQLite
// must be finished before calling close and it is unsafe to use SQLite after calling
// Close. TODO this should be improved with a "isClosed" guard around function calls
// and Close should respect ongoing read/writes and gracefully shut them down.
func (s *SQLite) Close() error {
	for _, bs := range s.blocksets {
		for _, b := range bs.blocks {
			b.db.Close()
		}
	}
	return nil
}

func (s *SQLite) Read(ctx context.Context, uri string, start, end int64, batch int) (<-chan []pkg.SeqRecord, pkg.Wait) {
	ch := make(chan []pkg.SeqRecord)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		bs, err := s.blockset(ctx, uri)
		if err != nil {
			return err
		}
		for start < end {
			records, err := bs.read(ctx, start, end, batch)
			if err != nil {
				return err
			}
			if len(records) == 0 {
				return nil
			}
			start = records[len(records)-1].Seq + 1
			select {
			case <-ctx.Done():
				return nil
			case ch <- records:
			}
		}
		return nil
	})
	return ch, eg.Wait
}

func (s *SQLite) URIs(ctx context.Context) ([]string, error) {
	uris := make([]string, 0)
	s.l.Lock()
	for uri := range s.blocksets {
		uris = append(uris, uri)
	}
	s.l.Unlock()
	return uris, nil
}

// Watch allows the caller to block until either their context expires or a value > seq
// is available from storage.
func (s *SQLite) Watch(ctx context.Context, uri string, seq int64) error {
	bs, err := s.blockset(ctx, uri)
	if err != nil {
		return err
	}
	return bs.wait(ctx, seq)
}

func (s *SQLite) Write(ctx context.Context, uri string, records []pkg.Record) error {
	bs, err := s.blockset(ctx, uri)
	if err != nil {
		return err
	}
	return bs.write(ctx, time.Now(), records)
}

func (s *SQLite) blockset(ctx context.Context, uri string) (*blockset, error) {
	var bs *blockset
	s.l.Lock()
	bs = s.blocksets[uri]
	if bs != nil {
		s.l.Unlock()
		return bs, nil
	}
	defer s.l.Unlock()
	bs, err := s.resolver.Create(ctx, uri)
	if err != nil {
		return nil, err
	}
	blk, err := bs.create(ctx, 0)
	if err != nil {
		return nil, err
	}
	bs.blocks = append(bs.blocks, blk)
	s.blocksets[uri] = bs
	return bs, nil
}
