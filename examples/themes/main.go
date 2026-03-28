package main

import (
	"fmt"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
	widgets "github.com/losinggeneration/tui/ui/widgets"
)

// Custom actions for this application.
// Using an offset (100) to avoid conflicts with ui.Action values.
const (
	ActionTheme1 ui.Action = iota + 100
	ActionTheme2
	ActionTheme3
	ActionQuit
)

type themeKeymap struct{}

func (themeKeymap) Resolve(ctx ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	switch k.Key {
	case event.KeyEsc, event.KeyCtrlC:
		return ActionQuit, true
	case event.KeyLeft, event.KeyShiftTab:
		return ui.ActionFocusPrev, true
	case event.KeyRight, event.KeyTab:
		return ui.ActionFocusNext, true
	}

	if ctx == ui.KeyCtxTextInput {
		return ui.ActionNone, false
	}

	if k.Key == event.KeyRune {
		switch k.Rune {
		case '1':
			return ActionTheme1, true
		case '2':
			return ActionTheme2, true
		case '3':
			return ActionTheme3, true
		}
	}

	return ui.ActionNone, false
}

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
	buttonNormal    tui.View
	buttonAccent    tui.View
	buttonSuccess   tui.View
	buttonWarning   tui.View
	buttonDanger    tui.View
	buttonDisabled  tui.View

	// Mutable styles for semantic buttons — updated on theme switch.
	// Buttons hold pointers to these, so in-place updates take effect at paint.
	accentNormal, accentFocused   style.Style
	successNormal, successFocused style.Style
	warningNormal, warningFocused style.Style
	dangerNormal, dangerFocused   style.Style
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

	// Create app with initial theme and semantic action resolver.
	// This demonstrates the recommended pattern from doc/keybindings.md:
	// - Use a composite keymap (app-specific + default)
	// - Configure ResolveAction to enable the semantic action system
	app, err := tui.New(tui.AppOpts{
		Theme: state.themes[state.currentThemeIndex].theme,
		Input: tui.InputOpts{
			Mouse: true,
		},
		ResolveAction: ui.NewResolver(ui.CompositeKeymap{
			Keymaps: []ui.Keymap{
				themeKeymap{},      // App-specific bindings (1/2/3 for themes, Esc to quit)
				ui.DefaultKeymap{}, // Standard navigation bindings
			},
		}),
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

	// Create buttons with mouse support and semantic colors
	state.buttonNormal = createMouseAwareButton("Normal", func(ctx *tui.Ctx) {
		state.buttonsClicked++
		state.updateStatus("Normal button clicked!", "normal")
		invalidateIfNeeded(ctx)
	}, false)

	// Demonstrate per-widget style overrides using ButtonOpts.
	// Buttons hold pointers into state's mutable style fields, which are
	// updated on theme switch so the buttons track the current theme.
	state.updateButtonStyles()

	state.buttonAccent = createStyledButton("Accent",
		&state.accentNormal, &state.accentFocused, func(ctx *tui.Ctx) {
			state.buttonsClicked++
			state.updateStatus("Accent button clicked!", "info")
			invalidateIfNeeded(ctx)
		})

	state.buttonSuccess = createStyledButton("Success",
		&state.successNormal, &state.successFocused, func(ctx *tui.Ctx) {
			state.updateStatus("Success operation completed!", "success")
			invalidateIfNeeded(ctx)
		})

	state.buttonWarning = createStyledButton("Warning",
		&state.warningNormal, &state.warningFocused, func(ctx *tui.Ctx) {
			state.updateStatus("Warning: Check your inputs!", "warning")
			invalidateIfNeeded(ctx)
		})

	state.buttonDanger = createStyledButton("Danger",
		&state.dangerNormal, &state.dangerFocused, func(ctx *tui.Ctx) {
			state.updateStatus("Critical error occurred!", "danger")
			invalidateIfNeeded(ctx)
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
	themeSelector.AddChild(hspacer)

	// Build button row
	buttonRow := layout.NewHStack()
	buttonRow.SetGap(1)
	buttonRow.AddChild(layout.Child{View: state.buttonNormal, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonAccent, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonSuccess, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonWarning, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(layout.Child{View: state.buttonDanger, Opts: layout.SizePolicy{}})
	// Disabled button wrapped in focusBlockWrapper to prevent focus
	buttonRow.AddChild(layout.Child{View: &focusBlockWrapper{id: tui.NewID(), view: state.buttonDisabled}, Opts: layout.SizePolicy{}})
	buttonRow.AddChild(hspacer)

	// Build content list
	contentList := buildContentList(state)

	listSection := layout.NewVStack()
	listSection.SetGap(1)
	listSection.Add(buildInputSection(state))
	listSection.Add(contentList)
	listSection.AddChild(layout.GrowYChild(widgets.NewLabel(""), 1))

	borderedList := layout.NewBorder(listSection)
	borderedList.SetTitle(" Content List ")

	// Status bar
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

	root := &Root{CompositeView: mainLayout, app: app, state: state}
	app.SetRoot(root)

	// Request focus on the theme selector
	if err := app.Post(func(ctx *tui.UpdateCtx) {
		if ctx.RequestFocus != nil && themeSelector != nil {
			ctx.RequestFocus(themeSelector.ID())
		}
	}); err != nil {
		fmt.Printf("Failed to schedule focus request: %v\n", err)

		return
	}

	if err := app.Enable(); err != nil {
		fmt.Printf("Failed to enable app: %v\n", err)

		return
	}

	defer func() {
		err := app.Restore()
		if err != nil {
			fmt.Printf("Failed to restore terminal: %v\n", err)
		}
	}()

	// Explicitly set the theme after Enable() to ensure proper initialization.
	// This ensures the flusher's style cache is reset and all views are invalidated
	// with the correct theme colors.
	app.SetTheme(state.themes[state.currentThemeIndex].theme)

	// Run the app
	if err := app.Run(); err != nil {
		fmt.Printf("App error: %v\n", err)
	}
}

// invalidateIfNeeded safely invalidates the view if ctx is valid
func invalidateIfNeeded(ctx *tui.Ctx) {
	if ctx != nil && ctx.InvalidateAll != nil {
		ctx.InvalidateAll()
	}
}

// Helper functions

func createThemeButton(label string, themeIndex int, state *appState) tui.View {
	return createMouseAwareButton(label, func(ctx *tui.Ctx) {
		state.switchTheme(ctx, themeIndex)
	}, false)
}

func createMouseAwareButton(label string, onPress func(*tui.Ctx), disabled bool) tui.View {
	btn := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:    label,
		OnPress:  onPress,
		Disabled: disabled,
	})

	return &widgets.Clickable{View: btn, OnClick: onPress}
}

func createStyledButton(label string, normalSt, focusedSt *style.Style, onPress func(*tui.Ctx)) tui.View {
	btn := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:        label,
		OnPress:      onPress,
		StyleNormal:  normalSt,
		StyleFocused: focusedSt,
	})

	return &widgets.Clickable{View: btn, OnClick: onPress}
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

// deriveButtonStyles produces normal and focused styles from a palette role.
// For color themes, the semantic color becomes the BG (solid button look).
// For monochrome themes (ColorDefault FG/BG), attrs are preserved as-is.
func deriveButtonStyles(role style.Style) (normal, focused style.Style) {
	isMonochrome := role.FG == style.ColorDefault && role.BG == style.ColorDefault

	if isMonochrome {
		// Monochrome: keep attrs, use reverse for focus.
		normal = role
		focused = role.WithAttr(style.AttrReverse)
	} else {
		// Color: put the semantic FG color as BG for a solid button look.
		normal = style.Style{FG: role.BG, BG: role.FG, Attr: role.Attr}
		focused = style.Style{FG: role.FG, BG: role.BG, Attr: role.Attr}
	}

	return normal, focused
}

// updateButtonStyles refreshes the mutable style values from the current theme.
func (s *appState) updateButtonStyles() {
	p := s.themes[s.currentThemeIndex].theme.Palette
	s.accentNormal, s.accentFocused = deriveButtonStyles(p.Accent)
	s.successNormal, s.successFocused = deriveButtonStyles(p.Success)
	s.warningNormal, s.warningFocused = deriveButtonStyles(p.Warning)
	s.dangerNormal, s.dangerFocused = deriveButtonStyles(p.Danger)
}

func (s *appState) switchTheme(ctx *tui.Ctx, index int) {
	// Validate index
	if index < 0 || index >= len(s.themes) {
		return
	}

	s.currentThemeIndex = index
	s.statusMessage = fmt.Sprintf("Switched to %s theme", s.themes[index].name)
	s.statusType = "info"

	if s.updateStatus != nil {
		s.updateStatus(s.statusMessage, s.statusType)
	}

	// Update semantic button styles for the new theme
	s.updateButtonStyles()

	// Update theme labels
	s.updateLabels()

	// Switch theme via App.Post to ensure it's updated on the app goroutine
	err := s.app.Post(func(updateCtx *tui.UpdateCtx) {
		s.app.SetTheme(s.themes[index].theme)
	})
	if err != nil {
		s.statusMessage = fmt.Sprintf("Failed to switch theme: %v", err)

		s.statusType = "danger"
		if s.updateStatus != nil {
			s.updateStatus(s.statusMessage, s.statusType)
		}
	}
}

func (s *appState) updateLabels() {
	// Update theme selector
	s.themeSelector.SetText(nil, "Theme: "+s.themes[s.currentThemeIndex].name)

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

	s.capabilityLabel.SetText(nil, "Capability: "+capText)
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

// Root embeds CompositeView and handles app-level semantic actions.
type Root struct {
	tui.CompositeView

	app   *tui.App
	state *appState
}

func (r *Root) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
	case ActionTheme1:
		r.state.switchTheme(ctx, 0)

		return true
	case ActionTheme2:
		r.state.switchTheme(ctx, 1)

		return true
	case ActionTheme3:
		r.state.switchTheme(ctx, 2)

		return true
	case ActionQuit:
		if ctx != nil && ctx.Quit != nil {
			ctx.Quit()
		}

		return true
	}

	return false
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

func (w *focusBlockWrapper) Rect() tui.Rect {
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
