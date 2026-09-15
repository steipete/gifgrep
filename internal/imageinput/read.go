// Package imageinput bounds encoded images before buffering them in memory.
package imageinput

import (
	"io"
	"os"

	"github.com/steipete/gifgrep/gifdecode"
)

func Read(r io.Reader) ([]byte, error) {
	limit := gifdecode.DefaultOptions().MaxBytes
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, gifdecode.ErrTooLarge
	}
	return data, nil
}

func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return Read(f)
}
