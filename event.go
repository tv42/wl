package wl

import (
	"bytes"
	"syscall"
)

type Event struct {
	pid    ProxyId
	opcode uint32
	data   *bytes.Buffer
	scms   []syscall.SocketControlMessage
}

func (e *Event) Proxy(c *Connection) Proxy {
	return c.lookupProxy(ProxyId(e.Uint32()))
}

func (e *Event) FD() uintptr {
	if e.scms == nil {
		return 0
	}
	fds, err := syscall.ParseUnixRights(&e.scms[0])
	if err != nil {
		panic("Unable to parse unix rights")
	}
	e.scms = append(e.scms[0:], e.scms[1:]...)
	if len(fds) != 1 {
		panic("Expected 1 file descriptor, got more")
	}
	return uintptr(fds[0])
}

func (e *Event) String() string {
	l := int(e.Uint32())
	buf := e.data.Next(l)
	if len(buf) != l {
		panic("Unable to read string")
	}
	ret := string(bytes.TrimRight(buf, "\x00"))
	//padding to 32 bit boundary
	if (l & 0x3) != 0 {
		e.data.Next(4 - (l & 0x3))
	}
	return ret
}

func (e *Event) Int32() int32 {
	return int32(e.Uint32())
}

func (e *Event) Uint32() uint32 {
	buf := e.data.Next(4)
	if len(buf) != 4 {
		panic("Unable to read unsigned int")
	}
	return order.Uint32(buf)
}

func (e *Event) Float32() float32 {
	return float32(fixedToFloat64(e.Int32()))
}

func (e *Event) Array() []int32 {
	l := e.Uint32()
	arr := make([]int32, l/4)
	for i := range arr {
		arr[i] = e.Int32()
	}
	return arr
}
