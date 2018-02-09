package storages

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/supershabam/emitio/emitio/pkg"
)

type SQLite struct {
	db *sql.DB
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

// func NewSQLite(ctx context.Context, rawuri string) (*SQLite, error) {
// 	if rawuri != "memory://" {
// 		return nil, errors.New("in memory is the only supported  right now")
// 	}
// 	db, err := sql.Open("sqlite3", ":memory:")
// 	if err != nil {
// 		return nil, err
// 	}
// 	_, err = db.ExecContext(ctx, `CREATE TABLE t(seq INTEGER PRIMARY KEY ASC, at INTEGER, blob BLOB);`)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &SQLite{db}, nil
// }

// func (s *SQLite) Write(ctx context.Context, uri string, records []pkg.Record) error {
// 	zap.L().Debug("sqlite  write", zap.String("uri", uri), zap.Int("num_records", len(records)))
// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	stmt, err := tx.PrepareContext(ctx, `INSERT INTO t (seq, at, blob) VALUES (?, ?, ?)`)
// 	if err != nil {
// 		return err
// 	}
// 	for _, record := range records {
// 		_, err := stmt.ExecContext(ctx, record.Seq, record.At.UnixNano(), record.Blob)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return tx.Commit()
// }

// func (s *SQLite) Read(ctx context.Context, uri string, start, end int64, batchSize int) (<-chan []pkg.Record, pkg.Wait) {
// 	zap.L().Debug("sqlite  read", zap.String("uri", uri), zap.Int64("start", start), zap.Int64("end", end))
// 	ch := make(chan []pkg.Record)
// 	eg, ctx := errgroup.WithContext(ctx)
// 	eg.Go(func() error {
// 		defer close(ch)
// 		stmt, err := s.db.PrepareContext(ctx, `SELECT at, blob, seq FROM t WHERE seq >= ? AND seq < ? LIMIT ?`)
// 		if err != nil {
// 			return err
// 		}
// 		defer stmt.Close()
// 		for start < end {
// 			records, err := s.read(ctx, stmt, start, end, batchSize)
// 			if err != nil {
// 				return err
// 			}
// 			if len(records) == 0 {
// 				return nil
// 			}
// 			start = records[len(records)-1].Seq + 1
// 			select {
// 			case <-ctx.Done():
// 				return nil
// 			case ch <- records:
// 				zap.L().Debug("sqlite  read send records", zap.String("uri", uri), zap.Int64("start", start), zap.Int64("end", end), zap.Int("num_records", len(records)))
// 			}
// 		}
// 		return nil
// 	})
// 	return ch, eg.Wait
// }

// func (s *SQLite) read(ctx context.Context, stmt *sql.Stmt, start, end int64, batchSize int) ([]pkg.Record, error) {
// 	rows, err := stmt.QueryContext(ctx, start, end, batchSize)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	result := make([]pkg.Record, 0, batchSize)
// 	for rows.Next() {
// 		var (
// 			at   int64
// 			blob []byte
// 			seq  int64
// 		)
// 		err := rows.Scan(&at, &blob, &seq)
// 		if err != nil {
// 			return nil, err
// 		}
// 		result = append(result, pkg.Record{
// 			At:   time.Unix(0, at),
// 			Blob: blob,
// 			Seq:  seq,
// 		})
// 	}
// 	return result, nil
// }
