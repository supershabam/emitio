package pkg

import (
	"context"
	"time"
)

type Record struct {
	At   time.Time
	Blob []byte
}

type SeqRecord struct {
	At   time.Time
	Blob []byte
	Seq  int64
}

type Storage interface {
	Read(ctx context.Context, uri string, start, end int64, batchSize int) (<-chan []SeqRecord, Wait)
	Write(ctx context.Context, uri string, records []Record) error
}
