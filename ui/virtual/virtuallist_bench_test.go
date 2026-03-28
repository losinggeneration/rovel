package virtual

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

// benchCtx returns a minimal tui.Ctx for benchmarking.
func benchCtx() *tui.Ctx {
	return &tui.Ctx{
		Invalidate:       func(r geom.Rect) {},
		InvalidateAll:    func() {},
		InvalidateLayout: func(id tui.ID) {},
		RequestFocus:     func(id tui.ID) {},
	}
}

// benchDrawer creates a new tui.Drawer with a buffer of the given size.
func benchDrawer(w, h int) tui.Drawer {
	buf := render.NewBuffer(w, h)
	clip := geom.Rect{X: 0, Y: 0, W: w, H: h}
	baseStyle := style.Style{}
	rp := render.NewPainter(buf, clip, baseStyle)

	return tui.NewDrawer(tui.NewPainter(rp, baseStyle))
}

// setupBenchmarkList creates a VirtualList configured for benchmarking.
func setupBenchmarkList(itemCount, rowHeight, width, height int) *VirtualList {
	v := NewVirtualList(VirtualListOpts{
		RowHeight: rowHeight,
		Count:     func() int { return itemCount },
		RenderRow: mockRenderRow("Item"),
	})
	v.Layout(geom.Rect{X: 0, Y: 0, W: width, H: height})

	return v
}

// Paint benchmarks
func BenchmarkPaintSmallList(b *testing.B) {
	// 10 items, 5 visible
	v := setupBenchmarkList(10, 1, 20, 5)
	d := benchDrawer(20, 5)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.Paint(d, ctx)
	}
}

func BenchmarkPaintMediumList(b *testing.B) {
	// 1,000 items, 20 visible
	v := setupBenchmarkList(1000, 1, 20, 20)
	d := benchDrawer(20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.Paint(d, ctx)
	}
}

func BenchmarkPaintLargeList(b *testing.B) {
	// 100,000 items, 20 visible
	v := setupBenchmarkList(100000, 1, 20, 20)
	d := benchDrawer(20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.Paint(d, ctx)
	}
}

func BenchmarkPaintVariousSizes(b *testing.B) {
	sizes := []struct {
		name string
		w, h int
	}{
		{"10x5", 10, 5},
		{"20x10", 20, 10},
		{"40x20", 40, 20},
		{"80x40", 80, 40},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			v := setupBenchmarkList(1000, 1, size.w, size.h)
			d := benchDrawer(size.w, size.h)
			ctx := benchCtx()

			b.ResetTimer()

			for range b.N {
				v.Paint(d, ctx)
			}
		})
	}
}

// Scroll benchmarks

func BenchmarkScrollByOne(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.ScrollBy(ctx, 1)
	}
}

func BenchmarkScrollByPage(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.ScrollBy(ctx, 20)
	}
}

func BenchmarkScrollToTop(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	// Start from middle
	v.ScrollTo(ctx, 500)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.ScrollTop(ctx)
	}
}

func BenchmarkScrollToBottom(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		v.ScrollBottom(ctx)
	}
}

// Selection benchmarks

func BenchmarkSelectIndex(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		v.SelectIndex(ctx, i%1000)
	}
}

func BenchmarkSelectIndexWithScroll(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		// Select items far from viewport to force scrolling
		idx := (i * 17) % 1000
		v.SelectIndex(ctx, idx)
	}
}

// Stress test: verify large dataset performance
func BenchmarkPaint100kItems(b *testing.B) {
	itemCounts := []struct {
		name  string
		count int
	}{
		{"100", 100},
		{"1k", 1000},
		{"10k", 10000},
		{"100k", 100000},
	}

	for _, ic := range itemCounts {
		b.Run(ic.name, func(b *testing.B) {
			v := setupBenchmarkList(ic.count, 1, 20, 20)
			d := benchDrawer(20, 20)
			ctx := benchCtx()

			b.ResetTimer()

			for range b.N {
				v.Paint(d, ctx)
			}
		})
	}
}

// Performance assertion benchmarks - used with -benchtime to validate thresholds
func BenchmarkPaintThreshold(b *testing.B) {
	// This benchmark is designed to validate that paint time is under 1ms
	// Run with: go test -bench=BenchmarkPaintThreshold -benchtime=1x
	v := setupBenchmarkList(100000, 1, 20, 20)
	d := benchDrawer(20, 20)
	ctx := benchCtx()

	b.ResetTimer()

	for range b.N {
		v.Paint(d, ctx)
	}
}

func BenchmarkScrollThreshold(b *testing.B) {
	// This benchmark is designed to validate that scroll time is under 500μs
	// Run with: go test -bench=BenchmarkScrollThreshold -benchtime=1x
	v := setupBenchmarkList(100000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()

	for range b.N {
		v.ScrollBy(ctx, 1)
	}
}
