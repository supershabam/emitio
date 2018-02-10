package storages

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/supershabam/emitio/emitio/pkg"
)

type SQLite struct {
	blocksets map[string]*blockset
	l         sync.Mutex
	resolver  resolver
}

type SQLiteOption func(ctx context.Context, s *SQLite) error

type resolver interface {
	Blocksets(context.Context) (map[string]*blockset, error)
	Create(context.Context, string) (*blockset, error)
}

const fileInfoVersion1 = "v1"

type fileInfo struct {
	ID      string `json:"id"`
	URI     string `json:"uri"`
	Version string `json:"v"`
}

type fileResolver struct {
	base string
}

func (fr *fileResolver) blocksetFn(uri string) func(ctx context.Context) (*blockset, error) {
	return func(ctx context.Context) (*blockset, error) {
		bs := &blockset{
			maxSize: 1024,      // TODO make configurable
			maxAge:  time.Hour, // TODO make configurable
			create: func(ctx context.Context, seq int64) (*block, error) {
				id := uuid.NewV4().String()
				b, err := json.Marshal(fileInfo{
					ID:      id,
					URI:     uri,
					Version: fileInfoVersion1,
				})
				if err != nil {
					return nil, errors.Wrap(err, "json marshal filename contents")
				}
				file := base64.RawURLEncoding.EncodeToString(b) + ".db"
				zap.L().Debug("creating new block", zap.String("file", file))
				return newBlock(ctx, path.Join(fr.base, file), seq)
			},
			blocks: []*block{},
		}
		return bs, nil
	}
}

func (fr *fileResolver) Blocksets(ctx context.Context) (map[string]*blockset, error) {
	err := os.MkdirAll(fr.base, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "ensuring base path")
	}
	blocksets := map[string]*blockset{}
	err = filepath.Walk(fr.base, func(p string, info os.FileInfo, err error) error {
		zap.L().Debug("walking", zap.String("file", p))
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".db") {
			return nil
		}
		filename := strings.TrimSuffix(path.Base(p), ".db")
		zap.L().Debug("parsing filename for information", zap.String("filename", filename))
		b, err := base64.RawURLEncoding.DecodeString(filename)
		if err != nil {
			// skip
			zap.L().Debug("skipping file because of base64 decode error", zap.Error(err))
			return nil
		}
		var fi fileInfo
		err = json.Unmarshal(b, &fi)
		if err != nil {
			// skip
			zap.L().Debug("skipping file because of json unmarshal error", zap.Error(err))
			return nil
		}
		if fi.Version != fileInfoVersion1 {
			// skip
			zap.L().Debug("skipping file because version mismatch", zap.String("version", fi.Version))
			return nil
		}
		blk, err := openBlock(ctx, p)
		if err != nil {
			return err
		}
		if _, ok := blocksets[fi.URI]; !ok {
			bs, err := fr.blocksetFn(fi.URI)(ctx)
			if err != nil {
				return err
			}
			blocksets[fi.URI] = bs
		}
		bs := blocksets[fi.URI]
		bs.blocks = append(bs.blocks, blk)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, blockset := range blocksets {
		sort.Slice(blockset.blocks, func(i, j int) bool {
			// handle the case where we've created a new block, but not inserted anything into it
			if blockset.blocks[i].lastSeq == blockset.blocks[j].lastSeq {
				return blockset.blocks[i].count > blockset.blocks[j].count
			}
			return blockset.blocks[i].lastSeq < blockset.blocks[j].lastSeq
		})
	}
	return blocksets, nil
}

func (fr *fileResolver) Create(ctx context.Context, uri string) (*blockset, error) {
	bs, err := fr.blocksetFn(uri)(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create")
	}
	blk, err := bs.create(ctx, 0)
	if err != nil {
		return nil, errors.Wrap(err, "memory resolver populate first block")
	}
	bs.blocks = append(bs.blocks, blk)
	return bs, nil
}

type memoryResolver struct{}

func (mr *memoryResolver) Blocksets(context.Context) (map[string]*blockset, error) {
	return map[string]*blockset{}, nil
}

func (mr *memoryResolver) Create(ctx context.Context, uri string) (*blockset, error) {
	bs := &blockset{
		maxSize: 1024,
		maxAge:  time.Hour,
		create: func(ctx context.Context, seq int64) (*block, error) {
			return newBlock(ctx, ":memory:", seq)
		},
		blocks: []*block{},
	}
	blk, err := bs.create(ctx, 0)
	if err != nil {
		return nil, errors.Wrap(err, "memory resolver populate first block")
	}
	bs.blocks = append(bs.blocks, blk)
	return bs, nil
}

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
	zap.L().Debug("opening block", zap.String("dsn", dsn))
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
