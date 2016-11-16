package wl

import (
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func init() {
	log.SetFlags(0)
}

type Connection struct {
	mu           sync.RWMutex
	conn         *net.UnixConn
	currentId    ProxyId
	objects      map[ProxyId]Proxy
	dispatchChan chan bool
	exitChan     chan bool
}

func (context *Connection) Register(proxy Proxy) {
	context.mu.Lock()
	defer context.mu.Unlock()
	context.currentId += 1
	proxy.SetId(context.currentId)
	proxy.SetConnection(context)
	context.objects[context.currentId] = proxy
}

func (context *Connection) lookupProxy(id ProxyId) Proxy {
	context.mu.RLock()
	proxy, ok := context.objects[id]
	context.mu.RUnlock()
	if !ok {
		return nil
	}
	return proxy
}

func (context *Connection) Unregister(proxy Proxy) {
	context.mu.Lock()
	defer context.mu.Unlock()
	delete(context.objects, proxy.Id())
}

func (context *Connection) Close() error {
	if context.conn == nil {
		return errors.New("Wayland connection not established.")
	}
	context.conn.Close()
	context.exitChan <- true
	return nil
}

func (context *Connection) Dispatch() chan<- bool {
	return context.dispatchChan
}

func ConnectDisplay(addr string) (ret *Display, err error) {
	runtime_dir := os.Getenv("XDG_RUNTIME_DIR")
	if runtime_dir == "" {
		return nil, errors.New("XDG_RUNTIME_DIR not set in the environment.")
	}
	if addr == "" {
		addr = os.Getenv("WAYLAND_DISPLAY")
	}
	if addr == "" {
		addr = "wayland-0"
	}
	addr = runtime_dir + "/" + addr
	ctx := &Connection{}
	ctx.objects = make(map[ProxyId]Proxy)
	ctx.currentId = 0
	ctx.dispatchChan = make(chan bool)
	ctx.exitChan = make(chan bool)
	ctx.conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: addr, Net: "unix"})
	if err != nil {
		return nil, err
	}
	ret = NewDisplay(ctx)
	// dispatch events in separate gorutine
	go ctx.run()
	return ret, nil
}

func (context *Connection) SendRequest(proxy Proxy, opcode uint32, args ...interface{}) (err error) {
	if context.conn == nil {
		return errors.New("No wayland connection established for Proxy object.")
	}
	msg := NewRequest(proxy, opcode)

	for _, arg := range args {
		if err = msg.Write(arg); err != nil {
			return err
		}
	}

	return SendWaylandMessage(context.conn, msg)
}

//var dispatched int
func (context *Connection) run() {
	context.conn.SetReadDeadline(time.Time{})
loop:
	for {
		select {
		case <-context.dispatchChan:
			ev, err := ReadWaylandMessage(context.conn)
			if err != nil {
				//log.Printf("ReadWaylandMessage Err:%s", err)
				//read unix @->/run/user/1000/wayland-0: use of closed network connection
				continue
			}

			proxy := context.lookupProxy(ev.pid)
			if proxy != nil {
				if dispatcher, ok := proxy.(EventDispatcher); ok {
					dispatcher.Dispatch(ev)
					//					dispatched++
					//				log.Printf("%d dispatched",dispatched)
				} else {
					log.Println("Not Dispatched")
				}
			} else {
				log.Println("Proxy NULL")
			}
		case <-context.exitChan:
			break loop
		}
	}
}
