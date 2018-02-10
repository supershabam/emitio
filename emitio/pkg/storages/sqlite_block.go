package storages

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/supershabam/emitio/emitio/pkg"
	"go.uber.org/zap"
)

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
