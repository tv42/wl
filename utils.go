package wl

import (
	"encoding/binary"
	"io/ioutil"
	"math"
	"os"
	"sync"
	"unsafe"
)

var (
	order    binary.ByteOrder
	bytePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64)
		},
	}
)

func init() {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
}

func CreateAnonymousFile(size int64) (*os.File, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		panic("XDG_RUNTIME_DIR not defined.")
	}
	file, err := ioutil.TempFile(dir, "wayland-shared")
	if err != nil {
		return nil, err
	}
	err = file.Truncate(size)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func fixedToFloat64(fixed int32) float64 {
	dat := ((int64(1023 + 44)) << 52) + (1 << 51) + int64(fixed)
	return math.Float64frombits(uint64(dat)) - float64(3<<43)
}

func float64ToFixed(v float64) int32 {
	dat := v + float64(int64(3)<<(51-8))
	return int32(math.Float64bits(dat))
}
