package tui

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
)

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
