package main

import (
	"fmt"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/ui/layout"
	widgets "github.com/losinggeneration/tui/ui/widgets"
)

type themeOption struct {
	name  string
	theme tui.Theme
}

type appState struct {
	app               *tui.App
	currentThemeIndex int
	themes            []themeOption

	// UI state
	inputText      string
	buttonsClicked int
	listItems      []string
	statusMessage  string
	statusType     string // "success", "warning", "danger", "info", "normal"
	updateStatus   func(msg, statusType string)

	// Widgets
	themeSelector   *widgets.Label
	capabilityLabel *widgets.Label
	textInput       tui.View // Use View interface to allow read-only wrapper
	buttonNormal    tui.View // Use View interface to allow semantic wrapper
	buttonAccent    tui.View // Use View interface to allow semantic wrapper
	buttonSuccess   tui.View // Use View interface to allow semantic wrapper
	buttonWarning   tui.View // Use View interface to allow semantic wrapper
	buttonDanger    tui.View // Use View interface to allow semantic wrapper
	buttonDisabled  tui.View // Use View interface to allow semantic wrapper
}

func main() {
	// Initialize app state
	state := &appState{
		currentThemeIndex: 2, // Start with Modern
		themes: []themeOption{
			{"Default", tui.DefaultTheme()},
			{"Classic", tui.DefaultThemeClassic()},
			{"Modern", tui.DefaultThemeModern()},
		},
		inputText:     "Type something...",
		listItems:     []string{"Item 1", "Item 2", "Item 3", "Item 4", "Item 5"},
		statusMessage: "Application ready",
		statusType:    "info",
	}

	// Create app with initial theme
	app, err := tui.New(tui.AppOpts{
		Theme: state.themes[state.currentThemeIndex].theme,
	})
	if err != nil {
		fmt.Printf("Failed to create app: %v\n", err)
		return
	}
	state.app = app

	// Create widgets
	state.themeSelector = widgets.NewLabel("")
	state.capabilityLabel = widgets.NewLabel("")
	state.updateLabels()

	ti := widgets.NewTextInput()
	ti.SetText(nil, state.inputText)
	// Wrap text input to make it read-only
	state.textInput = &readOnlyTextInput{TextInput: ti}

	state.buttonNormal = widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: "Normal",
		OnPress: func(ctx *tui.Ctx) {
			state.buttonsClicked++
			state.updateStatus("Normal button clicked!", "normal")
			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})

	baseAccent := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: "Accent",
		OnPress: func(ctx *tui.Ctx) {
			state.buttonsClicked++
			state.updateStatus("Accent button clicked!", "info")
			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})
	state.buttonAccent = createSemanticButton(baseAccent, func(t *tui.Theme) style.Style {
		return t.Palette.Accent
	})

	baseSuccess := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: "Success",
		OnPress: func(ctx *tui.Ctx) {
			state.updateStatus("Success operation completed!", "success")
			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})
	state.buttonSuccess = createSemanticButton(baseSuccess, func(t *tui.Theme) style.Style {
		return t.Palette.Success
	})

	baseWarning := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: "Warning",
		OnPress: func(ctx *tui.Ctx) {
			state.updateStatus("Warning: Check your inputs!", "warning")
			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})
	state.buttonWarning = createSemanticButton(baseWarning, func(t *tui.Theme) style.Style {
		return t.Palette.Warning
	})

	baseDanger := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: "Danger",
		OnPress: func(ctx *tui.Ctx) {
			state.updateStatus("Critical error occurred!", "danger")
			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})
	state.buttonDanger = createSemanticButton(baseDanger, func(t *tui.Theme) style.Style {
		return t.Palette.Danger
	})

	state.buttonDisabled = widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:    "Disabled",
		Disabled: true,
	})

	hspacer := layout.GrowXChild(widgets.NewLabel(""), 1)

	// Build theme selector buttons
	themeSelector := layout.NewHStack()
	themeSelector.SetGap(2)
	themeSelector.Add(createThemeButton("1. Default", 0, state))
	themeSelector.Add(createThemeButton("2. Classic", 1, state))
	themeSelector.Add(createThemeButton("3. Modern", 2, state))
	// Add spacer to prevent the last button from growing
	themeSelector.AddChild(hspacer)

	// Build button row - use fixed size policies to prevent horizontal growth
	buttonRow := layout.NewHStack()
	buttonRow.SetGap(1)
	// Use explicit size constraints to prevent buttons from growing
	buttonRow.AddChild(layout.Child{View: state.buttonNormal, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonAccent, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonSuccess, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonWarning, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonDanger, Opts: layout.SizePolicy{}})
	// Disabled button wrapped in focusBlockWrapper to prevent focus
	buttonRow.AddChild(layout.Child{View: &focusBlockWrapper{id: tui.NewID(), view: state.buttonDisabled}, Opts: layout.SizePolicy{}})
	// Add spacer to prevent the last button from growing
	buttonRow.AddChild(hspacer)

	// Build content list
	contentList := buildContentList(state)

	listSection := layout.NewVStack()
	listSection.SetGap(1)
	listSection.Add(buildInputSection(state))
	// listSection.Add(buttonRow)
	listSection.Add(contentList)
	listSection.AddChild(layout.GrowYChild(widgets.NewLabel(""), 1))

	borderedList := layout.NewBorder(listSection)
	borderedList.SetTitle(" Content List ")

	// Status bar outside the content list border
	statusBar := buildStatusBar(state)

	// Build main layout
	mainLayout := layout.NewVStack()
	mainLayout.SetGap(1)
	mainLayout.Add(state.themeSelector)
	mainLayout.Add(state.capabilityLabel)
	mainLayout.Add(themeSelector)
	mainLayout.Add(buttonRow)
	// Wrap demo content in one focus block, let it grow to fill space
	mainLayout.AddChild(layout.Child{View: &focusBlockWrapper{id: tui.NewID(), view: borderedList}, Opts: layout.SizePolicy{GrowY: true, StretchY: 1}})
	// Status at bottom with fixed height
	mainLayout.Add(statusBar)

	// Wrap in focus ring
	focusRing := widgets.NewFocusRing(mainLayout)

	// Wrap root with event handler for theme switching and quit
	root := &eventHandler{
		id:       tui.NewID(),
		app:      app,
		state:    state,
		rootView: focusRing,
	}

	// Set root and run
	app.SetRoot(root)

	// Request focus on the theme selector
	err = app.Post(func(ctx *tui.UpdateCtx) {
		if ctx.RequestFocus != nil && themeSelector != nil {
			ctx.RequestFocus(themeSelector.ID())
		}
	})
	if err != nil {
		fmt.Printf("Failed to add to schedule loop: %v\n", err)
		return
	}

	if err := app.Enable(); err != nil {
		fmt.Printf("Failed to enable app: %v\n", err)
		return
	}
	defer func() {
		if err := app.Restore(); err != nil {
			fmt.Printf("Failed to restore terminal back to normal. It may be in an inconsistent state. Run `reset` if needed: %v\n", err)
		}
	}()

	// Run the app
	if err := app.Run(); err != nil {
		fmt.Printf("App error: %v\n", err)
	}
}

// Helper functions

func createThemeButton(label string, themeIndex int, state *appState) *widgets.Button {
	return widgets.NewButtonOpts(widgets.ButtonOpts{
		Label: label,
		OnPress: func(ctx *tui.Ctx) {
			state.switchTheme(ctx, themeIndex)
		},
	})
}

func buildInputSection(state *appState) tui.View {
	inputBorder := layout.NewBorder(state.textInput)
	inputBorder.SetTitle(" Text Input ")
	return inputBorder
}

func buildContentList(state *appState) tui.View {
	listText := ""
	for i, item := range state.listItems {
		listText += fmt.Sprintf("%d. %s\n", i+1, item)
		listText += "   Theme colors: Surface, Text, Border, Focus\n"
	}

	label := widgets.NewLabel(listText)

	border := layout.NewBorder(label)
	border.SetTitle(" Sample List ")
	return border
}

func buildStatusBar(state *appState) tui.View {
	ti := widgets.NewTextInput()

	quitHint := " | Press 1/2/3 to switch themes, Esc to quit"

	// Initial message
	msg := state.statusMessage
	msgType := state.statusType
	text := msg + quitHint

	switch msgType {
	case "success":
		text = "✓ " + text
	case "warning":
		text = "⚠ " + text
	case "danger":
		text = "✗ " + text
	case "normal":
		text = "✓ " + text
	default:
		text = "ℹ " + text
	}

	ti.SetText(nil, text)

	border := layout.NewBorder(ti)
	border.SetTitle(" Status ")

	wrapper := &statusWrapper{
		id:       tui.NewID(),
		textView: ti,
		border:   border,
		msg:      msg,
		msgType:  msgType,
	}

	// Register updater so buttons can update status
	state.updateStatus = wrapper.UpdateStatus

	return wrapper
}

func (s *appState) switchTheme(ctx *tui.Ctx, index int) {
	s.currentThemeIndex = index
	s.statusMessage = fmt.Sprintf("Switched to %s theme", s.themes[index].name)
	s.statusType = "info"

	if s.updateStatus != nil {
		s.updateStatus(s.statusMessage, s.statusType)
	}

	// Switch theme via App.Post
	if ctx != nil && ctx.Quit != nil {
		_ = s.app.Post(func(updateCtx *tui.UpdateCtx) {
			s.app.SetTheme(s.themes[index].theme)
		})
	}
}

func (s *appState) updateLabels() {
	// Update theme selector
	s.themeSelector.SetText(nil, fmt.Sprintf("Theme: %s", s.themes[s.currentThemeIndex].name))

	// Update capability label
	var capText string
	switch s.currentThemeIndex {
	case 1: // Classic theme
		capText = "Basic 16-color"
	case 2: // Modern theme uses RGB
		capText = "TrueColor (may degrade)"
	default: // Default theme - monochrome
		capText = "Monochrome"
	}
	s.capabilityLabel.SetText(nil, fmt.Sprintf("Capability: %s", capText))
}

// statusWrapper paints a background color for status messages
type statusWrapper struct {
	id       tui.ID
	textView tui.View
	border   tui.View
	rect     geom.Rect

	// Dynamic content - updated via UpdateStatus
	msg     string
	msgType string
}

func (w *statusWrapper) ID() tui.ID {
	return w.id
}

func (w *statusWrapper) Rect() tui.Rect {
	if w.border != nil {
		return w.border.Rect()
	}
	if w.textView != nil {
		return w.textView.Rect()
	}
	return tui.Rect{}
}

func (w *statusWrapper) Layout(r geom.Rect) {
	w.rect = r
	if w.border != nil {
		w.border.Layout(r)
	}
	if w.textView != nil {
		inner := layout.InsetRect(r, 1, 1, 1, 1)
		w.textView.Layout(inner)
	}
}

func (w *statusWrapper) MinSize() geom.Size {
	if w.border != nil {
		return w.border.MinSize()
	}
	if w.textView != nil {
		return w.textView.MinSize()
	}
	return geom.Size{W: 40, H: 1}
}

func (w *statusWrapper) PreferredSize() geom.Size {
	return w.MinSize()
}

func (w *statusWrapper) UpdateStatus(msg, msgType string) {
	w.msg = msg
	w.msgType = msgType
}

func (w *statusWrapper) Paint(p *tui.Painter, ctx *tui.Ctx) {
	// Get current text and style based on message type
	quitHint := " | Press 1/2/3 to switch themes, Esc to quit"
	text := w.msg + quitHint
	bgStyle := style.Style{}

	switch w.msgType {
	case "success":
		text = "✓ " + text
		bgStyle = style.Style{BG: style.ColorBasic(2)}
	case "warning":
		text = "⚠ " + text
		bgStyle = style.Style{BG: style.ColorBasic(3)}
	case "danger":
		text = "✗ " + text
		bgStyle = style.Style{BG: style.ColorBasic(1)}
	case "normal":
		text = "✓ " + text
		bgStyle = style.Style{BG: style.ColorBasic(4)}
	default:
		text = "ℹ " + text
	}

	// Update the text in the text view
	if w.textView != nil {
		if ti, ok := w.textView.(*widgets.TextInput); ok {
			ti.SetText(nil, text)
		}
	}

	// Fill with background color
	if bgStyle != (style.Style{}) {
		p.Fill(w.Rect(), ' ', bgStyle)
	}

	// Paint border and text
	if w.border != nil {
		w.border.Paint(p, ctx)
	} else if w.textView != nil {
		w.textView.Paint(p, ctx)
	}
}

func (w *statusWrapper) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if w.border != nil {
		if w.border.Handle(e, ctx) {
			return true
		}
	}
	if w.textView != nil {
		if w.textView.Handle(e, ctx) {
			return true
		}
	}
	return false
}

func (w *statusWrapper) Focusable() bool {
	return false
}

func (w *statusWrapper) FocusScope() bool {
	return true
}

// eventHandler wraps the root and handles global keyboard events (theme switching, quit)
type eventHandler struct {
	id       tui.ID
	app      *tui.App
	state    *appState
	rootView tui.View
}

func (h *eventHandler) ID() tui.ID {
	return h.id
}

func (h *eventHandler) MinSize() geom.Size {
	return h.rootView.MinSize()
}

func (h *eventHandler) Layout(r geom.Rect) {
	h.rootView.Layout(r)
}

func (h *eventHandler) Rect() geom.Rect {
	return h.rootView.Rect()
}

func (h *eventHandler) Paint(p *tui.Painter, ctx *tui.Ctx) {
	h.rootView.Paint(p, ctx)
}

func (h *eventHandler) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Check for keyboard shortcuts first
	ke, ok := e.(tui.KeyEvent)
	if ok {
		switch ke.Key {
		case tui.KeyRune:
			// Number keys 1-3 to switch themes
			if ke.Rune >= '1' && ke.Rune <= '3' {
				index := int(ke.Rune - '1')
				h.state.switchTheme(ctx, index)
				return true
			}
		case tui.KeyEsc, tui.KeyCtrlC:
			if ctx != nil && ctx.Quit != nil {
				ctx.Quit()
			}
			return true
		}
	}

	// Pass event to root
	return h.rootView.Handle(e, ctx)
}

func (h *eventHandler) Focusable() bool {
	return false
}

func (h *eventHandler) Children() []tui.View {
	if c, ok := h.rootView.(interface{ Children() []tui.View }); ok {
		return c.Children()
	}
	return nil
}

// readOnlyTextInput wraps a TextInput to make it read-only (display-only)
type readOnlyTextInput struct {
	*widgets.TextInput
}

func (t *readOnlyTextInput) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return false
}

func (t *readOnlyTextInput) Focusable() bool {
	return false
}

func (t *readOnlyTextInput) FocusScope() bool {
	return true
}

// focusBlockWrapper wraps a view and prevents it from receiving focus.
// It implements FocusScope to block focus traversal into children.
type focusBlockWrapper struct {
	id   tui.ID
	view tui.View
}

func (w *focusBlockWrapper) ID() tui.ID {
	return w.id
}

func (w *focusBlockWrapper) MinSize() geom.Size {
	return w.view.MinSize()
}

func (w *focusBlockWrapper) Layout(r geom.Rect) {
	w.view.Layout(r)
}

func (w *focusBlockWrapper) Rect() geom.Rect {
	return w.view.Rect()
}

func (w *focusBlockWrapper) Paint(p *tui.Painter, ctx *tui.Ctx) {
	w.view.Paint(p, ctx)
}

func (w *focusBlockWrapper) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return w.view.Handle(e, ctx)
}

func (w *focusBlockWrapper) Focusable() bool {
	return false
}

func (w *focusBlockWrapper) FocusScope() bool {
	return true
}

func (w *focusBlockWrapper) Children() []tui.View {
	if c, ok := w.view.(interface{ Children() []tui.View }); ok {
		return c.Children()
	}
	return nil
}

// semanticButton wraps a Button and applies theme semantic colors
type semanticButton struct {
	*widgets.Button
	semanticRole func(*tui.Theme) style.Style // Function to extract semantic color from theme
}

func (b *semanticButton) Paint(p *tui.Painter, ctx *tui.Ctx) {
	if ctx == nil {
		b.Button.Paint(p, ctx)
		return
	}

	// Temporarily override the theme with semantic colors
	originalTheme := ctx.Theme
	semanticStyle := b.semanticRole(&originalTheme)

	// Create a modified context with the semantic color applied
	modifiedCtx := *ctx
	// Surface = semantic color for normal state
	modifiedCtx.Theme.Palette.Surface = semanticStyle
	// Focus = inverted FG/BG for highlighted state
	modifiedCtx.Theme.Palette.Focus = style.Style{
		FG: semanticStyle.BG,
		BG: semanticStyle.FG,
	}
	// Disabled = grayed out
	modifiedCtx.Theme.Palette.Disabled = style.Style{
		FG: style.ColorBasic(8),
		BG: style.ColorBasic(0),
	}

	b.Button.Paint(p, &modifiedCtx)
}

func createSemanticButton(baseButton *widgets.Button, roleFunc func(*tui.Theme) style.Style) *semanticButton {
	return &semanticButton{
		Button:       baseButton,
		semanticRole: roleFunc,
	}
}
