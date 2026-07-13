package rovel

import (
	"errors"
	"testing"

	"github.com/losinggeneration/rovel/backend/headless"
	"github.com/losinggeneration/rovel/geom"
)

// clipboardErrBackend is a headless backend that also implements
// backend.ClipboardBackend, returning a fixed error from ClipboardWrite.
type clipboardErrBackend struct {
	*headless.Backend

	err error
}

func (b *clipboardErrBackend) ClipboardWrite(string) error    { return b.err }
func (b *clipboardErrBackend) ClipboardRead() (string, error) { return "", nil }

// A clipboard write failure must be routed into the error buffer instead of
// being silently swallowed.
func TestClipboardWriteError_RoutedToErrbuf(t *testing.T) {
	wantErr := errors.New("clipboard boom")
	be := &clipboardErrBackend{Backend: headless.New(geom.Size{W: 80, H: 24}), err: wantErr}

	app := enableSignalApp(t, be, newCountingRoot())
	defer be.Close()

	ctx := app.mkCtx()
	if ctx.ClipboardWrite == nil {
		t.Fatal("ClipboardWrite closure should be wired")
	}

	ctx.ClipboardWrite("hello")

	if !containsErr(app.Errors(), wantErr) {
		t.Fatalf("clipboard error not routed to errbuf: %v", app.Errors())
	}
}

// A successful clipboard write records nothing.
func TestClipboardWriteSuccess_NoError(t *testing.T) {
	be := &clipboardErrBackend{Backend: headless.New(geom.Size{W: 80, H: 24}), err: nil}

	app := enableSignalApp(t, be, newCountingRoot())
	defer be.Close()

	app.mkCtx().ClipboardWrite("hello")

	if len(app.Errors()) != 0 {
		t.Fatalf("successful clipboard write recorded errors: %v", app.Errors())
	}
}

// TerminalSize reports the physical terminal size while Size reports the
// (possibly smaller) render region.
func TestTerminalSizeVsRenderSize(t *testing.T) {
	be := headless.New(geom.Size{W: 80, H: 24})

	app, err := New(AppOpts{Backend: be, RenderSize: geom.Size{W: 40, H: 10}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newCountingRoot())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	defer be.Close()

	if got := app.Size(); got != (geom.Size{W: 40, H: 10}) {
		t.Errorf("Size() = %v, want render region 40x10", got)
	}

	if got := app.TerminalSize(); got != (geom.Size{W: 80, H: 24}) {
		t.Errorf("TerminalSize() = %v, want terminal 80x24", got)
	}
}

func containsErr(errs []error, target error) bool {
	for _, e := range errs {
		if errors.Is(e, target) {
			return true
		}
	}

	return false
}
