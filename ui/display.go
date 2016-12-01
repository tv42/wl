package ui

import (
	"fmt"
	"log"
)

import (
	wl ".."
)

type Display struct {
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
	log.Println("Globals registered")
	err = d.registerInputs()
	if err != nil {
		return nil, err
	}

	if d.keyboard == nil {
		log.Fatal("Keyboard == NULL")
	}

	log.Println("Inputs registered")
	return d, nil
}

func (d *Display) Disconnect() {
	d.display.Context().Close()
}

func (d *Display) Dispatch() chan<- bool {
	return d.Context().Dispatch()
}

func (d *Display) Context() *wl.Context {
	return d.display.Context()
}

func (d *Display) Keyboard() *wl.Keyboard {
	return d.keyboard
}

func (d *Display) Pointer() *wl.Pointer {
	return d.pointer
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
	if d.seat == nil {
		log.Fatal("seat == NİL")
	}

	d.seat.AddCapabilitiesHandler(ch)
loop:
	for {
		//log.Print("in registerInputs loop")
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
		/*
			case "text_cursor_position":
				ret := wl.NewTextCursorPosition(d.Context())
				err := registry.Bind(ev.Name, ev.Interface, ev.Version, ret)
				if err != nil {
					return fmt.Errorf("Unable to bind TextCursorPosition interface: %s", err)
				}
				d.wltextCursorPosition = ret
		*/
		/*
			default:
				log.Printf("%s interface is not registered", ev.Interface)
		*/
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
