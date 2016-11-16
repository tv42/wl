package wl

import (
	"encoding/binary"
	"math"
	"sync"
	"unsafe"
)

type BytePool struct {
	sync.Pool
}

var (
	order    binary.ByteOrder
	bytePool = &BytePool{
		sync.Pool{
			New: func() interface{} {
				return make([]byte, 64) // increase when panic: runtime error: slice bounds out of range
			},
		},
	}
)

func (bp *BytePool) Take(n int) []byte {
	buf := bp.Get().([]byte)
	return buf[:n]
}

func init() {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
}

func fixedToFloat64(fixed int32) float64 {
	dat := ((int64(1023 + 44)) << 52) + (1 << 51) + int64(fixed)
	return math.Float64frombits(uint64(dat)) - float64(3<<43)
}

func float64ToFixed(v float64) int32 {
	dat := v + float64(int64(3)<<(51-8))
	return int32(math.Float64bits(dat))
}
