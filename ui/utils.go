package ui

import (
	"errors"
	"io/ioutil"
	"os"
)

func TempFile(size int64) (*os.File, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return nil, errors.New("XDG_RUNTIME_DIR is not defined in env")
	}
	file, err := ioutil.TempFile(dir, "go-wayland-shared")
	if err != nil {
		return nil, err
	}
	err = file.Truncate(size)
	if err != nil {
		return nil, err
	}
	err = os.Remove(file.Name())
	if err != nil {
		return nil, err
	}
	return file, nil
}

//https://github.com/golang/exp/blob/master/shiny/driver/internal/swizzle/swizzle_common.go
func BGRA(p []byte) {
	if len(p)%4 != 0 {
		panic("input slice length is not a multiple of 4")
	}

	for i := 0; i < len(p); i += 4 {
		p[i+0], p[i+2] = p[i+2], p[i+0]
	}
}
