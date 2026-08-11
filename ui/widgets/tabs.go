package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
)

// TabsOpts holds options for creating Tabs.
type TabsOpts struct {
	ID    rovel.ID
	Tabs  []Tab
	OnTab func(index int, ctx *rovel.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleBar      *style.Style // tab bar background / unselected tabs
	StyleSelected *style.Style // selected tab (unfocused)
	StyleFocused  *style.Style // selected tab (focused)
}

// Tab represents a single tab with a title and content view.
type Tab struct {
	Title   string
	Content rovel.View
}

// Tabs is a tab bar with switchable content panels.
// The tab bar occupies one line at the top; the content fills the rest.
type Tabs struct {
	id       rovel.ID
	rect     rovel.Rect
	tabs     []Tab
	selected int
	onTab    func(index int, ctx *rovel.Ctx)

	stBar      *style.Style
	stSelected *style.Style
	stFocused  *style.Style
}

func NewTabs(tabs []Tab) *Tabs {
	return NewTabsOpts(TabsOpts{Tabs: tabs})
}

func NewTabsOpts(opts TabsOpts) *Tabs {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	return &Tabs{
		id:         id,
		tabs:       opts.Tabs,
		onTab:      opts.OnTab,
		stBar:      opts.StyleBar,
		stSelected: opts.StyleSelected,
		stFocused:  opts.StyleFocused,
	}
}

func (t *Tabs) ID() rovel.ID     { return t.id }
func (t *Tabs) Rect() rovel.Rect { return t.rect }
func (t *Tabs) Focusable() bool  { return len(t.tabs) > 0 }

func (t *Tabs) Selected() int { return t.selected }

func (t *Tabs) SetSelected(ctx *rovel.Ctx, idx int) {
	if idx < 0 || idx >= len(t.tabs) || idx == t.selected {
		return
	}

	t.selected = idx
	t.layoutContent()

	if ctx != nil {
		ctx.Invalidate(t.rect)
	}
}

func (t *Tabs) Layout(r rovel.Rect) {
	t.rect = r
	t.layoutContent()
}

func (t *Tabs) MinSize() geom.Size {
	barW := 0
	for _, tab := range t.tabs {
		barW += 2 + text.Width(tab.Title) + 1 // " title "
	}

	minH := 1 // at least the tab bar
	// Add content min height
	for _, tab := range t.tabs {
		if tab.Content != nil {
			ms := tab.Content.MinSize()
			if ms.H+1 > minH {
				minH = ms.H + 1
			}

			if ms.W > barW {
				barW = ms.W
			}
		}
	}

	return geom.Size{W: barW, H: minH}
}

func (t *Tabs) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := t.rect
	if r.W <= 0 || r.H <= 0 || len(t.tabs) == 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == t.id
	t.paintTabBar(d, ctx, focused)

	// Paint content
	cr := t.contentRect()
	if cr.H > 0 && t.selected >= 0 && t.selected < len(t.tabs) {
		if content := t.tabs[t.selected].Content; content != nil {
			d.WithClip(cr, func(cd rovel.Drawer) {
				content.Paint(cd, ctx)
			})
		}
	}
}

func (t *Tabs) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress && len(t.tabs) > 0 {
			// Click on the tab bar row?
			if me.Y == t.rect.Y {
				if ctx != nil && ctx.RequestFocus != nil {
					ctx.RequestFocus(t.id)
				}
				// Walk tab positions to find which was clicked
				x := t.rect.X

				for i, tab := range t.tabs {
					w := 2 + text.Width(tab.Title) // " title "
					if me.X >= x && me.X < x+w {
						t.switchTab(ctx, i)

						return true
					}

					x += w
				}
			}
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok || len(t.tabs) == 0 {
		return false
	}

	switch ke.Key {
	case rovel.KeyLeft:
		if t.selected > 0 {
			t.switchTab(ctx, t.selected-1)
		}

		return true
	case rovel.KeyRight:
		if t.selected < len(t.tabs)-1 {
			t.switchTab(ctx, t.selected+1)
		}

		return true
	default:
	}

	return false
}

// HandleAction handles semantic actions.
// Left/Right and Activate only apply when the tab bar itself has focus.
func (t *Tabs) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if len(t.tabs) == 0 {
		return false
	}

	if ctx == nil || ctx.FocusedID != t.id {
		return false
	}

	switch act {
	case ui.ActionMoveLeft:
		if t.selected > 0 {
			t.switchTab(ctx, t.selected-1)
		}

		return true
	case ui.ActionMoveRight:
		if t.selected < len(t.tabs)-1 {
			t.switchTab(ctx, t.selected+1)
		}

		return true
	case ui.ActionActivate:
		t.focusContent(ctx)

		return true
	default:
	}

	return false
}

// Children returns the current tab's content view for focus traversal.
func (t *Tabs) Children() []rovel.View {
	if t.selected >= 0 && t.selected < len(t.tabs) {
		if c := t.tabs[t.selected].Content; c != nil {
			return []rovel.View{c}
		}
	}

	return nil
}

func (t *Tabs) layoutContent() {
	if len(t.tabs) == 0 {
		return
	}

	cr := t.contentRect()
	if cr.H > 0 && t.selected >= 0 && t.selected < len(t.tabs) {
		if content := t.tabs[t.selected].Content; content != nil {
			content.Layout(cr)
		}
	}
}

func (t *Tabs) contentRect() geom.Rect {
	r := t.rect
	if r.H <= 1 {
		return geom.Rect{}
	}

	return geom.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}
}

func (t *Tabs) paintTabBar(d rovel.Drawer, ctx *rovel.Ctx, focused bool) {
	r := t.rect

	barFallback := ctx.Theme.Palette.Surface
	if barFallback == (style.Style{}) {
		barFallback = ctx.Theme.Base
	}

	barSt := resolveStyle(t.stBar, barFallback)

	d.FillRect(geom.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, barSt)

	x := r.X
	for i, tab := range t.tabs {
		if x >= r.X+r.W {
			break
		}

		label := " " + tab.Title + " "
		w := text.Width(label)

		var st style.Style

		if i == t.selected {
			if focused {
				st = resolveStyle(t.stFocused, ctx.Theme.Palette.Focus)
			} else {
				st = resolveStyle(t.stSelected, ctx.Theme.Palette.Accent)
			}
		} else {
			st = barSt
		}

		lbl := text.Truncate(label, r.X+r.W-x, false)
		d.DrawText(rovel.Point{X: x, Y: r.Y}, lbl, st)
		x += w
	}
}

func (t *Tabs) switchTab(ctx *rovel.Ctx, idx int) {
	if idx < 0 || idx >= len(t.tabs) || idx == t.selected {
		return
	}

	t.selected = idx
	t.layoutContent()

	if ctx != nil {
		ctx.InvalidateLayout()
		ctx.Invalidate(t.rect)
		// Focus the new tab's content if it's focusable
		t.focusContent(ctx)
	}

	if t.onTab != nil {
		t.onTab(idx, ctx)
	}
}

// focusContent focuses the tab's content view, or the first focusable
// descendant within it.
func (t *Tabs) focusContent(ctx *rovel.Ctx) {
	if ctx == nil || ctx.RequestFocus == nil {
		return
	}

	if t.selected < 0 || t.selected >= len(t.tabs) {
		return
	}

	content := t.tabs[t.selected].Content
	if content == nil {
		return
	}

	type focusable interface{ Focusable() bool }

	type composite interface{ Children() []rovel.View }

	var walk func(v rovel.View) bool

	walk = func(v rovel.View) bool {
		if f, ok := v.(focusable); ok && f.Focusable() {
			ctx.RequestFocus(v.ID())

			return true
		}

		if c, ok := v.(composite); ok {
			for _, child := range c.Children() {
				if walk(child) {
					return true
				}
			}
		}

		return false
	}
	walk(content)
}
