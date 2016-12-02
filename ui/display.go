package ui

import (
	"fmt"
	"log"
	"sync"
	"syscall"
)

import (
	wl ".."
)

type Display struct {
	mu                sync.RWMutex
	display           *wl.Display
	registry          *wl.Registry
	compositor        *wl.Compositor
	subCompositor     *wl.Subcompositor
	shell             *wl.Shell
	shm               *wl.Shm
	seat              *wl.Seat
	dataDeviceManager *wl.DataDeviceManager
	pointer           *wl.Pointer
	keyboard          *wl.Keyboard
	touch             *wl.Touch
	windows           []*Window
}

func Connect(addr string) (*Display, error) {
	d := new(Display)
	display, err := wl.Connect(addr)
	if err != nil {
		return nil, fmt.Errorf("Connect to Wayland server failed %s", err)
	}

	display.AddErrorHandler(d)

	d.display = display
	err = d.registerGlobals()
	if err != nil {
		return nil, err
	}
	d.checkGlobalsRegistered()

	err = d.registerInputs()
	if err != nil {
		return nil, err
	}
	d.checkInputsRegistered()

	return d, nil
}

func (d *Display) Disconnect() {
	d.keyboard.Release()
	d.pointer.Release()
	if d.touch != nil {
		d.touch.Release()
	}

	d.seat.Release()

	d.display.Context().Close()
}

func (d *Display) Dispatch() chan<- bool {
	return d.Context().Dispatch()
}

func (d *Display) Context() *wl.Context {
	return d.display.Context()
}

func (d *Display) registerGlobals() error {
	registry, err := d.display.GetRegistry()
	d.registry = registry
	if err != nil {
		return fmt.Errorf("Display.GetRegistry failed : %s", err)
	}
	callback, err := d.display.Sync()
	if err != nil {
		return fmt.Errorf("Display.Sync failed %s", err)
	}

	rgec := make(chan wl.RegistryGlobalEvent)
	f := func(ev interface{}) {
		if e, ok := ev.(wl.RegistryGlobalEvent); ok {
			rgec <- e
		}
	}
	evf := wl.HandlerFunc(f)
	registry.AddGlobalHandler(evf)

	dc := make(chan wl.CallbackDoneEvent)
	cdf := func(e interface{}) {
		if ev, ok := e.(wl.CallbackDoneEvent); ok {
			dc <- ev
		}
	}
	dh := wl.HandlerFunc(cdf)
	callback.AddDoneHandler(dh)
loop:
	for {
		select {
		case ev := <-rgec:
			if err := d.registerInterface(registry, ev); err != nil {
				return err
			}
		case d.Dispatch() <- true:
		case <-dc:
			break loop
		}
	}

	registry.RemoveGlobalHandler(evf)
	callback.RemoveDoneHandler(dh)
	return nil
}

func (d *Display) registerInputs() error {
	callback, err := d.display.Sync()
	if err != nil {
		return fmt.Errorf("Display.Sync failed %s", err)
	}

	dc := make(chan wl.CallbackDoneEvent)
	cbd := func(e interface{}) {
		if ev, ok := e.(wl.CallbackDoneEvent); ok {
			dc <- ev
		}
	}
	dh := wl.HandlerFunc(cbd)
	callback.AddDoneHandler(dh)

	scc := make(chan wl.SeatCapabilitiesEvent)
	scf := func(e interface{}) {
		if ev, ok := e.(wl.SeatCapabilitiesEvent); ok {
			scc <- ev
		}
	}
	ch := wl.HandlerFunc(scf)
	d.seat.AddCapabilitiesHandler(ch)
loop:
	for {
		select {
		case ev := <-scc:
			if (ev.Capabilities & wl.SeatCapabilityPointer) != 0 {
				pointer, err := d.seat.GetPointer()
				if err != nil {
					return fmt.Errorf("Unable to get Pointer object: %s", err)
				}
				d.pointer = pointer
			}
			if (ev.Capabilities & wl.SeatCapabilityKeyboard) != 0 {
				keyboard, err := d.seat.GetKeyboard()
				if err != nil {
					return fmt.Errorf("Unable to get Keyboard object: %s", err)
				}
				d.keyboard = keyboard
			}
			if (ev.Capabilities & wl.SeatCapabilityTouch) != 0 {
				touch, err := d.seat.GetTouch()
				if err != nil {
					return fmt.Errorf("Unable to get Touch object: %s", err)
				}
				d.touch = touch
			}
		case d.Dispatch() <- true:
		case <-dc:
			break loop
		}
	}
	d.seat.RemoveCapabilitiesHandler(ch)
	callback.RemoveDoneHandler(dh)
	return nil
}

func (d *Display) registerInterface(registry *wl.Registry, ev wl.RegistryGlobalEvent) error {
	switch ev.Interface {
	case "wl_shm":
		ret := wl.NewShm(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind Shm interface: %s", err)
		}
		d.shm = ret
	case "wl_compositor":
		ret := wl.NewCompositor(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind Compositor interface: %s", err)
		}
		d.compositor = ret
	case "wl_shell":
		ret := wl.NewShell(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind Shell interface: %s", err)
		}
		d.shell = ret
	case "wl_seat":
		ret := wl.NewSeat(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind Seat interface: %s", err)
		}
		d.seat = ret
	case "wl_data_device_manager":
		ret := wl.NewDataDeviceManager(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind DataDeviceManager interface: %s", err)
		}
		d.dataDeviceManager = ret
	case "wl_subcompositor":
		ret := wl.NewSubcompositor(d.Context())
		err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
		if err != nil {
			return fmt.Errorf("Unable to bind Subcompositor interface: %s", err)
		}
		d.subCompositor = ret
	}
	return nil
}

func (d *Display) Handle(e interface{}) {
	switch ev := e.(type) {
	case wl.DisplayErrorEvent:
		log.Fatalf("Display Error Event: %d - %s - %d", ev.ObjectId.Id(), ev.Message, ev.Code)
	default:
		log.Print("unhandled event")
	}
}

func (d *Display) newBuffer(width, height, stride int32) (*wl.Buffer, []byte, error) {
	size := stride * height

	file, err := TempFile(int64(size))
	if err != nil {
		return nil, nil, fmt.Errorf("TempFile failed: %s", err)
	}
	defer file.Close()

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("syscall.Mmap failed: %s", err)
	}

	pool, err := d.shm.CreatePool(file.Fd(), size)
	if err != nil {
		return nil, nil, fmt.Errorf("Shm.CreatePool failed: %s", err)
	}

	buf, err := pool.CreateBuffer(0, width, height, stride, wl.ShmFormatArgb8888)
	if err != nil {
		return nil, nil, fmt.Errorf("Pool.CreateBuffer failed : %s", err)
	}
	defer pool.Destroy()

	return buf, data, nil
}

func (d *Display) registerWindow(w *Window) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.windows = append(d.windows, w)
}

func (d *Display) unregisterWindow(w *Window) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, _w := range d.windows {
		if _w == w {
			d.windows = append(d.windows[:i], d.windows[i+1:]...)
			break
		}
	}
}

// TODO
func (d *Display) FindWindow() *Window {
	return nil
}

func (d *Display) checkGlobalsRegistered() {
	if d.seat == nil {
		log.Fatal("Seat is not registered")
	}

	if d.compositor == nil {
		log.Fatal("Compositor is not registered")
	}

	if d.shm == nil {
		log.Fatal("Shm is not registered")
	}

	if d.shell == nil {
		log.Fatal("Shell is not registered")
	}

	if d.dataDeviceManager == nil {
		log.Fatal("DataDeviceManager is not registered")
	}
}

func (d *Display) Keyboard() *wl.Keyboard {
	return d.keyboard
}

func (d *Display) Pointer() *wl.Pointer {
	return d.pointer
}

func (d *Display) Touch() *wl.Touch {
	return d.touch
}

func (d *Display) checkInputsRegistered() {
	if d.keyboard == nil {
		log.Fatal("Keyboard is not registered")
	}

	if d.pointer == nil {
		log.Fatal("Pointer is not registered")
	}
}
