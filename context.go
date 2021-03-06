package wl

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

func init() {
	log.SetFlags(0)
}

type Context struct {
	mu        sync.RWMutex
	conn      *net.UnixConn
	currentId ProxyId
	objects   map[ProxyId]Proxy
}

func (ctx *Context) Register(proxy Proxy) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.currentId += 1
	proxy.SetId(ctx.currentId)
	proxy.SetContext(ctx)
	ctx.objects[ctx.currentId] = proxy
}

func (ctx *Context) lookupProxy(id ProxyId) Proxy {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	proxy, ok := ctx.objects[id]
	if !ok {
		return nil
	}
	return proxy
}

func (c *Context) Close() {
	c.conn.CloseWrite()
}

func Connect(addr string) (ret *Display, err error) {
	runtime_dir := os.Getenv("XDG_RUNTIME_DIR")
	if runtime_dir == "" {
		return nil, errors.New("XDG_RUNTIME_DIR not set in the environment")
	}
	if addr == "" {
		addr = os.Getenv("WAYLAND_DISPLAY")
	}
	if addr == "" {
		addr = "wayland-0"
	}
	addr = runtime_dir + "/" + addr
	c := new(Context)
	c.objects = make(map[ProxyId]Proxy)
	c.currentId = 0
	c.conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: addr, Net: "unix"})
	if err != nil {
		return nil, err
	}
	//dispatch events in separate gorutine
	go c.run()
	return NewDisplay(c), nil
}

func (c *Context) run() {
	for {
		ev, err := c.readEvent()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}

		proxy := c.lookupProxy(ev.pid)
		if proxy != nil {
			if dispatcher, ok := proxy.(Dispatcher); ok {
				dispatcher.Dispatch(ev)
			} else {
				log.Print("Not dispatched")
			}
		} else {
			log.Print("Proxy NULL")
		}
	}
}
