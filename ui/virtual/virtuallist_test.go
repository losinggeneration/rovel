package virtual

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
)

func TestNotifyCountChanged(t *testing.T) {
	tests := []struct {
		name              string
		initialCount      int
		newCount          int
		initialSelection  int
		initialScroll     int
		expectInvalidated bool
		expectSelection   int
		expectScroll      int
	}{
		{
			name:              "Count decreases, scroll is clamped",
			initialCount:      100,
			newCount:          50,
			initialSelection:  75,
			initialScroll:     60,
			expectInvalidated: true,
			expectSelection:   49, // Clamped to new max
			expectScroll:      30, // Clamped to new max (50 - 20 visible)
		},
		{
			name:              "Count increases, no clamping needed",
			initialCount:      50,
			newCount:          100,
			initialSelection:  25,
			initialScroll:     10,
			expectInvalidated: true, // Count changed = must invalidate
			expectSelection:   25,
			expectScroll:      10,
		},
		{
			name:              "Count goes to zero, selection cleared",
			initialCount:      100,
			newCount:          0,
			initialSelection:  50,
			initialScroll:     40,
			expectInvalidated: true,
			expectSelection:   -1, // Cleared
			expectScroll:      0,  // Reset
		},
		{
			name:              "Selection beyond new count is clamped",
			initialCount:      100,
			newCount:          10,
			initialSelection:  50,
			initialScroll:     40,
			expectInvalidated: true,
			expectSelection:   9, // Clamped to new max
			expectScroll:      0, // Clamped
		},
		{
			name:              "Scroll beyond new count is clamped",
			initialCount:      100,
			newCount:          10,
			initialSelection:  5,
			initialScroll:     50,
			expectInvalidated: true,
			expectSelection:   5,
			expectScroll:      0, // Clamped to max (10 - 20 visible = 0)
		},
		{
			name:              "Count 0 to 1 always invalidates",
			initialCount:      0,
			newCount:          1,
			initialSelection:  -1,
			initialScroll:     0,
			expectInvalidated: true,
			expectSelection:   -1,
			expectScroll:      0,
		},
		{
			name:              "Count 1 to 0 always invalidates",
			initialCount:      1,
			newCount:          0,
			initialSelection:  0,
			initialScroll:     0,
			expectInvalidated: true,
			expectSelection:   -1,
			expectScroll:      0,
		},
		{
			name:              "Append at tail with no scroll/selection change",
			initialCount:      100,
			newCount:          150,
			initialSelection:  10,
			initialScroll:     5,
			expectInvalidated: true,
			expectSelection:   10,
			expectScroll:      5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up count variable that can be changed
			count := tt.initialCount

			v := NewVirtualList(VirtualListOpts{
				RowHeight: 1,
				Count:     func() int { return count },
				RenderRow: mockRenderRow("Item"),
			})

			// Layout with space for 20 visible items
			v.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 20})

			// Set initial state
			if tt.initialSelection >= 0 {
				v.SelectIndex(nil, tt.initialSelection)
			}
			v.SetScrollItem(nil, tt.initialScroll)

			// Track invalidation
			invalidated := false
			ctx := &tui.Ctx{
				Invalidate: func(r geom.Rect) {
					invalidated = true
				},
			}

			// Change count
			count = tt.newCount
			v.NotifyCountChanged(ctx)

			// Check invalidation
			if invalidated != tt.expectInvalidated {
				t.Errorf("Expected invalidated=%v, got %v", tt.expectInvalidated, invalidated)
			}

			// Check selection
			if v.SelectedIndex() != tt.expectSelection {
				t.Errorf("Expected selection=%d, got %d", tt.expectSelection, v.SelectedIndex())
			}

			// Check scroll
			if v.ScrollItem() != tt.expectScroll {
				t.Errorf("Expected scroll=%d, got %d", tt.expectScroll, v.ScrollItem())
			}
		})
	}
}

func TestNotifyCountChangedWithNilContext(t *testing.T) {
	count := 100
	v := NewVirtualList(VirtualListOpts{
		RowHeight: 1,
		Count:     func() int { return count },
		RenderRow: mockRenderRow("Item"),
	})

	v.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 20})
	v.SelectIndex(nil, 50)
	v.SetScrollItem(nil, 40)

	// Should not panic with nil context
	count = 50
	v.NotifyCountChanged(nil)

	// State should still be updated correctly
	if v.SelectedIndex() != 49 {
		t.Errorf("Expected selection to be clamped to 49, got %d", v.SelectedIndex())
	}
}

func TestNotifyCountChangedWithNilInvalidate(t *testing.T) {
	count := 100
	v := NewVirtualList(VirtualListOpts{
		RowHeight: 1,
		Count:     func() int { return count },
		RenderRow: mockRenderRow("Item"),
	})

	v.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 20})
	v.SelectIndex(nil, 50)
	v.SetScrollItem(nil, 40)

	// Context with nil Invalidate func
	ctx := &tui.Ctx{}

	count = 50
	v.NotifyCountChanged(ctx)

	// State should still be updated correctly
	if v.SelectedIndex() != 49 {
		t.Errorf("Expected selection to be clamped to 49, got %d", v.SelectedIndex())
	}
}

func TestNotifyCountChangedScrollSelectionIntoView(t *testing.T) {
	count := 100
	v := NewVirtualList(VirtualListOpts{
		RowHeight: 1,
		Count:     func() int { return count },
		RenderRow: mockRenderRow("Item"),
	})

	// Layout with space for 10 visible items
	v.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	// Set selection to item 95 (near end)
	v.SelectIndex(nil, 95)
	// Scroll should be adjusted to show item 95
	if v.ScrollItem() < 85 {
		t.Errorf("Expected scroll to be at least 85 to show item 95, got %d", v.ScrollItem())
	}

	invalidated := false
	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {
			invalidated = true
		},
	}

	// Reduce count so selection needs to be clamped
	count = 50
	v.NotifyCountChanged(ctx)

	// Selection should be clamped to 49 (max index for count=50)
	if v.SelectedIndex() != 49 {
		t.Errorf("Expected selection=49, got %d", v.SelectedIndex())
	}

	// Scroll should be adjusted to keep selection visible
	// With count=50 and height=10, max scroll is 40
	if v.ScrollItem() > 40 {
		t.Errorf("Expected scroll <= 40, got %d", v.ScrollItem())
	}

	if !invalidated {
		t.Error("Expected invalidation when count changed and selection was clamped")
	}
}
