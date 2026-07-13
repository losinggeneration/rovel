package widgets

import (
	"testing"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
)

// dropdownCtx builds a Ctx wired to a real OverlayManager so Select's
// dropdown lifecycle can be exercised without a full App.
func dropdownCtx(om *rovel.OverlayManager, dismissed *int) *rovel.Ctx {
	return &rovel.Ctx{
		Invalidate: func(geom.Rect) {},
		ShowOverlay: func(opts rovel.OverlayOpts) *rovel.Overlay {
			return om.PushOverlay(opts, 0)
		},
		DismissOverlay: func() *rovel.Overlay {
			*dismissed++

			return om.PopOverlay()
		},
	}
}

func TestSelect_DropdownDismissesOnEscape(t *testing.T) {
	s := NewSelect([]string{"a", "b"})

	var (
		om        rovel.OverlayManager
		dismissed int
	)

	ctx := dropdownCtx(&om, &dismissed)

	if !s.Handle(event.KeyEvent{Key: event.KeyEnter}, ctx) {
		t.Fatal("Enter did not open dropdown")
	}

	top := om.TopOverlay()
	if top == nil {
		t.Fatal("no overlay after opening dropdown")
	}

	// The dropdown is a modal overlay: outside clicks and main-tree keys are
	// blocked, and app-level Escape handling only exists when ResolveAction
	// is configured. The list itself must therefore handle Escape, or the
	// dropdown becomes a keyboard trap.
	if !top.Root().Handle(event.KeyEvent{Key: event.KeyEsc}, ctx) {
		t.Fatal("dropdown list did not handle Escape")
	}

	if dismissed != 1 || om.HasOverlays() {
		t.Fatalf("dropdown not dismissed on Escape: dismissed=%d overlays=%d", dismissed, om.OverlayCount())
	}

	if s.Selected() != -1 {
		t.Fatalf("Escape must not change the selection, got %d", s.Selected())
	}
}
