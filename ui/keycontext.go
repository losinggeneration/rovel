package ui

// KeyContext indicates the context for keymap resolution.
type KeyContext int

const (
	KeyCtxGlobal    KeyContext = iota
	KeyCtxTextInput            // Active when focused view is in text input mode
	KeyCtxOverlay              // Active when an overlay/modal is shown
)

// TextInputMode is implemented by views that are in text editing mode.
// When the focused view implements this and returns true, the keymap
// uses KeyCtxTextInput context (fewer key bindings, more keys passed through).
type TextInputMode interface {
	IsTextInputMode() bool
}
