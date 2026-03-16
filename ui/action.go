package ui

import "github.com/losinggeneration/tui"

// Action represents a semantic UI action.
type Action int

const (
	ActionNone Action = iota
	ActionFocusNext
	ActionFocusPrev
	ActionFocusFirst
	ActionFocusLast
	ActionActivate
	ActionCancel
	ActionSubmit
	ActionMoveLeft
	ActionMoveRight
	ActionMoveUp
	ActionMoveDown
	ActionPageUp
	ActionPageDown
	ActionHome
	ActionEnd
	ActionDeleteBackward
	ActionDeleteForward
	ActionPaste
	ActionSelectAll
	ActionCopy
	ActionCut
)

// ActionHandler is implemented by views that can handle semantic actions.
type ActionHandler interface {
	HandleAction(act Action, ctx *tui.Ctx) bool
}
