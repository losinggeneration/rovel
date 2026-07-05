// Package action defines the semantic UI actions shared between the tui runtime
// and the ui keybinding layer. It is a leaf package — it imports nothing from
// tui or ui — so both sides can reference the same typed constants without an
// import cycle.
package action

// Action is a semantic UI action produced by keybinding resolution and consumed
// by a view's ActionHandler.
type Action int

const (
	None Action = iota
	FocusNext
	FocusPrev
	FocusFirst
	FocusLast
	Activate
	Cancel
	Submit
	MoveLeft
	MoveRight
	MoveUp
	MoveDown
	PageUp
	PageDown
	Home
	End
	DeleteBackward
	DeleteForward
	Paste
	SelectAll
	Copy
	Cut
)
