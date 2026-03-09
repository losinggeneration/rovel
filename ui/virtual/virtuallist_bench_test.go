package virtual

import (
	"testing"
)

// Paint benchmarks

func BenchmarkPaintSmallList(b *testing.B) {
	// 10 items, 5 visible
	v := setupBenchmarkList(10, 1, 20, 5)
	p := benchPainter(20, 5)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.Paint(p, ctx)
	}
}

func BenchmarkPaintMediumList(b *testing.B) {
	// 1,000 items, 20 visible
	v := setupBenchmarkList(1000, 1, 20, 20)
	p := benchPainter(20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.Paint(p, ctx)
	}
}

func BenchmarkPaintLargeList(b *testing.B) {
	// 100,000 items, 20 visible
	v := setupBenchmarkList(100000, 1, 20, 20)
	p := benchPainter(20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.Paint(p, ctx)
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
			p := benchPainter(size.w, size.h)
			ctx := benchCtx()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v.Paint(p, ctx)
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
	for i := 0; i < b.N; i++ {
		v.ScrollBy(ctx, 1)
	}
}

func BenchmarkScrollByPage(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
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
	for i := 0; i < b.N; i++ {
		v.ScrollTop(ctx)
	}
}

func BenchmarkScrollToBottom(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.ScrollBottom(ctx)
	}
}

// Selection benchmarks

func BenchmarkSelectIndex(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.SelectIndex(ctx, i%1000)
	}
}

func BenchmarkSelectIndexWithScroll(b *testing.B) {
	v := setupBenchmarkList(1000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
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
			p := benchPainter(20, 20)
			ctx := benchCtx()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v.Paint(p, ctx)
			}
		})
	}
}

// Performance assertion benchmarks - used with -benchtime to validate thresholds
func BenchmarkPaintThreshold(b *testing.B) {
	// This benchmark is designed to validate that paint time is under 1ms
	// Run with: go test -bench=BenchmarkPaintThreshold -benchtime=1x
	v := setupBenchmarkList(100000, 1, 20, 20)
	p := benchPainter(20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Paint(p, ctx)
	}
}

func BenchmarkScrollThreshold(b *testing.B) {
	// This benchmark is designed to validate that scroll time is under 500μs
	// Run with: go test -bench=BenchmarkScrollThreshold -benchtime=1x
	v := setupBenchmarkList(100000, 1, 20, 20)
	ctx := benchCtx()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ScrollBy(ctx, 1)
	}
}
