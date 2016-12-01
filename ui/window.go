package ui

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"syscall"
)

import (
	wl ".."
)

type Window struct {
	width, height int32
	surface       *wl.Surface
	shSurface     *wl.ShellSurface
	buffer        *wl.Buffer
	data          []byte
	image         *image.RGBA
}

func (d *Display) NewWindow(width, height int32) (*Window, error) {
	var err error
	stride := width * 4

	w := new(Window)
	w.width = width
	w.height = height

	w.surface, err = d.compositor.CreateSurface()
	if err != nil {
		return nil, fmt.Errorf("Surface creation failed: %s", err)
	}

	w.buffer, w.data, err = d.newBuffer(width, height, stride)
	if err != nil {
		return nil, err
	}

	w.shSurface, err = d.shell.GetShellSurface(w.surface)
	if err != nil {
		return nil, fmt.Errorf("Shell.GetShellSurface failed: %s", err)
	}

	w.shSurface.AddPingHandler(w)

	w.shSurface.SetToplevel()

	err = w.surface.Attach(w.buffer, width, height)
	if err != nil {
		return nil, fmt.Errorf("Surface.Attach failed: %s", err)
	}

	err = w.surface.Damage(0, 0, width, height)
	if err != nil {
		return nil, fmt.Errorf("Surface.Damage failed: %s", err)
	}

	err = w.surface.Commit()
	if err != nil {
		return nil, fmt.Errorf("Surface.Commit failed: %s", err)
	}

	w.image = &image.RGBA{
		Pix:    w.data,
		Stride: int(stride),
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	return w, nil
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

func (w *Window) Draw(img image.Image) {
	draw.Draw(w.image, img.Bounds(), img, img.Bounds().Min, draw.Src)
	//draw.Draw(w.image, img.Bounds(), img, image.ZP, draw.Src)
	BGRA(w.image.Pix)
}

func (w *Window) Dispose() {
	w.shSurface.RemovePingHandler(w)
	w.surface.Destroy()
	w.buffer.Destroy()
	syscall.Munmap(w.data)
}

func (w *Window) Handle(e interface{}) {
	switch ev := e.(type) {
	case wl.ShellSurfacePingEvent:
		w.shSurface.Pong(ev.Serial)
	default:
		log.Print("unhandled event")
	}
}
