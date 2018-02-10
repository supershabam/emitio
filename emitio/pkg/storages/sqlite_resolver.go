package storages

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

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
