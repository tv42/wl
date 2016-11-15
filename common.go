package wl

type ProxyId uint32

type EventDispatcher interface {
	Dispatch(*Event)
}

type Disposer interface {
	Dispose()
}

type Proxy interface {
	Connection() *Connection
	SetConnection(c *Connection)
	Id() ProxyId
	SetId(id ProxyId)
}

type BaseProxy struct {
	id   ProxyId
	conn *Connection
}

func (p *BaseProxy) Id() ProxyId {
	return p.id
}

func (p *BaseProxy) SetId(id ProxyId) {
	p.id = id
}

func (p *BaseProxy) Connection() *Connection {
	return p.conn
}

func (p *BaseProxy) SetConnection(c *Connection) {
	p.conn = c
}
