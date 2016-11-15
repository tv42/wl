package wl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"syscall"
)

type Message struct {
	Id           ProxyId
	Opcode       uint32
	size         uint32
	data         *bytes.Buffer
	control      *bytes.Buffer
	control_msgs []syscall.SocketControlMessage
}

func ReadWaylandMessage(conn *net.UnixConn) (*Event, error) {
	var buf [8]byte
	control := make([]byte, 24)

	n, oobn, _, _, err := conn.ReadMsgUnix(buf[:], control)
	if err != nil {
		return nil, err
	}
	if n != 8 {
		return nil, errors.New("Unable to read message header.")
	}

	ev := new(Event)

	if oobn > 0 {
		if oobn > len(control) {
			panic("Unsufficient control msg buffer")
		}
		scms, err := syscall.ParseSocketControlMessage(control)
		if err != nil {
			log.Panicf("Control message parse error: %s", err)
		}
		ev.scms = scms
	}

	ev.pid = ProxyId(order.Uint32(buf[0:4]))
	ev.opcode = uint32(order.Uint16(buf[4:6]))
	size := uint32(order.Uint16(buf[6:8]))

	// subtract 8 bytes from header
	data := make([]byte, size-8)

	n, err = conn.Read(data)
	if err != nil {
		return nil, err
	}
	if n != int(size)-8 {
		return nil, errors.New("Invalid message size.")
	}
	ev.data = data //bytes.NewBuffer(data)
	return ev, nil
}

func (m *Message) Write(arg interface{}) error {
	switch t := arg.(type) {
	case Proxy:
		return binary.Write(m.data, order, uint32(t.Id()))
	case uint32, int32:
		return binary.Write(m.data, order, t)
	case float32:
		f := float64ToFixed(float64(t))
		return binary.Write(m.data, order, f)
	case string:
		tail := 4 - (len(t) & 0x3)
		err := binary.Write(m.data, order, uint32(len(t)+tail))
		if err != nil {
			return err
		}
		err = binary.Write(m.data, order, []byte(t))
		if err != nil {
			return err
		}
		// if padding required
		if tail > 0 {
			padding := make([]byte, tail)
			return binary.Write(m.data, order, padding)
		}
		return nil
	case []int32:
		err := binary.Write(m.data, order, uint32(len(t)))
		if err != nil {
			return err
		}
		return binary.Write(m.data, order, t)
	case uintptr:
		rights := syscall.UnixRights(int(t))
		return binary.Write(m.control, order, rights)
	default:
		panic("Invalid Wayland request parameter type.")
	}
}

func NewRequest(p Proxy, opcode uint32) *Message {
	msg := Message{}
	msg.Opcode = opcode
	msg.Id = p.Id()
	msg.data = &bytes.Buffer{}
	msg.control = &bytes.Buffer{}

	return &msg
}

func SendWaylandMessage(conn *net.UnixConn, m *Message) error {
	header := &bytes.Buffer{}
	// calculate message total size
	m.size = uint32(m.data.Len() + 8)
	binary.Write(header, order, uint32(m.Id))
	binary.Write(header, order, m.size<<16|m.Opcode&0x0000ffff)

	d, c, err := conn.WriteMsgUnix(append(header.Bytes(), m.data.Bytes()...), m.control.Bytes(), nil)
	if err != nil {
		panic(err)
	}
	if c != m.control.Len() || d != (header.Len()+m.data.Len()) {
		panic("WriteMsgUnix failed.")
	}
	return err
}
