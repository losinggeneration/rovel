package sdl3

import (
	"testing"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/event"
)

var (
	_ backend.Backend             = (*Core)(nil)
	_ backend.CapabilityReporter  = (*Core)(nil)
	_ backend.InputFeatureEnabler = (*Core)(nil)
	_ backend.CellFrameSink       = (*Core)(nil)
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.WindowWidth <= 0 || opts.WindowHeight <= 0 {
		t.Fatalf("window size = %dx%d, want positive values", opts.WindowWidth, opts.WindowHeight)
	}

	if opts.CellWidth <= 0 || opts.CellHeight <= 0 {
		t.Fatalf("cell size = %dx%d, want positive values", opts.CellWidth, opts.CellHeight)
	}

	if opts.FontSize <= 0 {
		t.Fatalf("font size = %d, want positive value", opts.FontSize)
	}

	if opts.DefaultFG.A != 0xFF || opts.DefaultBG.A != 0xFF {
		t.Fatalf("default colors should be opaque: fg=%+v bg=%+v", opts.DefaultFG, opts.DefaultBG)
	}
}

func TestCoreEnableAndResizeWindow(t *testing.T) {
	core := NewCore(Options{
		WindowWidth:  320,
		WindowHeight: 160,
		CellWidth:    8,
		CellHeight:   16,
	})

	size, err := core.Enable()
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}

	if size.W != 40 || size.H != 10 {
		t.Fatalf("initial size = %v, want {40 10}", size)
	}

	ev := core.ResizeWindow(640, 320)
	if ev.W != 80 || ev.H != 20 {
		t.Fatalf("ResizeWindow event = %+v, want {80 20}", ev)
	}

	read := core.ReadEvent()

	re, ok := read.(event.ResizeEvent)
	if !ok {
		t.Fatalf("ReadEvent() = %T, want ResizeEvent", read)
	}

	if re.W != 80 || re.H != 20 {
		t.Fatalf("ReadEvent resize = %+v, want {80 20}", re)
	}
}

func TestCoreMapMouseAndPresentFrame(t *testing.T) {
	core := NewCore(DefaultOptions())

	me := core.MapMouse(17, 33, event.MouseButtonLeft, event.MousePress, event.ModShift)
	if me.X != 2 || me.Y != 2 {
		t.Fatalf("MapMouse = %+v, want cell position 2,2", me)
	}

	frame := backend.CellFrame{
		W: 2,
		H: 1,
		Cells: []backend.FrameCell{
			{R: 'O'},
			{R: 'K'},
		},
	}
	if err := core.PresentCellFrame(frame); err != nil {
		t.Fatalf("PresentCellFrame: %v", err)
	}

	got, ok := core.LastFrame()
	if !ok {
		t.Fatal("LastFrame() returned no frame")
	}

	if got.W != 2 || got.H != 1 || string(got.RowRunes(0)) != "OK" {
		t.Fatalf("LastFrame = %+v / %q, want 2x1 and OK", got, string(got.RowRunes(0)))
	}
}

func TestCoreRefreshEmitsCurrentSize(t *testing.T) {
	core := NewCore(Options{
		WindowWidth:  640,
		WindowHeight: 320,
		CellWidth:    8,
		CellHeight:   16,
	})

	if _, err := core.Enable(); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	want := core.Size()

	ev := core.Refresh()
	if ev.W != want.W || ev.H != want.H {
		t.Fatalf("Refresh() = %+v, want current size %+v", ev, want)
	}

	read := core.ReadEvent()

	re, ok := read.(event.ResizeEvent)
	if !ok {
		t.Fatalf("ReadEvent() = %T, want ResizeEvent", read)
	}

	if re.W != want.W || re.H != want.H {
		t.Fatalf("ReadEvent refresh = %+v, want %+v", re, want)
	}
}
