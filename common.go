package wl

//go:generate go run github.com/dkolbly/wl-scanner -source https://cgit.freedesktop.org/wayland/wayland/plain/protocol/wayland.xml -output client.go

type ProxyId uint32

type Dispatcher interface {
	Dispatch(*Event)
}

// ensure this interface matches what the generated code does
var _ Dispatcher = (*Registry)(nil)

type Proxy interface {
	Context() *Context
	SetContext(c *Context)
	Id() ProxyId
	SetId(id ProxyId)
}

type BaseProxy struct {
	id  ProxyId
	ctx *Context
}

func (p *BaseProxy) Id() ProxyId {
	return p.id
}

func (p *BaseProxy) SetId(id ProxyId) {
	p.id = id
}

func (p *BaseProxy) Context() *Context {
	return p.ctx
}

func (p *BaseProxy) SetContext(c *Context) {
	p.ctx = c
}

type Handler interface {
	Handle(ev interface{})
}

type eventHandler struct {
	f func(interface{})
}

func HandlerFunc(f func(interface{})) Handler {
	return &eventHandler{f}
}

func (h *eventHandler) Handle(ev interface{}) {
	h.f(ev)
}
