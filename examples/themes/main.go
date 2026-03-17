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

// themeKeymap implements ui.Keymap to provide application-specific key bindings.
// This demonstrates the pattern to have
// physical key events -> semantic actions -> handlers
type themeKeymap struct{}

func (themeKeymap) Resolve(ctx ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	switch k.Key {
	case event.KeyRune:
		// Number keys 1-3 switch themes
		switch k.Rune {
		case '1':
			return ActionTheme1, true
		case '2':
			return ActionTheme2, true
		case '3':
			return ActionTheme3, true
		}
	case event.KeyEsc:
		// Escape quits the application
		return ActionQuit, true
	case event.KeyCtrlC:
		// Ctrl-C also quits (common TUI convention)
		return ActionQuit, true
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

	state.buttonAccent = createSemanticButton("Accent", func(t *tui.Theme) style.Style {
		return t.Palette.Accent
	}, func(ctx *tui.Ctx) {
		state.buttonsClicked++
		state.updateStatus("Accent button clicked!", "info")
		invalidateIfNeeded(ctx)
	})

	state.buttonSuccess = createSemanticButton("Success", func(t *tui.Theme) style.Style {
		return t.Palette.Success
	}, func(ctx *tui.Ctx) {
		state.updateStatus("Success operation completed!", "success")
		invalidateIfNeeded(ctx)
	})

	state.buttonWarning = createSemanticButton("Warning", func(t *tui.Theme) style.Style {
		return t.Palette.Warning
	}, func(ctx *tui.Ctx) {
		state.updateStatus("Warning: Check your inputs!", "warning")
		invalidateIfNeeded(ctx)
	})

	state.buttonDanger = createSemanticButton("Danger", func(t *tui.Theme) style.Style {
		return t.Palette.Danger
	}, func(ctx *tui.Ctx) {
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

	// Wrap in focus ring
	focusRing := widgets.NewFocusRing(mainLayout)

	// Wrap root with event handler for semantic actions
	root := &eventHandler{
		id:       tui.NewID(),
		app:      app,
		state:    state,
		rootView: focusRing,
	}

	// Set root and run
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
		if err := app.Restore(); err != nil {
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

func createSemanticButton(label string, roleFunc func(*tui.Theme) style.Style, onPress func(*tui.Ctx)) tui.View {
	btn := widgets.NewButtonOpts(widgets.ButtonOpts{
		Label:   label,
		OnPress: onPress,
	})
	sb := &semanticButton{Button: btn, semanticRole: roleFunc}

	return &widgets.Clickable{View: sb, OnClick: onPress}
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

	// Update theme labels
	s.updateLabels()

	// Switch theme via App.Post to ensure it's updated on the app goroutine
	if err := s.app.Post(func(updateCtx *tui.UpdateCtx) {
		s.app.SetTheme(s.themes[index].theme)
	}); err != nil {
		s.statusMessage = fmt.Sprintf("Failed to switch theme: %v", err)

		s.statusType = "danger"
		if s.updateStatus != nil {
			s.updateStatus(s.statusMessage, s.statusType)
		}
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

// eventHandler wraps the root and handles global keyboard shortcuts.
// This demonstrates handling app-global actions (quit, theme switching) that
// are not tied to any specific focused widget.
//
// Note: The framework's semantic action system (HandleAction) is designed for
// focused view actions. For app-global shortcuts, handle them directly in the
// root view's Handle() method.
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

func (h *eventHandler) Rect() tui.Rect {
	return h.rootView.Rect()
}

func (h *eventHandler) Paint(p *tui.Painter, ctx *tui.Ctx) {
	h.rootView.Paint(p, ctx)
}

// HandleAction is provided for documentation purposes to demonstrate how
// semantic actions would be handled. It is not currently called by the
// framework because the framework's action resolver only checks the focused
// view, not the root view.
//
// For app-global shortcuts like quit and theme switching, we handle them
// directly in Handle() instead.
func (h *eventHandler) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
	case ActionTheme1:
		h.state.switchTheme(ctx, 0)

		return true
	case ActionTheme2:
		h.state.switchTheme(ctx, 1)

		return true
	case ActionTheme3:
		h.state.switchTheme(ctx, 2)

		return true
	case ActionQuit:
		if ctx != nil && ctx.Quit != nil {
			ctx.Quit()
		}

		return true
	}

	return false
}

// Handle processes events and delegates to the root view.
// For key events, it also checks for global actions like quit and theme switching.
// This ensures that app-wide actions work even when no focused view handles them.
func (h *eventHandler) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Check for keyboard events that might be global actions
	if ke, ok := e.(tui.KeyEvent); ok {
		// Handle quit keys globally
		if ke.Key == tui.KeyEsc || ke.Key == tui.KeyCtrlC {
			if ctx != nil && ctx.Quit != nil {
				ctx.Quit()
			}

			return true
		}

		// Handle theme switching keys (1/2/3)
		if ke.Key == tui.KeyRune {
			switch ke.Rune {
			case '1':
				h.state.switchTheme(ctx, 0)

				return true
			case '2':
				h.state.switchTheme(ctx, 1)

				return true
			case '3':
				h.state.switchTheme(ctx, 2)

				return true
			}
		}
	}

	// Pass all other events to the root view
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

// semanticButton wraps a Button and applies theme semantic colors.
// This demonstrates a workaround for semantic color variants until
// the theme system supports proper semantic roles (see doc/styling_theming.md).
type semanticButton struct {
	*widgets.Button
	semanticRole func(*tui.Theme) style.Style
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
