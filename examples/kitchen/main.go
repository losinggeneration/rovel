// Package main is a kitchen-sink demo showing every widget in the toolkit.
//
// Widgets demonstrated:
//   - Button (normal, solid chrome, disabled)
//   - TextInput (single-line editing with paste support)
//   - TextArea (multi-line editing with selection)
//   - Checkbox (toggle with label)
//   - RadioGroup (mutually exclusive selection)
//   - Select / Dropdown (overlay-based)
//   - ProgressBar (display-only indicator)
//   - Tabs (switchable content panels)
//   - ScrollView (scrollable container with mouse wheel + scrollbar)
//   - VirtualList (efficient large-dataset rendering)
//   - Canvas (custom paint/handle escape hatch)
//   - Dialog (modal overlay)
//   - Label (static text)
//   - Clickable (mouse click wrapper)
//   - Border, HStack, VStack, Split, Padding (layout containers)
package main

import (
	"fmt"
	"math"
	"os"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/overlay"
	"github.com/losinggeneration/tui/ui/virtual"
	"github.com/losinggeneration/tui/ui/widgets"
)

// Custom actions for this app.
const (
	ActionQuit ui.Action = iota + 100
)

type appKeymap struct{}

func (appKeymap) Resolve(ctx ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	// In text input mode, let Ctrl+C be handled as copy by the default keymap.
	if ctx == ui.KeyCtxTextInput && k.Key == event.KeyCtrlC {
		return ui.ActionNone, false
	}

	switch k.Key {
	case event.KeyCtrlC:
		return ActionQuit, true
	case event.KeyEsc:
		return ActionQuit, true
	}

	return ui.ActionNone, false
}

// state holds all mutable UI state.
type state struct {
	app    *tui.App
	status *widgets.Label

	textInput *widgets.TextInput
	textArea  *widgets.TextArea
	checkbox1 *widgets.Checkbox
	checkbox2 *widgets.Checkbox
	radio     *widgets.RadioGroup
	progress  *widgets.ProgressBar
	selectW   *widgets.Select
	tabs      *widgets.Tabs
	vlist     *virtual.VirtualList

	// VirtualList data
	listItems []string
}

func main() {
	s := &state{}

	// Populate virtual list data
	s.listItems = make([]string, 1000)
	for i := range s.listItems {
		s.listItems[i] = fmt.Sprintf("Item %d — virtual list row", i+1)
	}

	app, err := tui.New(tui.AppOpts{
		Theme: tui.DefaultThemeModern(),
		Input: tui.InputOpts{
			Mouse:          true,
			BracketedPaste: true,
		},
		ResolveAction: ui.NewResolver(ui.CompositeKeymap{
			Keymaps: []ui.Keymap{
				appKeymap{},
				ui.DefaultKeymap{},
			},
		}),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	s.app = app

	root := s.buildUI()
	app.SetRoot(root)

	if err := app.Enable(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	defer func() { _ = app.Restore() }()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func (s *state) buildUI() tui.View {
	s.status = widgets.NewLabel("Kitchen Sink — Tab to navigate, Esc/Ctrl-C to quit")

	left := s.buildLeftColumn()
	right := s.buildRightColumn()

	split := layout.NewSplit(layout.Horizontal)
	split.SetFirst(left)
	split.SetSecond(right)
	split.SetRatio(0.45)

	mainVStack := layout.NewVStack()
	mainVStack.AddChild(layout.NewChild(s.status))
	mainVStack.AddChild(layout.GrowChild(split, 1, 1))

	return &rootView{
		id:    tui.NewID(),
		app:   s.app,
		View:  mainVStack,
		state: s,
	}
}

// ── left column ──────────────────────────────────────────────────────

func (s *state) buildLeftColumn() tui.View {
	col := layout.NewVStack()
	col.SetGap(1)

	ringCol := layout.NewVStack()
	ringCol.AddChild(layout.NewChild(s.buildButtonSection()))
	ringCol.AddChild(layout.NewChild(s.buildTextInputSection()))

	ring := widgets.NewFocusRing(ringCol)
	col.AddChild(layout.NewChild(ring))
	col.AddChild(layout.NewChild(s.buildCheckboxSection()))
	col.AddChild(layout.NewChild(s.buildRadioSection()))
	col.AddChild(layout.NewChild(s.buildOverlaySection()))
	col.AddChild(layout.NewChild(s.buildCanvasSection()))

	return col
}

func (s *state) buildButtonSection() tui.View {
	btn1 := widgets.NewButton("Normal")
	btn1.SetOnPress(func(ctx *tui.Ctx) {
		s.setStatus(ctx, "Normal button pressed")
	})

	btn2 := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:  "Solid",
		Chrome: widgets.ButtonChromeSolid,
	})
	btn2.SetOnPress(func(ctx *tui.Ctx) {
		s.setStatus(ctx, "Solid button pressed")
	})

	btn3 := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:    "Disabled",
		Disabled: true,
	})

	row := layout.NewHStackWithGap([]layout.Child{
		layout.NewChild(btn1),
		layout.NewChild(btn2),
		layout.NewChild(btn3),
	}, 1)

	b := layout.NewBorder(row)
	b.SetTitle("Buttons")

	return b
}

func (s *state) buildTextInputSection() tui.View {
	s.textInput = widgets.NewTextInput()
	s.textInput.SetText(nil, "editable text — try typing or pasting")

	b := layout.NewBorder(s.textInput)
	b.SetTitle("Text Input (paste-aware)")

	return b
}

func (s *state) buildCheckboxSection() tui.View {
	s.checkbox1 = widgets.NewCheckboxOpts(widgets.CheckboxOpts{
		Label:   "Enable feature A",
		Checked: true,
		OnChange: func(checked bool, ctx *tui.Ctx) {
			if checked {
				s.setStatus(ctx, "Feature A enabled")
			} else {
				s.setStatus(ctx, "Feature A disabled")
			}
		},
	})

	s.checkbox2 = widgets.NewCheckboxOpts(widgets.CheckboxOpts{
		Label: "Enable feature B",
		OnChange: func(checked bool, ctx *tui.Ctx) {
			if checked {
				s.setStatus(ctx, "Feature B enabled")
			} else {
				s.setStatus(ctx, "Feature B disabled")
			}
		},
	})

	col := layout.NewVStack()
	col.Add(s.checkbox1)
	col.Add(s.checkbox2)

	b := layout.NewBorder(col)
	b.SetTitle("Checkboxes")

	return b
}

func (s *state) buildRadioSection() tui.View {
	s.radio = widgets.NewRadioGroupOpts(widgets.RadioGroupOpts{
		Items:    []string{"Option Alpha", "Option Beta", "Option Gamma"},
		Selected: 0,
		OnChange: func(idx int, ctx *tui.Ctx) {
			names := []string{"Alpha", "Beta", "Gamma"}
			s.setStatus(ctx, fmt.Sprintf("Radio: %s", names[idx]))
		},
	})

	b := layout.NewBorder(s.radio)
	b.SetTitle("Radio Group")

	return b
}

func (s *state) buildOverlaySection() tui.View {
	btnDialog := widgets.NewButton("Open Dialog")
	btnDialog.SetOnPress(func(ctx *tui.Ctx) { s.openDialog(ctx) })

	btnClipRead := widgets.NewButton("Read Clipboard")
	btnClipRead.SetOnPress(func(ctx *tui.Ctx) {
		if ctx.ClipboardRead != nil {
			ctx.ClipboardRead()
			s.setStatus(ctx, "Clipboard read requested...")
		} else {
			s.setStatus(ctx, "Clipboard read not available")
		}
	})

	row := layout.NewHStackWithGap([]layout.Child{
		layout.NewChild(btnDialog),
		layout.NewChild(btnClipRead),
	}, 1)

	b := layout.NewBorder(row)
	b.SetTitle("Overlays / Clipboard")

	return b
}

func (s *state) buildCanvasSection() tui.View {
	// Canvas: custom paint escape hatch — draws a simple bar chart.
	canvas := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 20, H: 3},
		Paint: func(p *tui.Painter, r geom.Rect, ctx *tui.Ctx) {
			if r.W <= 0 || r.H <= 0 {
				return
			}

			accentSt := ctx.Theme.Palette.Accent
			if accentSt == (style.Style{}) {
				accentSt = ctx.Theme.Base
			}
			// Draw a sine-wave bar chart
			for x := range r.W {
				v := (math.Sin(float64(x)*0.5) + 1) / 2 // 0..1

				barH := int(v * float64(r.H))
				if barH < 1 {
					barH = 1
				}

				for y := r.H - barH; y < r.H; y++ {
					p.SetCell(r.X+x, r.Y+y, '▮', accentSt)
				}
			}
		},
	})

	b := layout.NewBorder(canvas)
	b.SetTitle("Canvas (custom paint)")

	return b
}

// ── right column ─────────────────────────────────────────────────────

func (s *state) buildRightColumn() tui.View {
	col := layout.NewVStack()
	col.SetGap(1)

	col.AddChild(layout.NewChild(s.buildSelectSection()))
	col.AddChild(layout.NewChild(s.buildProgressSection()))
	col.AddChild(layout.GrowChild(s.buildTabsSection(), 1, 1))

	return col
}

func (s *state) buildSelectSection() tui.View {
	s.selectW = widgets.NewSelectOpts(widgets.SelectOpts{
		Items:       []string{"Red", "Green", "Blue", "Yellow", "Magenta", "Cyan"},
		Placeholder: "Pick a color...",
		OnChange: func(idx int, ctx *tui.Ctx) {
			colors := []string{"Red", "Green", "Blue", "Yellow", "Magenta", "Cyan"}
			s.setStatus(ctx, fmt.Sprintf("Color: %s", colors[idx]))
		},
	})

	b := layout.NewBorder(s.selectW)
	b.SetTitle("Select / Dropdown (overlay)")

	return b
}

func (s *state) buildProgressSection() tui.View {
	s.progress = widgets.NewProgressBarOpts(widgets.ProgressBarOpts{
		Value: 0.35,
	})

	btnLess := widgets.NewButton("-10%")
	btnLess.SetOnPress(func(ctx *tui.Ctx) {
		v := s.progress.Value() - 0.1
		if v < 0 {
			v = 0
		}

		s.progress.SetValue(ctx, v)
		s.setStatus(ctx, fmt.Sprintf("Progress: %.0f%%", s.progress.Value()*100))
	})

	btnMore := widgets.NewButton("+10%")
	btnMore.SetOnPress(func(ctx *tui.Ctx) {
		v := s.progress.Value() + 0.1
		if v > 1 {
			v = 1
		}

		s.progress.SetValue(ctx, v)
		s.setStatus(ctx, fmt.Sprintf("Progress: %.0f%%", s.progress.Value()*100))
	})

	barRow := layout.NewHStackWithChildren([]layout.Child{
		layout.GrowXChild(s.progress, 1),
	})

	btnRow := layout.NewHStackWithGap([]layout.Child{
		layout.NewChild(btnLess),
		layout.NewChild(btnMore),
	}, 1)

	col := layout.NewVStack()
	col.AddChild(layout.NewChild(barRow))
	col.AddChild(layout.NewChild(btnRow))

	b := layout.NewBorder(col)
	b.SetTitle("Progress Bar")

	return b
}

func (s *state) buildTabsSection() tui.View {
	tab1 := s.buildInfoTab()
	tab2 := s.buildScrollTab()
	tab3 := s.buildVirtualListTab()
	tab4 := s.buildTextAreaTab()

	s.tabs = widgets.NewTabsOpts(widgets.TabsOpts{
		Tabs: []widgets.Tab{
			{Title: "Info", Content: tab1},
			{Title: "ScrollView", Content: tab2},
			{Title: "VirtualList", Content: tab3},
			{Title: "TextArea", Content: tab4},
		},
		OnTab: func(idx int, ctx *tui.Ctx) {
			titles := []string{"Info", "ScrollView", "VirtualList", "TextArea"}
			s.setStatus(ctx, fmt.Sprintf("Tab: %s", titles[idx]))
		},
	})

	b := layout.NewBorder(s.tabs)
	b.SetTitle("Tabs (Left/Right to switch)")

	return b
}

func (s *state) buildInfoTab() tui.View {
	return widgets.NewLabel(
		"Kitchen Sink — all widgets in one place\n" +
			"\n" +
			"Keyboard:\n" +
			"  Tab / Shift-Tab   move focus\n" +
			"  Enter / Space     activate\n" +
			"  Arrow keys        move within widget\n" +
			"  Esc / Ctrl-C      quit\n" +
			"\n" +
			"Mouse:\n" +
			"  Click any widget to focus + activate\n" +
			"  Click radio/list items to select\n" +
			"  Click tab bar to switch tabs\n" +
			"  Click text input to position cursor\n" +
			"  Scroll wheel on ScrollView / VirtualList\n" +
			"  Click/drag scrollbar to scroll\n" +
			"\n" +
			"Widgets: Button, TextInput, Checkbox,\n" +
			"RadioGroup, Select, ProgressBar, Tabs,\n" +
			"ScrollView, Scrollbar, VirtualList,\n" +
			"Canvas, Dialog, Label, FocusRing, Border,\n" +
			"HStack, VStack, Split, Padding")
}

func (s *state) buildScrollTab() tui.View {
	// 200 lines to exercise scrolling + mouse wheel
	var lines string
	for i := 1; i <= 200; i++ {
		lines += fmt.Sprintf("Line %3d  — scroll with arrow keys, PgUp/PgDn, or mouse wheel\n", i)
	}

	label := widgets.NewLabel(lines)

	sv := widgets.NewScrollView(widgets.ScrollViewOpts{
		Child:     label,
		Focusable: true,
		Scrollbar: widgets.ScrollbarAuto,
	})

	return sv
}

func (s *state) buildVirtualListTab() tui.View {
	s.vlist = virtual.NewVirtualList(virtual.VirtualListOpts{
		RowHeight: 1,
		Count:     func() int { return len(s.listItems) },
		RenderRow: func(i int, selected, focused bool, p *tui.Painter, r geom.Rect) {
			var st style.Style
			if selected && focused {
				st = style.Style{FG: style.ColorDefault, BG: style.ColorDefault, Attr: style.AttrReverse}
			} else if selected {
				st = style.Style{Attr: style.AttrUnderline}
			} else {
				st = style.Style{FG: style.ColorDefault, BG: style.ColorDefault}
			}

			p.Fill(r, ' ', st)
			label := text.Truncate(s.listItems[i], r.W, false)
			p.Text(r.X, r.Y, label, st)
		},
		OnActivate: func(i int, ctx *tui.Ctx) {
			s.setStatus(ctx, fmt.Sprintf("VirtualList activated: %s", s.listItems[i]))
		},
	})

	return newScrollPanel(s.vlist, widgets.NewScrollbar(widgets.ScrollbarOpts{
		OnScroll: func(pos int, ctx *tui.Ctx) {
			s.vlist.ScrollTo(ctx, pos)
		},
	}), func() (contentSize, viewSize, position int) {
		return len(s.listItems), s.vlist.Rect().H, s.vlist.ScrollItem()
	})
}

func (s *state) buildTextAreaTab() tui.View {
	s.textArea = widgets.NewTextArea()
	s.textArea.SetText(nil, "Multi-line text editor\n"+
		"\n"+
		"Try editing this text:\n"+
		"  - Arrow keys to move cursor\n"+
		"  - Up/Down navigates lines\n"+
		"  - Home/End for line start/end\n"+
		"  - Enter to insert newline\n"+
		"  - Backspace/Delete to remove\n"+
		"  - Shift+Arrow to select\n"+
		"  - Ctrl+A to select all\n"+
		"  - Ctrl+C to copy, Ctrl+X to cut\n"+
		"  - Mouse wheel to scroll\n"+
		"\n"+
		"The cursor column is \"sticky\" — moving\n"+
		"through short lines preserves your\n"+
		"original column position.")

	return newScrollPanel(s.textArea, widgets.NewScrollbar(widgets.ScrollbarOpts{
		OnScroll: func(pos int, ctx *tui.Ctx) {
			s.textArea.SetScrollY(ctx, pos)
		},
	}), func() (contentSize, viewSize, position int) {
		return s.textArea.LineCount(), s.textArea.Rect().H, s.textArea.ScrollY()
	})
}

// ── dialog overlay ───────────────────────────────────────────────────

func (s *state) openDialog(ctx *tui.Ctx) {
	if ctx == nil || ctx.ShowOverlay == nil {
		return
	}

	dlg := widgets.NewDialog(widgets.DialogOpts{
		Title:   "Confirm Action",
		Message: "This is a modal dialog.\nPress OK or Cancel.\nEsc also dismisses.",
		Buttons: []widgets.DialogButton{
			{
				Label: "OK",
				OnPress: func(ctx *tui.Ctx) {
					s.setStatus(ctx, "Dialog: OK")

					if ctx.DismissOverlay != nil {
						ctx.DismissOverlay()
					}
				},
			},
			{
				Label: "Cancel",
				OnPress: func(ctx *tui.Ctx) {
					s.setStatus(ctx, "Dialog: Cancelled")

					if ctx.DismissOverlay != nil {
						ctx.DismissOverlay()
					}
				},
			},
		},
	})

	ctx.ShowOverlay(tui.OverlayOpts{
		Root:  dlg,
		Modal: true,
		Place: overlay.Centered{},
	})
}

// ── helpers ──────────────────────────────────────────────────────────

func (s *state) setStatus(ctx *tui.Ctx, msg string) {
	s.status.SetText(ctx, msg)
}

// rootView wraps the view tree with global action handling.
type rootView struct {
	tui.View

	id    tui.ID
	app   *tui.App
	rect  geom.Rect
	state *state
}

func (r *rootView) ID() tui.ID           { return r.id }
func (r *rootView) Focusable() bool      { return false }
func (r *rootView) Children() []tui.View { return []tui.View{r.View} }

func (r *rootView) Layout(rect geom.Rect) {
	r.rect = rect
	// Status bar at top (1 row)
	r.View.Layout(geom.Rect{X: rect.X, Y: rect.Y + 1, W: rect.W, H: rect.H - 1})
}

func (r *rootView) Paint(p *tui.Painter, ctx *tui.Ctx) {
	barSt := ctx.Theme.Palette.SurfaceMuted
	if barSt == (style.Style{}) {
		barSt = ctx.Theme.Base
	}

	p.Fill(geom.Rect{X: r.rect.X, Y: r.rect.Y, W: r.rect.W, H: 1}, ' ', barSt)
	r.state.status.Layout(geom.Rect{X: r.rect.X + 1, Y: r.rect.Y, W: r.rect.W - 2, H: 1})
	r.state.status.Paint(p, ctx)

	r.View.Paint(p, ctx)
}

func (r *rootView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if cr, ok := e.(tui.ClipboardResponseEvent); ok {
		txt := cr.Text
		if len(txt) > 60 {
			txt = txt[:60] + "..."
		}

		r.state.setStatus(ctx, fmt.Sprintf("Clipboard: %s", txt))

		return true
	}

	return r.View.Handle(e, ctx)
}

func (r *rootView) HandleAction(act int, ctx *tui.Ctx) bool {
	if ui.Action(act) == ActionQuit {
		r.app.Quit()

		return true
	}

	return false
}

// scrollPanel composes a content view with a standalone scrollbar.
// It demonstrates using Scrollbar outside of ScrollView.
//
// Invalidation challenge: the child may scroll or change line count via
// keyboard actions, rune input, or paste — and some of those event paths
// bypass this view entirely (e.g. VStack.Handle dispatches directly to the
// focused descendant). So we can't rely on intercepting events.
//
// Instead, Paint tracks the last-known scroll state. When the state has
// changed since the last paint (meaning the child scrolled or content changed),
// it schedules a follow-up invalidation for the scrollbar column. This adds
// one extra repaint frame for the scrollbar but is completely reliable.
type scrollPanel struct {
	id        tui.ID
	content   tui.View
	scrollbar *widgets.Scrollbar
	getState  func() (contentSize, viewSize, position int)
	rect      geom.Rect

	// Last-known state for change detection in Paint.
	lastCS, lastVS, lastPos int
}

func newScrollPanel(
	content tui.View,
	sb *widgets.Scrollbar,
	getState func() (contentSize, viewSize, position int),
) *scrollPanel {
	return &scrollPanel{
		id:        tui.NewID(),
		content:   content,
		scrollbar: sb,
		getState:  getState,
		lastPos:   -1, // sentinel so first paint always syncs
	}
}

func (sp *scrollPanel) ID() tui.ID           { return sp.id }
func (sp *scrollPanel) Rect() geom.Rect      { return sp.rect }
func (sp *scrollPanel) Focusable() bool      { return false }
func (sp *scrollPanel) Children() []tui.View { return []tui.View{sp.content} }

// MouseOpaque ensures scrollPanel receives all mouse events for its rect,
// so it can intercept scrollbar clicks/drags and wheel events.
func (sp *scrollPanel) MouseOpaque() {}

func (sp *scrollPanel) MinSize() geom.Size {
	ms := sp.content.MinSize()
	ms.W++

	return ms
}

func (sp *scrollPanel) Layout(r geom.Rect) {
	sp.rect = r
	if r.W > 1 {
		sp.content.Layout(geom.Rect{X: r.X, Y: r.Y, W: r.W - 1, H: r.H})
		sp.scrollbar.Layout(geom.Rect{X: r.X + r.W - 1, Y: r.Y, W: 1, H: r.H})
	} else {
		sp.content.Layout(r)
	}
}

func (sp *scrollPanel) Paint(p *tui.Painter, ctx *tui.Ctx) {
	sp.content.Paint(p, ctx)
	cs, vs, pos := sp.getState()
	sp.scrollbar.SetState(cs, vs, pos)
	sp.scrollbar.Paint(p, ctx)

	// If scroll state changed since last paint, the scrollbar column was
	// probably not in the damage region (child only invalidated its own rect).
	// Schedule a follow-up invalidation so the scrollbar repaints next frame.
	if cs != sp.lastCS || vs != sp.lastVS || pos != sp.lastPos {
		ctx.Invalidate(sp.scrollbar.Rect())
		sp.lastCS, sp.lastVS, sp.lastPos = cs, vs, pos
	}
}

func (sp *scrollPanel) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		// Scrollbar drag in progress — delegate regardless of X position
		if sp.scrollbar.Dragging() {
			sp.scrollbar.Handle(me, ctx)
			ctx.Invalidate(sp.scrollbar.Rect())

			return true
		}

		// Click/drag in scrollbar column
		sbRect := sp.scrollbar.Rect()
		if me.X >= sbRect.X && me.X < sbRect.X+sbRect.W {
			sp.scrollbar.Handle(me, ctx)
			ctx.Invalidate(sp.scrollbar.Rect())

			return true
		}

		// Wheel events: forward to content, then invalidate scrollbar
		if me.Button == tui.MouseButtonWheelUp || me.Button == tui.MouseButtonWheelDown {
			handled := sp.content.Handle(me, ctx)
			ctx.Invalidate(sp.scrollbar.Rect())

			return handled
		}

		// Other mouse events: forward to content
		return sp.content.Handle(me, ctx)
	}

	return sp.content.Handle(e, ctx)
}
