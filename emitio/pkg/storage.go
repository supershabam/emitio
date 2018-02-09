package pkg

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/errgroup"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type Record struct {
	At   time.Time
	Blob []byte
	Seq  int64
}

type Storage interface {
	Read(ctx context.Context, uri string, start, end, batchSize int) (<-chan []Record, Wait)
	Write(ctx context.Context, uri string, records []Record) error
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(ctx context.Context, rawuri string) (*SQLiteStore, error) {
	if rawuri != "memory:///" {
		return nil, errors.New("in memory is the only supported store right now")
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE t(seq INTEGER PRIMARY KEY ASC, at INTEGER, blob BLOB);`)
	if err != nil {
		return nil, err
	}
	return &SQLiteStore{db}, nil
}

func (s *SQLiteStore) Write(ctx context.Context, uri string, records []Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO t (seq, at, blob) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, record := range records {
		_, err := stmt.ExecContext(ctx, record.Seq, record.At.UnixNano(), record.Blob)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) Read(ctx context.Context, uri string, start, end int64, batchSize int) (<-chan []Record, Wait) {
	ch := make(chan []Record)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		stmt, err := s.db.PrepareContext(ctx, `SELECT at, blob, seq FROM t WHERE seq >= ? AND seq < ? LIMIT ?`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for start < end {
			records, err := s.read(ctx, stmt, start, end, batchSize)
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

func (s *SQLiteStore) read(ctx context.Context, stmt *sql.Stmt, start, end int64, batchSize int) ([]Record, error) {
	rows, err := stmt.QueryContext(ctx, start, end, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Record, 0, batchSize)
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
		result = append(result, Record{
			At:   time.Unix(0, at),
			Blob: blob,
			Seq:  seq,
		})
	}
	return result, nil
}
