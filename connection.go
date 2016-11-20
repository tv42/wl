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
	mu        sync.RWMutex
	conn      *net.UnixConn
	currentId ProxyId
	objects   map[ProxyId]Proxy
	dispatchChan chan bool
	exitChan chan bool
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

func (c *Connection) Close() {
	c.conn.Close()
	c.exitChan <- true
	close(c.dispatchChan)
}

func (c *Connection) Dispatch() chan<- bool {
	return c.dispatchChan
}

func Connect(addr string) (ret *Display, err error) {
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
	c := new(Connection)
	c.objects = make(map[ProxyId]Proxy)
	c.currentId = 0
	c.dispatchChan = make(chan bool)
	c.exitChan = make(chan bool)
	c.conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: addr, Net: "unix"})
	if err != nil {
		return nil, err
	}
	c.conn.SetReadDeadline(time.Time{})
	//dispatch events in separate gorutine
	go c.run()
	return NewDisplay(c) , nil
}

func (c *Connection) run() {
loop:
	for {
		select {
		case <-c.dispatchChan:
			ev, err := c.readEvent()
			if err != nil {
				continue
			}

			proxy := c.lookupProxy(ev.pid)
			if proxy != nil {
				if dispatcher, ok := proxy.(EventDispatcher); ok {
					dispatcher.Dispatch(ev)
					bytePool.Give(ev.data)
				} else {
					log.Print("Not dispatched")
				}
			} else {
				log.Print("Proxy NULL")
			}

		case <-c.exitChan:
			break loop
		}
	}
}
