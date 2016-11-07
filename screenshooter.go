package wl

const (
	_WESTON_SCREENSHOOTER_SHOOT = 0
)

type WestonScreenshooter struct {
	BaseProxy
	DoneChan chan WestonScreenshooterDoneEvent
}

type WestonScreenshooterDoneEvent struct {
}

func NewWestonScreenshooter(conn *Connection) *WestonScreenshooter {
	ret := new(WestonScreenshooter)
	ret.DoneChan = make(chan WestonScreenshooterDoneEvent)
	conn.Register(ret)
	return ret
}

func (p *WestonScreenshooter) Shoot(output Output, buffer Buffer) error {
	return p.Connection().SendRequest(p, _WESTON_SCREENSHOOTER_SHOOT, output, buffer)
}
