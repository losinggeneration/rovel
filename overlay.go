package rovel

import (
	"slices"

	"github.com/losinggeneration/rovel/geom"
)

// Placement determines how an overlay is positioned relative to the screen.
type Placement interface {
	Resolve(root View, screenSize geom.Size) geom.Rect
}

// Overlay represents a view displayed above the main view tree.
type Overlay struct {
	id         ID
	root       View
	modal      bool
	place      Placement
	rect       geom.Rect
	savedFocus ID
	onDismiss  func()
}

// OverlayOpts configures a new overlay.
type OverlayOpts struct {
	// Root is the view tree for the overlay content.
	Root View

	// Modal traps focus within the overlay subtree.
	Modal bool

	// Place determines positioning. Caller must set this.
	Place Placement

	// OnDismiss is called when the overlay is dismissed.
	OnDismiss func()
}

func (o *Overlay) ID() ID            { return o.id }
func (o *Overlay) Root() View        { return o.root }
func (o *Overlay) Modal() bool       { return o.modal }
func (o *Overlay) Rect() geom.Rect   { return o.rect }
func (o *Overlay) SavedFocus() ID    { return o.savedFocus }
func (o *Overlay) OnDismiss() func() { return o.onDismiss }

// OverlayManager manages a stack of overlays.
type OverlayManager struct {
	stack []*Overlay
}

// PushOverlay adds an overlay to the top of the stack. Returns the overlay.
func (m *OverlayManager) PushOverlay(opts OverlayOpts, currentFocus ID) *Overlay {
	o := &Overlay{
		id:         NewID(),
		root:       opts.Root,
		modal:      opts.Modal,
		place:      opts.Place,
		onDismiss:  opts.OnDismiss,
		savedFocus: currentFocus,
	}
	m.stack = append(m.stack, o)

	return o
}

// PopOverlay removes the topmost overlay. Returns it, or nil.
func (m *OverlayManager) PopOverlay() *Overlay {
	if len(m.stack) == 0 {
		return nil
	}

	last := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]

	return last
}

// PopOverlayByID removes a specific overlay by ID. Returns it, or nil.
func (m *OverlayManager) PopOverlayByID(id ID) *Overlay {
	for i, o := range m.stack {
		if o.id == id {
			m.stack = append(m.stack[:i], m.stack[i+1:]...)

			return o
		}
	}

	return nil
}

// TopOverlay returns the topmost overlay, or nil.
func (m *OverlayManager) TopOverlay() *Overlay {
	if len(m.stack) == 0 {
		return nil
	}

	return m.stack[len(m.stack)-1]
}

// TopModal returns the topmost modal overlay, or nil.
func (m *OverlayManager) TopModal() *Overlay {
	for i := len(m.stack) - 1; i >= 0; i-- {
		if m.stack[i].modal {
			return m.stack[i]
		}
	}

	return nil
}

// RaiseOverlay moves the overlay with the given id to the top of the stack,
// preserving its identity. No onDismiss fires and no saved focus is restored —
// raising is purely a z-order change. A non-modal overlay is raised only above
// other non-modal overlays: it stays below the topmost modal overlay, so a
// modal keeps blocking clicks and keys regardless of z-order changes.
// Returns the raised overlay, or nil if the id is unknown.
func (m *OverlayManager) RaiseOverlay(id ID) *Overlay {
	i := slices.IndexFunc(m.stack, func(o *Overlay) bool { return o.id == id })
	if i < 0 {
		return nil
	}

	o := m.stack[i]

	// Top of the allowed region: the raised overlay stops below the *lowest*
	// modal above it, so every modal it was under keeps blocking it.
	insertAt := len(m.stack)
	if !o.modal {
		for j := i + 1; j < len(m.stack); j++ {
			if m.stack[j].modal {
				insertAt = j
				break
			}
		}
	}

	m.stack = append(m.stack[:i], m.stack[i+1:]...)
	if insertAt > i {
		insertAt--
	}
	m.stack = slices.Insert(m.stack, insertAt, o)

	return o
}

// OverlayCount returns the number of overlays.
func (m *OverlayManager) OverlayCount() int { return len(m.stack) }

// HasOverlays returns true if there are any overlays.
func (m *OverlayManager) HasOverlays() bool { return len(m.stack) > 0 }

// OverlayStack returns a copy of the overlay stack (bottom to top).
func (m *OverlayManager) OverlayStack() []*Overlay {
	out := make([]*Overlay, len(m.stack))
	copy(out, m.stack)

	return out
}

// layoutOverlays resolves placement and runs Layout on every overlay root.
func (m *OverlayManager) layoutOverlays(screenSize geom.Size) {
	for _, o := range m.stack {
		o.rect = o.place.Resolve(o.root, screenSize)
		o.root.Layout(o.rect)
	}
}

// paintOverlays paints all overlays in z-order (bottom to top).
func (m *OverlayManager) paintOverlays(d Drawer, ctx *Ctx) {
	for _, o := range m.stack {
		d.WithClip(o.rect, func(cd Drawer) {
			cd.FillRect(o.rect, ctx.Theme.Base)
			o.root.Paint(cd, ctx)
		})
	}
}

// overlayHitTest checks overlays top-to-bottom for a hit. Returns the hit
// view and whether a modal overlay blocked further propagation.
func (m *OverlayManager) overlayHitTest(x, y int) (target View, blocked bool) {
	for i := len(m.stack) - 1; i >= 0; i-- {
		o := m.stack[i]

		r := o.rect
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			if hit := hitTestView(o.root, x, y); hit != nil {
				return hit, true
			}
		}

		if o.modal {
			return nil, true // modal blocks even if miss
		}
	}

	return nil, false
}

func hitTestView(v View, x, y int) View {
	r := v.Rect()
	if x < r.X || x >= r.X+r.W || y < r.Y || y >= r.Y+r.H {
		return nil
	}
	// Mirror the mouseOpaque check in App.hitTest (app.go).
	if _, ok := v.(mouseOpaque); ok {
		return v
	}

	if c, ok := v.(viewChildren); ok {
		children := c.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if hit := hitTestView(children[i], x, y); hit != nil {
				return hit
			}
		}
	}

	return v
}
