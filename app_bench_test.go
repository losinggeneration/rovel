package rovel

import (
	"testing"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/backend/headless"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// discardCellSink accepts cell frames without retaining them, isolating the
// presenter's own allocation behavior.
type discardCellSink struct{}

func (discardCellSink) PresentCellFrame(backend.CellFrame) error { return nil }

func BenchmarkPostInvalidateBurst(b *testing.B) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	rects := []geom.Rect{
		{X: 0, Y: 0, W: 20, H: 1},
		{X: 0, Y: 1, W: 20, H: 1},
		{X: 0, Y: 2, W: 20, H: 1},
		{X: 0, Y: 3, W: 20, H: 1},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := app.PostInvalidate(rects[i%len(rects)]); err != nil {
			b.Fatalf("PostInvalidate failed: %v", err)
		}

		app.postMu.Lock()
		queue := app.postQueue
		app.postQueue = nil
		app.postMu.Unlock()

		ctx := app.mkUpdateCtx()
		for _, fn := range queue {
			fn(ctx)
		}

		app.invalidRects = app.invalidRects[:0]
	}
}

// BenchmarkMkCtx guards that the shared dispatch context is built once and
// reused, so per-event/per-paint context creation does not allocate.
func BenchmarkMkCtx(b *testing.B) {
	app, err := New(AppOpts{Backend: headless.New(geom.Size{W: 80, H: 24})})
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	if err := app.Enable(); err != nil {
		b.Fatalf("Enable: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = app.mkCtx()
	}
}

// BenchmarkCellFramePresent guards that presenting a cell frame reuses its
// cell buffer instead of allocating a W×H slice per frame.
func BenchmarkCellFramePresent(b *testing.B) {
	size := geom.Size{W: 80, H: 24}
	p := newCellFramePresenter(discardCellSink{})
	frame, ok := newCellRenderer(size).Frame(size, style.Style{}).(*cellFrame)

	if !ok {
		b.Fatal("Frame did not return *cellFrame")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := p.PresentFrame(frame); err != nil {
			b.Fatalf("PresentFrame: %v", err)
		}
	}
}
