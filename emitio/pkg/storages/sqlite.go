package storages

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/supershabam/emitio/emitio/pkg"
)

type SQLite struct {
	blocksets map[string]*blockset
	l         sync.Mutex
}

type SQLiteOption func(ctx context.Context, s *SQLite) error

func NewSQLite(ctx context.Context, opts ...SQLiteOption) (*SQLite, error) {
	s := &SQLite{
		blocksets: map[string]*blockset{},
	}
	for _, opt := range opts {
		err := opt(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

type blockset struct {
	maxSize int
	maxAge  time.Duration
	create  func(context.Context, int64) (*block, error)
	blocks  []*block
	l       sync.RWMutex
}

func (bs *blockset) read(ctx context.Context, start, end int64, batch int) ([]pkg.SeqRecord, error) {
	bs.l.RLock()
	defer bs.l.RUnlock()
	for _, blk := range bs.blocks {
		if blk.lastSeq < start {
			continue
		}
		records, err := blk.read(ctx, start, end, batch)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			continue
		}
		return records, nil
	}
	return []pkg.SeqRecord{}, nil
}

func (bs *blockset) write(ctx context.Context, at time.Time, records []pkg.Record) error {
	bs.l.Lock()
	defer bs.l.Unlock()
	blk := bs.blocks[len(bs.blocks)-1]
	var rest []pkg.Record
	if !blk.lastAt.IsZero() && at.Sub(blk.lastAt) > bs.maxAge {
		rest = records
		records = []pkg.Record{}
	} else if blk.count+int64(len(records)) > int64(bs.maxSize) {
		// for now, just make a whole new block if we're going to be over, it's easier
		rest = records
		records = []pkg.Record{}
	}
	err := blk.write(ctx, records)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return nil
	}
	// assume we won't be writing more records in a single write than a max size
	blk, err = bs.create(ctx, blk.lastSeq)
	if err != nil {
		return err
	}
	bs.blocks = append(bs.blocks, blk)
	return blk.write(ctx, rest)
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
	// TODO make configurable
	bs = &blockset{
		maxAge:  time.Hour * 24,
		maxSize: 1e6,
		blocks:  []*block{},
		create: func(ctx context.Context, seq int64) (*block, error) {
			name, err := ioutil.TempDir("", "emitio")
			if err != nil {
				return nil, err
			}
			return newBlock(ctx, fmt.Sprintf("%s/%d.sq3", name, seq), seq)
		},
	}
	blk, err := bs.create(ctx, 0)
	if err != nil {
		return nil, err
	}
	bs.blocks = append(bs.blocks, blk)
	s.blocksets[uri] = bs
	return bs, nil
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

func (s *SQLite) Write(ctx context.Context, uri string, records []pkg.Record) error {
	bs, err := s.blockset(ctx, uri)
	if err != nil {
		return err
	}
	return bs.write(ctx, time.Now(), records)
}

type block struct {
	count   int64
	db      *sql.DB
	stmt    *sql.Stmt
	lastAt  time.Time
	lastSeq int64
	l       sync.Mutex
}

func newBlock(ctx context.Context, dsn string, seq int64) (*block, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE t(seq INTEGER PRIMARY KEY ASC, at INTEGER, blob BLOB);`)
	if err != nil {
		return nil, err
	}
	// set sentinel row to set base seq for the table
	_, err = db.ExecContext(ctx, `INSERT INTO t (seq, at) VALUES (?, ?)`, seq-1, 0)
	if err != nil {
		return nil, err
	}
	return &block{
		count:   0,
		db:      db,
		lastAt:  time.Time{},
		lastSeq: seq,
	}, nil
}

func openBlock(ctx context.Context, dsn string) (*block, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `SELECT seq, at FROM t ORDER BY seq DESC LIMIT 1`)
	var seq, at int64
	err = row.Scan(&seq, &at)
	if err != nil {
		return nil, err
	}
	row = db.QueryRowContext(ctx, `SELECT count(*) FROM t`)
	var count int64
	err = row.Scan(&count)
	if err != nil {
		return nil, err
	}
	return &block{
		count:   count - 1, // remove sentinel value from count
		db:      db,
		lastAt:  time.Unix(0, at),
		lastSeq: seq,
	}, nil
}

func (b *block) write(ctx context.Context, records []pkg.Record) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO t (at, blob) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	for _, record := range records {
		_, err := stmt.ExecContext(ctx, record.At.UnixNano(), record.Blob)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	atomic.AddInt64(&b.count, int64(len(records)))
	return nil
}

func (b *block) read(ctx context.Context, start, end int64, batchSize int) ([]pkg.SeqRecord, error) {
	var stmt *sql.Stmt
	b.l.Lock()
	if b.stmt == nil {
		s, err := b.db.PrepareContext(ctx, `SELECT at, blob, seq FROM t WHERE seq >= ? AND seq < ? AND at != 0 LIMIT ?`)
		if err != nil {
			b.l.Unlock()
			return nil, err
		}
		b.stmt = s
	}
	stmt = b.stmt
	b.l.Unlock()
	rows, err := stmt.QueryContext(ctx, start, end, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pkg.SeqRecord, 0, batchSize)
	for rows.Next() {
		var (
			at   int64
			blob []byte
			seq  int64
		)
		err := rows.Scan(&at, &blob, &seq)
		if err != nil {
			return nil, err
		}
		result = append(result, pkg.SeqRecord{
			At:   time.Unix(0, at),
			Blob: blob,
			Seq:  seq,
		})
	}
	return result, nil
}
