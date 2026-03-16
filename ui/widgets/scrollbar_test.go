package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

func TestScrollbar_ThumbHeight(t *testing.T) {
	tests := []struct {
		name        string
		contentSize int
		viewSize    int
		rectH       int
		wantThumbH  int
	}{
		{"half viewport", 20, 10, 10, 5},
		{"tiny content ratio", 200, 10, 20, 1}, // clamped to minimum 1
		{"content equals view", 10, 10, 10, 0}, // no scrollbar needed
		{"content less than view", 5, 10, 10, 0},
		{"full height", 10, 10, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewScrollbar(ScrollbarOpts{
				ContentSize: tt.contentSize,
				ViewSize:    tt.viewSize,
			})
			sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: tt.rectH})
			sb.SetState(tt.contentSize, tt.viewSize, 0)

			thumbH, _ := sb.thumbGeometry()
			if thumbH != tt.wantThumbH {
				t.Errorf("thumbH = %d, want %d", thumbH, tt.wantThumbH)
			}
		})
	}
}

func TestScrollbar_ThumbPosition(t *testing.T) {
	sb := NewScrollbar(ScrollbarOpts{})
	sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})

	// 100 content, 20 view → thumbH = 4, maxScroll = 80
	// At position 0: thumbY = 0
	sb.SetState(100, 20, 0)
	thumbH, thumbY := sb.thumbGeometry()
	if thumbH != 4 {
		t.Errorf("thumbH = %d, want 4", thumbH)
	}
	if thumbY != 0 {
		t.Errorf("at pos 0: thumbY = %d, want 0", thumbY)
	}

	// At max scroll (80): thumbY should be at bottom = 20 - 4 = 16
	sb.SetState(100, 20, 80)
	_, thumbY = sb.thumbGeometry()
	if thumbY != 16 {
		t.Errorf("at pos 80: thumbY = %d, want 16", thumbY)
	}

	// At midpoint (40): thumbY = 40 * 16 / 80 = 8
	sb.SetState(100, 20, 40)
	_, thumbY = sb.thumbGeometry()
	if thumbY != 8 {
		t.Errorf("at pos 40: thumbY = %d, want 8", thumbY)
	}
}

func TestScrollbar_ClickTrack(t *testing.T) {
	var scrolledTo int
	sb := NewScrollbar(ScrollbarOpts{
		ContentSize: 100,
		ViewSize:    20,
		OnScroll: func(pos int, ctx *tui.Ctx) {
			scrolledTo = pos
		},
	})
	sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	sb.SetState(100, 20, 0)

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {},
	}

	// Click at bottom of track (y=19) — should scroll near max
	handled := sb.Handle(event.MouseEvent{
		X:      0,
		Y:      19,
		Button: event.MouseButtonLeft,
		Action: event.MousePress,
	}, ctx)

	if !handled {
		t.Error("expected click on track to be handled")
	}
	// maxScroll=80, pos = 19*80/19 = 80
	if scrolledTo != 80 {
		t.Errorf("scrolledTo = %d, want 80", scrolledTo)
	}

	// Click at top of track (y=0) — should scroll to 0
	sb.Handle(event.MouseEvent{
		X:      0,
		Y:      0,
		Button: event.MouseButtonLeft,
		Action: event.MousePress,
	}, ctx)
	if scrolledTo != 0 {
		t.Errorf("scrolledTo = %d, want 0", scrolledTo)
	}
}

func TestScrollbar_DragThumb(t *testing.T) {
	var scrolledTo int
	sb := NewScrollbar(ScrollbarOpts{
		ContentSize: 100,
		ViewSize:    20,
		OnScroll: func(pos int, ctx *tui.Ctx) {
			scrolledTo = pos
		},
	})
	sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	sb.SetState(100, 20, 0)

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {},
	}

	// Thumb is at y=0..3 (height 4). Click on thumb at y=1 to start drag.
	sb.Handle(event.MouseEvent{
		X: 0, Y: 1,
		Button: event.MouseButtonLeft,
		Action: event.MousePress,
	}, ctx)

	if !sb.dragging {
		t.Fatal("expected dragging to be true after clicking thumb")
	}

	// Drag down by 8 pixels. Track space = 20-4 = 16.
	// newPos = 0 + 8*80/16 = 40
	sb.Handle(event.MouseEvent{
		X: 0, Y: 9,
		Button: event.MouseButtonLeft,
		Action: event.MouseDrag,
	}, ctx)

	if scrolledTo != 40 {
		t.Errorf("after drag: scrolledTo = %d, want 40", scrolledTo)
	}

	// Release
	sb.Handle(event.MouseEvent{
		X: 0, Y: 9,
		Button: event.MouseButtonLeft,
		Action: event.MouseRelease,
	}, ctx)

	if sb.dragging {
		t.Error("expected dragging to be false after release")
	}
}

func TestScrollbar_NoScrollWhenContentFits(t *testing.T) {
	sb := NewScrollbar(ScrollbarOpts{
		ContentSize: 10,
		ViewSize:    20,
	})
	sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})

	// Should not handle mouse events when content fits
	handled := sb.Handle(event.MouseEvent{
		X: 0, Y: 5,
		Button: event.MouseButtonLeft,
		Action: event.MousePress,
	}, nil)

	if handled {
		t.Error("expected click to not be handled when content fits viewport")
	}
}

func TestScrollbar_IgnoresNonMouse(t *testing.T) {
	sb := NewScrollbar(ScrollbarOpts{
		ContentSize: 100,
		ViewSize:    20,
	})
	sb.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})

	handled := sb.Handle(event.KeyEvent{Key: event.KeyUp}, nil)
	if handled {
		t.Error("expected key event to not be handled")
	}
}

func TestScrollView_ScrollbarAutoShown(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
		Scrollbar: ScrollbarAuto,
	})

	// Layout with viewport smaller than content — scrollbar should show
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	if !sv.showScrollbar {
		t.Error("expected scrollbar to be shown when content overflows")
	}
	// Child should get reduced width
	if child.rect.W != 19 {
		t.Errorf("expected child width 19, got %d", child.rect.W)
	}
}

func TestScrollView_ScrollbarAutoHidden(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 5}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
		Scrollbar: ScrollbarAuto,
	})

	// Layout with viewport larger than content — no scrollbar
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	if sv.showScrollbar {
		t.Error("expected scrollbar to be hidden when content fits")
	}
	if child.rect.W != 20 {
		t.Errorf("expected child width 20, got %d", child.rect.W)
	}
}

func TestScrollView_ScrollbarAlways(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 5}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
		Scrollbar: ScrollbarAlways,
	})

	// Even when content fits, scrollbar should reserve space
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	if !sv.showScrollbar {
		t.Error("expected scrollbar to always be shown")
	}
	if child.rect.W != 19 {
		t.Errorf("expected child width 19, got %d", child.rect.W)
	}

	// MinSize should add 1 for scrollbar
	ms := sv.MinSize()
	if ms.W != 11 {
		t.Errorf("expected MinSize.W 11, got %d", ms.W)
	}
}

func TestScrollView_ScrollbarHidden(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
		Scrollbar: ScrollbarHidden,
	})

	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	if sv.showScrollbar {
		t.Error("expected scrollbar to be hidden")
	}
	if child.rect.W != 20 {
		t.Errorf("expected child width 20, got %d", child.rect.W)
	}
}
