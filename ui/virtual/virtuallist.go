package virtual

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/ui"
)

// RenderRowFunc is the callback for rendering a single row in the virtual list.
// It receives the item index, selection state, focus state, painter, and row rect.
type RenderRowFunc func(
	i int,
	selected bool,
	focused bool,
	p *tui.Painter,
	r geom.Rect,
)

// VirtualListOpts holds options for creating a VirtualList.
type VirtualListOpts struct {
	ID         tui.ID
	RowHeight  int
	Count      func() int
	RenderRow  RenderRowFunc
	OnActivate func(i int, ctx *tui.Ctx)
}

// VirtualList provides virtualization for large datasets by rendering only visible items.
// Scrolling is item-based via scrollItem, not pixel-based.
type VirtualList struct {
	id            tui.ID
	rect          geom.Rect
	rowHeight     int
	count         func() int
	renderRow     RenderRowFunc
	onActivate    func(i int, ctx *tui.Ctx)
	scrollItem    int
	selectedIndex int // -1 = no selection
}

// Panics if Count, RenderRow are nil or RowHeight <= 0.
func NewVirtualList(opts VirtualListOpts) *VirtualList {
	if opts.Count == nil {
		panic("virtual.NewVirtualList: Count is required")
	}

	if opts.RenderRow == nil {
		panic("virtual.NewVirtualList: RenderRow is required")
	}

	if opts.RowHeight <= 0 {
		panic("virtual.NewVirtualList: RowHeight must be >= 1")
	}

	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	return &VirtualList{
		id:            id,
		rowHeight:     opts.RowHeight,
		count:         opts.Count,
		renderRow:     opts.RenderRow,
		onActivate:    opts.OnActivate,
		selectedIndex: -1,
	}
}

func (v *VirtualList) ID() tui.ID {
	return v.id
}

func (v *VirtualList) Rect() geom.Rect {
	return v.rect
}

func (v *VirtualList) Layout(r geom.Rect) {
	v.rect = r
	v.clampScroll()
	v.clampSelection()
}

func (v *VirtualList) MinSize() geom.Size {
	return geom.Size{
		W: 1,
		H: v.rowHeight,
	}
}

func (v *VirtualList) Focusable() bool {
	return true
}

// Paint renders the visible items only.
func (v *VirtualList) Paint(p *tui.Painter, ctx *tui.Ctx) {
	if v.rect.W <= 0 || v.rect.H <= 0 {
		return
	}

	n := v.safeCount()
	v.clampScroll()
	v.clampSelection()

	if n == 0 {
		v.paintEmpty(p, ctx)

		return
	}

	visible := v.visibleItems()
	if visible <= 0 {
		return
	}

	end := min(n, v.scrollItem+visible)
	focused := ctx != nil && ctx.FocusedID == v.id

	p.WithClip(v.rect, func(cp *tui.Painter) {
		for i := v.scrollItem; i < end; i++ {
			rowOffset := (i - v.scrollItem) * v.rowHeight
			rowY := v.rect.Y + rowOffset

			rowH := min(v.rowHeight, v.rect.Y+v.rect.H-rowY)
			if rowH <= 0 {
				break
			}

			rowRect := geom.Rect{
				X: v.rect.X,
				Y: rowY,
				W: v.rect.W,
				H: rowH,
			}

			cp.WithClip(rowRect, func(rp *tui.Painter) {
				v.renderRow(i, i == v.selectedIndex, focused, rp, rowRect)
			})
		}
	})
}

// PaintDrawer preserves compatibility with DrawerPaintable container paths on
// the current cell-family renderer by delegating to the underlying Painter when
// available. VirtualList remains a cell-shaped API because row rendering is
// still explicitly Painter-based.
func (v *VirtualList) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	if p := tui.PainterFromDrawer(d); p != nil {
		v.Paint(p, ctx)
	}
}

// Handle processes keyboard and mouse events.
// Accepts both KeyEvent and *KeyEvent for pipeline compatibility.
func (v *VirtualList) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Mouse click selects the item at the clicked row
	if me, ok := e.(tui.MouseEvent); ok {
		wheelItems := 3
		if me.WheelDelta > 0 {
			wheelItems *= me.WheelDelta
		}

		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress {
			if ctx != nil {
				if ctx.RequestFocus != nil {
					ctx.RequestFocus(v.id)
				}
			}

			rowOffset := me.Y - v.rect.Y

			idx := v.scrollItem + rowOffset/v.rowHeight
			if idx >= 0 && idx < v.safeCount() {
				v.SelectIndex(ctx, idx)
			}

			return true
		}
		// Mouse wheel scrolling
		switch me.Button {
		case tui.MouseButtonWheelUp:
			v.ScrollBy(ctx, -wheelItems)

			return true
		case tui.MouseButtonWheelDown:
			v.ScrollBy(ctx, wheelItems)

			return true
		}

		return false
	}

	// Handle both value and pointer key events
	switch ev := e.(type) {
	case tui.KeyEvent:
		return v.handleKey(ev, ctx)
	case *tui.KeyEvent:
		if ev == nil {
			return false
		}

		return v.handleKey(*ev, ctx)
	default:
		return false
	}
}

// HandleAction handles semantic actions for navigation and activation.
func (v *VirtualList) HandleAction(act int, ctx *tui.Ctx) bool {
	n := v.safeCount()
	if n == 0 {
		return false
	}

	visible := v.visibleItems()

	switch ui.Action(act) {
	case ui.ActionMoveUp:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex-1)
		}

		return true
	case ui.ActionMoveDown:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex+1)
		}

		return true
	case ui.ActionActivate:
		if v.selectedIndex >= 0 && v.selectedIndex < n && v.onActivate != nil {
			v.onActivate(v.selectedIndex, ctx)

			return true
		}

		return false
	case ui.ActionPageUp:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex-visible)
		}

		return true
	case ui.ActionPageDown:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex+visible)
		}

		return true
	case ui.ActionHome:
		v.SelectIndex(ctx, 0)

		return true
	case ui.ActionEnd:
		v.SelectIndex(ctx, n-1)

		return true
	}

	return false
}

// ScrollTo scrolls to the given item index.
func (v *VirtualList) ScrollTo(ctx *tui.Ctx, item int) {
	old := v.scrollItem
	v.scrollItem = item
	v.clampScroll()

	if v.scrollItem != old {
		v.invalidate(ctx)
	}
}

// ScrollBy scrolls by a relative delta.
func (v *VirtualList) ScrollBy(ctx *tui.Ctx, delta int) {
	v.ScrollTo(ctx, v.scrollItem+delta)
}

// ScrollTop scrolls to the top of the list.
func (v *VirtualList) ScrollTop(ctx *tui.Ctx) {
	v.ScrollTo(ctx, 0)
}

// ScrollBottom scrolls to the bottom of the list.
func (v *VirtualList) ScrollBottom(ctx *tui.Ctx) {
	n := v.safeCount()
	visible := v.visibleItems()
	v.ScrollTo(ctx, max(0, n-visible))
}

// ScrollItem returns the current scroll position (item index).
func (v *VirtualList) ScrollItem() int {
	return v.scrollItem
}

// SetScrollItem sets the scroll position to the given item index.
func (v *VirtualList) SetScrollItem(ctx *tui.Ctx, item int) {
	v.ScrollTo(ctx, item)
}

// SelectIndex sets the selected index and scrolls it into view.
func (v *VirtualList) SelectIndex(ctx *tui.Ctx, index int) {
	oldSel := v.selectedIndex
	oldScroll := v.scrollItem

	if v.safeCount() == 0 {
		v.selectedIndex = -1
	} else {
		v.selectedIndex = clamp(index, 0, v.safeCount()-1)
		v.scrollSelectionIntoView()
	}

	if v.selectedIndex != oldSel || v.scrollItem != oldScroll {
		v.invalidate(ctx)
	}
}

// SelectedIndex returns the current selected index, or -1 if no selection.
func (v *VirtualList) SelectedIndex() int {
	return v.selectedIndex
}

// ClearSelection clears the current selection.
func (v *VirtualList) ClearSelection(ctx *tui.Ctx) {
	if v.selectedIndex == -1 {
		return
	}

	v.selectedIndex = -1
	v.invalidate(ctx)
}

// NotifyCountChanged notifies the list that the item count has changed.
// It reclamps scroll/selection and conservatively invalidates the list rect.
// Models should call this whenever Count() may produce a different result.
func (v *VirtualList) NotifyCountChanged(ctx *tui.Ctx) {
	v.clampScroll()
	v.clampSelection()
	v.invalidate(ctx)
}

// handleKey processes key events for selection and activation.
func (v *VirtualList) handleKey(e tui.KeyEvent, ctx *tui.Ctx) bool {
	n := v.safeCount()
	if n == 0 {
		return false
	}

	switch e.Key {
	case tui.KeyUp:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex-1)
		}

		return true

	case tui.KeyDown:
		if v.selectedIndex == -1 {
			v.SelectIndex(ctx, 0)
		} else {
			v.SelectIndex(ctx, v.selectedIndex+1)
		}

		return true

	case tui.KeyEnter:
		if v.selectedIndex >= 0 && v.selectedIndex < n && v.onActivate != nil {
			v.onActivate(v.selectedIndex, ctx)

			return true
		}

		return false

	default:
		return false
	}
}

// safeCount returns the count, clamped to non-negative.
func (v *VirtualList) safeCount() int {
	n := v.count()
	if n < 0 {
		return 0
	}

	return n
}

// visibleItems returns the number of items that can fit in the current rect.
func (v *VirtualList) visibleItems() int {
	if v.rowHeight <= 0 || v.rect.H <= 0 {
		return 0
	}

	return ceilDiv(v.rect.H, v.rowHeight)
}

// clampScroll ensures scrollItem is within valid bounds.
func (v *VirtualList) clampScroll() {
	n := v.safeCount()
	visible := v.visibleItems()

	maxScroll := max(0, n-visible)
	v.scrollItem = clamp(v.scrollItem, 0, maxScroll)
}

// clampSelection ensures selectedIndex is within valid bounds.
func (v *VirtualList) clampSelection() {
	n := v.safeCount()
	if n == 0 {
		v.selectedIndex = -1

		return
	}

	if v.selectedIndex == -1 {
		return
	}

	v.selectedIndex = clamp(v.selectedIndex, 0, n-1)
}

// scrollSelectionIntoView adjusts scroll so the selected item is visible.
func (v *VirtualList) scrollSelectionIntoView() {
	if v.selectedIndex < 0 {
		return
	}

	visible := v.visibleItems()
	if visible <= 0 {
		return
	}

	if v.selectedIndex < v.scrollItem {
		v.scrollItem = v.selectedIndex

		return
	}

	lastVisible := v.scrollItem + visible - 1
	if v.selectedIndex > lastVisible {
		v.scrollItem = v.selectedIndex - visible + 1
	}
}

// paintEmpty paints the empty state (intentionally minimal for now).
func (v *VirtualList) paintEmpty(p *tui.Painter, ctx *tui.Ctx) {
	// Intentionally empty for now.
	// Paint Contract A already clears damaged regions.
}

// invalidate marks the list rect as needing repaint.
func (v *VirtualList) invalidate(ctx *tui.Ctx) {
	if ctx == nil || ctx.Invalidate == nil {
		return
	}

	if v.rect.W <= 0 || v.rect.H <= 0 {
		return
	}

	ctx.Invalidate(v.rect)
}

// clamp clamps v between lo and hi.
func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}

	return max(lo, min(hi, v))
}

// ceilDiv returns ceil(a / b).
func ceilDiv(a, b int) int {
	if b <= 0 {
		return 0
	}

	return (a + b - 1) / b
}
