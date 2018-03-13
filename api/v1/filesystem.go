package v1

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scritchley/orc"
	"github.com/supershabam/gohistogram"
)

type Filesystem struct {
	Root string
}

func (fs *Filesystem) Histogram(
	ctx context.Context,
	service string,
	rate float64,
	start, end time.Time,
	field string,
	predicates []Predicate,
) (gohistogram.Histogram, error) {
	files, err := fs.files(service)
	if err != nil {
		return nil, err
	}
	h := gohistogram.NewHistogram(40)
	fields := fs.fields(field, predicates)
	for _, file := range files {
		f, err := orc.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		c := f.Select(fields...)
		for c.Stripes() {
		Next:
			for c.Next() {
				if rand.Float64() > rate {
					continue
				}
				r := c.Row()
				for idx, p := range predicates {
					if !p.Match(r[idx+1]) {
						continue Next
					}
				}
				f, _ := r[0].(int64)
				h.Add(float64(f))
			}
		}
	}
	return h, nil
}

// files TODO filter on more things in the file name/path so that unnecessary files are never opened
func (fs *Filesystem) files(service string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(fs.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".orc") {
			return nil
		}
		if !strings.Contains(path, service) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (fs *Filesystem) fields(field string, predicates []Predicate) []string {
	fl := []string{field}
	for _, p := range predicates {
		fl = append(fl, p.Field)
	}
	return fl
}
