package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// TabsOpts holds options for creating Tabs.
type TabsOpts struct {
	ID    tui.ID
	Tabs  []Tab
	OnTab func(index int, ctx *tui.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleBar      *style.Style // tab bar background / unselected tabs
	StyleSelected *style.Style // selected tab (unfocused)
	StyleFocused  *style.Style // selected tab (focused)
}

// Tab represents a single tab with a title and content view.
type Tab struct {
	Title   string
	Content tui.View
}

// Tabs is a tab bar with switchable content panels.
// The tab bar occupies one line at the top; the content fills the rest.
type Tabs struct {
	id       tui.ID
	rect     tui.Rect
	tabs     []Tab
	selected int
	onTab    func(index int, ctx *tui.Ctx)

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
		id = tui.NewID()
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

func (t *Tabs) ID() tui.ID      { return t.id }
func (t *Tabs) Rect() tui.Rect  { return t.rect }
func (t *Tabs) Focusable() bool { return len(t.tabs) > 0 }

func (t *Tabs) Selected() int { return t.selected }

func (t *Tabs) SetSelected(ctx *tui.Ctx, idx int) {
	if idx < 0 || idx >= len(t.tabs) || idx == t.selected {
		return
	}

	t.selected = idx
	t.layoutContent()

	if ctx != nil {
		ctx.Invalidate(t.rect)
	}
}

func (t *Tabs) Layout(r tui.Rect) {
	t.rect = r
	t.layoutContent()
}

func (t *Tabs) layoutContent() {
	if len(t.tabs) == 0 {
		return
	}
	// Content area is everything below the tab bar (1 line)
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

func (t *Tabs) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r := t.rect
	if r.W <= 0 || r.H <= 0 || len(t.tabs) == 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == t.id
	t.paintTabBar(p, ctx, focused)

	// Paint content
	cr := t.contentRect()
	if cr.H > 0 && t.selected >= 0 && t.selected < len(t.tabs) {
		if content := t.tabs[t.selected].Content; content != nil {
			p.WithClip(cr, func(cp *tui.Painter) {
				content.Paint(cp, ctx)
			})
		}
	}
}

func (t *Tabs) paintTabBar(p *tui.Painter, ctx *tui.Ctx, focused bool) {
	r := t.rect

	barFallback := ctx.Theme.Palette.Surface
	if barFallback == (style.Style{}) {
		barFallback = ctx.Theme.Base
	}

	barSt := resolveStyle(t.stBar, barFallback)

	// Clear bar
	p.Fill(geom.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, ' ', barSt)

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
		p.Text(x, r.Y, lbl, st)
		x += w
	}
}

func (t *Tabs) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress && len(t.tabs) > 0 {
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

	ke, ok := e.(tui.KeyEvent)
	if !ok || len(t.tabs) == 0 {
		return false
	}

	switch ke.Key {
	case tui.KeyLeft:
		if t.selected > 0 {
			t.switchTab(ctx, t.selected-1)
		}

		return true
	case tui.KeyRight:
		if t.selected < len(t.tabs)-1 {
			t.switchTab(ctx, t.selected+1)
		}

		return true
	}

	return false
}

// HandleAction handles semantic actions.
// Left/Right and Activate only apply when the tab bar itself has focus.
func (t *Tabs) HandleAction(act int, ctx *tui.Ctx) bool {
	if len(t.tabs) == 0 {
		return false
	}

	if ctx == nil || ctx.FocusedID != t.id {
		return false
	}

	switch ui.Action(act) {
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
	}

	return false
}

// Children returns the current tab's content view for focus traversal.
func (t *Tabs) Children() []tui.View {
	if t.selected >= 0 && t.selected < len(t.tabs) {
		if c := t.tabs[t.selected].Content; c != nil {
			return []tui.View{c}
		}
	}

	return nil
}

func (t *Tabs) switchTab(ctx *tui.Ctx, idx int) {
	if idx < 0 || idx >= len(t.tabs) || idx == t.selected {
		return
	}

	t.selected = idx
	t.layoutContent()

	if ctx != nil {
		ctx.InvalidateLayout(t.id)
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
func (t *Tabs) focusContent(ctx *tui.Ctx) {
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

	type composite interface{ Children() []tui.View }

	var walk func(v tui.View) bool

	walk = func(v tui.View) bool {
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
