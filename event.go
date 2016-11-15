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
}

type readScm []syscall.SocketControlMessage

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
	ret := order.Uint32(ev.data)
	ev.data = ev.data[4:]
	return ret
}

func (ev *Event) Proxy(c *Connection) Proxy {
	pid := ProxyId(ev.Uint32())
	return c.lookupProxy(pid)
}

func (ev *Event) String() string {
	l := int(ev.Uint32())
	str := ev.data[:l]
	ret := string(bytes.TrimRight(str, "\x00"))
	//l++
	//padding to 32 bit boundary
	l += (4 - (l & 0x3))
	ev.data = ev.data[l:]
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
