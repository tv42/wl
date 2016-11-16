package wl

import (
	"bytes"
	"syscall"
)

type Event struct {
	pid    ProxyId
	opcode uint32
	data   []byte
	scms   []syscall.SocketControlMessage
	off    int
}

func (ev *Event) FD() uintptr {
	if ev.scms == nil {
		return 0
	}
	fds, err := syscall.ParseUnixRights(&ev.scms[0])
	if err != nil {
		panic("Unable to parse unix rights")
	}
	//TODO is this required
	ev.scms = append(ev.scms, ev.scms[1:]...)
	return uintptr(fds[0])
}

func (ev *Event) Uint32() uint32 {
	buf := ev.next(4)
	if len(buf) != 4 {
		panic("Unable to read unsigned int")
	}
	return order.Uint32(buf)
}

func (ev *Event) Proxy(c *Connection) Proxy {
	return c.lookupProxy(ProxyId(ev.Uint32()))
}

func (ev *Event) String() string {
	l := int(ev.Uint32())
	buf := ev.next(l)
	if len(buf) != l {
		panic("Unable to read string")
	}
	ret := string(bytes.TrimRight(buf, "\x00"))
	//padding to 32 bit boundary
	if (l & 0x3) != 0 {
		ev.next(4 - (l & 0x3))
	}
	return ret
}

func (ev *Event) Int32() int32 {
	return int32(ev.Uint32())
}

func (ev *Event) Float32() float32 {
	return float32(fixedToFloat64(ev.Int32()))
}

func (ev *Event) Array() []int32 {
	l := int(ev.Uint32())
	arr := make([]int32, l/4)
	for i := range arr {
		arr[i] = ev.Int32()
	}
	return arr
}

func (ev *Event) next(n int) []byte {
	ret := ev.data[ev.off : ev.off+n]
	ev.off += n
	return ret
}
