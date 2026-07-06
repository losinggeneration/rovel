package tui

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/backend/headless"
	"github.com/losinggeneration/tui/geom"
)

// signalBackend wraps a headless backend and implements
// backend.SignalController so suspend/terminate orchestration can be exercised
// without a real terminal (a real SIGTSTP would stop the test binary).
type signalBackend struct {
	*headless.Backend

	sigCh        chan backend.LifecycleSignal
	suspendCalls atomic.Int32
	suspendErr   error
	suspended    chan struct{}             // signalled after each Suspend call
	sizeOverride atomic.Pointer[geom.Size] // when set, Size() returns this
}

func newSignalBackend(size geom.Size) *signalBackend {
	return &signalBackend{
		Backend:   headless.New(size),
		sigCh:     make(chan backend.LifecycleSignal, 1),
		suspended: make(chan struct{}, 1),
	}
}

func (b *signalBackend) Signals() <-chan backend.LifecycleSignal { return b.sigCh }

func (b *signalBackend) Suspend() error {
	b.suspendCalls.Add(1)

	select {
	case b.suspended <- struct{}{}:
	default:
	}

	return b.suspendErr
}

// Size reports the override size when set (simulating a resize while stopped),
// otherwise the underlying headless size.
func (b *signalBackend) Size() geom.Size {
	if s := b.sizeOverride.Load(); s != nil {
		return *s
	}

	return b.Backend.Size()
}

// countingRoot counts Paint calls so tests can observe repaints.
type countingRoot struct {
	id     ID
	rect   geom.Rect
	paints atomic.Int32
}

func newCountingRoot() *countingRoot { return &countingRoot{id: NewID()} }

func (r *countingRoot) ID() ID                        { return r.id }
func (r *countingRoot) MinSize() geom.Size            { return geom.Size{W: 1, H: 1} }
func (r *countingRoot) Layout(gr geom.Rect)           { r.rect = gr }
func (r *countingRoot) Rect() geom.Rect               { return r.rect }
func (r *countingRoot) Paint(d Drawer, ctx *Ctx)      { r.paints.Add(1) }
func (r *countingRoot) Handle(e Event, ctx *Ctx) bool { return false }

func enableSignalApp(t *testing.T, be backend.Backend, root View) *App {
	t.Helper()

	app, err := New(AppOpts{Backend: be})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(root)

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	return app
}

func TestSignal_SuspendOrchestration(t *testing.T) {
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	root := newCountingRoot()
	app := enableSignalApp(t, be, root)

	paintsBefore := root.paints.Load()

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	be.sigCh <- backend.SignalSuspend

	select {
	case <-be.suspended:
	case <-time.After(2 * time.Second):
		t.Fatal("Suspend was not called after SignalSuspend")
	}

	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}

	if got := be.suspendCalls.Load(); got != 1 {
		t.Fatalf("Suspend called %d times, want 1", got)
	}

	// Suspend must trigger a repaint on resume.
	if root.paints.Load() <= paintsBefore {
		t.Fatalf("no repaint after resume: paints=%d, before=%d", root.paints.Load(), paintsBefore)
	}
}

func TestSignal_SuspendResizesOnResume(t *testing.T) {
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	newSize := geom.Size{W: 100, H: 40}
	be.sizeOverride.Store(&newSize)

	root := newCountingRoot()
	app := enableSignalApp(t, be, root)

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	be.sigCh <- backend.SignalSuspend

	select {
	case <-be.suspended:
	case <-time.After(2 * time.Second):
		t.Fatal("Suspend was not called")
	}

	app.Quit()
	be.Close()
	<-done

	// After resume with a changed size, the render region tracks the terminal.
	if app.size != newSize {
		t.Fatalf("render size after resume = %v, want %v", app.size, newSize)
	}

	if app.terminalSize != newSize {
		t.Fatalf("terminal size after resume = %v, want %v", app.terminalSize, newSize)
	}
}

func TestSignal_SuspendErrorStopsRun(t *testing.T) {
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	wantErr := errors.New("suspend boom")
	be.suspendErr = wantErr

	app := enableSignalApp(t, be, newCountingRoot())

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	be.sigCh <- backend.SignalSuspend

	select {
	case err := <-done:
		if !errors.Is(err, wantErr) {
			t.Fatalf("Run error = %v, want %v", err, wantErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Suspend error")
	}

	be.Close()
}

func TestSignal_TerminateQuitsRun(t *testing.T) {
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	app := enableSignalApp(t, be, newCountingRoot())

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	be.sigCh <- backend.SignalTerminate

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on terminate: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after SignalTerminate")
	}

	if be.suspendCalls.Load() != 0 {
		t.Fatalf("Suspend called on terminate: %d", be.suspendCalls.Load())
	}

	be.Close()
}

func TestSignal_AppSuspendFromPost(t *testing.T) {
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	app := enableSignalApp(t, be, newCountingRoot())

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	if err := app.Post(func(ctx *UpdateCtx) {
		if err := app.Suspend(); err != nil {
			t.Errorf("App.Suspend from post: %v", err)
		}
	}); err != nil {
		t.Fatalf("Post: %v", err)
	}

	select {
	case <-be.suspended:
	case <-time.After(2 * time.Second):
		t.Fatal("Suspend was not called from App.Suspend")
	}

	app.Quit()
	be.Close()
	<-done

	if be.suspendCalls.Load() != 1 {
		t.Fatalf("Suspend called %d times, want 1", be.suspendCalls.Load())
	}
}

func TestSignal_SuspendUnsupportedBackend(t *testing.T) {
	// Plain headless backend does not implement SignalController.
	be := headless.New(geom.Size{W: 80, H: 24})
	app := enableSignalApp(t, be, newCountingRoot())

	if app.signalCh != nil {
		t.Fatal("signalCh should be nil for a backend without signal support")
	}

	if err := app.Suspend(); !errors.Is(err, ErrSuspendUnsupported) {
		t.Fatalf("App.Suspend on unsupported backend = %v, want ErrSuspendUnsupported", err)
	}
}

func TestSignal_CtxSuspendWiring(t *testing.T) {
	// Suspendable backend: ctx.Suspend is wired.
	be := newSignalBackend(geom.Size{W: 80, H: 24})
	app := enableSignalApp(t, be, newCountingRoot())

	if ctx := app.mkCtx(nil); ctx.Suspend == nil {
		t.Fatal("ctx.Suspend should be non-nil for a suspendable backend")
	}

	// Non-suspendable backend: ctx.Suspend is nil.
	plain := headless.New(geom.Size{W: 80, H: 24})
	app2 := enableSignalApp(t, plain, newCountingRoot())

	if ctx := app2.mkCtx(nil); ctx.Suspend != nil {
		t.Fatal("ctx.Suspend should be nil for a non-suspendable backend")
	}
}
