package storages

import (
	"context"
	"sync"
	"time"

	"github.com/supershabam/emitio/emitio/pkg"
)

type blockset struct {
	maxSize int
	maxAge  time.Duration
	create  func(context.Context, int64) (*block, error)
	blocks  []*block
	l       sync.RWMutex
	lastSeq int64
	cond    sync.Cond
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

func (bs *blockset) wait(ctx context.Context, seq int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	active := true
	go func() {
		<-ctx.Done()
		active = false
		bs.cond.Broadcast()
	}()
	bs.cond.L.Lock()
	for bs.lastSeq == seq && active {
		bs.cond.Wait()
	}
	bs.cond.L.Unlock()
	return nil
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
