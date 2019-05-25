package wl

import (
	"encoding/binary"
	"unsafe"
)

var (
	order binary.ByteOrder
)

func init() {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
}

// from https://golang.org/src/math/unsafe.go
func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }
func Float64bits(f float64) uint64     { return *(*uint64)(unsafe.Pointer(&f)) }

func fixedToFloat64(fixed int32) float64 {
	dat := ((int64(1023 + 44)) << 52) + (1 << 51) + int64(fixed)
	return Float64frombits(uint64(dat)) - float64(3<<43)
}

func float64ToFixed(v float64) int32 {
	dat := v + float64(int64(3)<<(51-8))
	return int32(Float64bits(dat))
}
