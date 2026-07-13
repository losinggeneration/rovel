package ui

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/action"
)

// Action is a semantic UI action. It aliases action.Action so ui-layer code and
// the tui runtime share a single type across the package boundary.
type Action = action.Action

// Semantic action constants, re-exported from the action package so callers can
// use ui.ActionX alongside the rest of the ui symbols.
const (
	ActionNone           = action.None
	ActionFocusNext      = action.FocusNext
	ActionFocusPrev      = action.FocusPrev
	ActionFocusFirst     = action.FocusFirst
	ActionFocusLast      = action.FocusLast
	ActionActivate       = action.Activate
	ActionCancel         = action.Cancel
	ActionSubmit         = action.Submit
	ActionMoveLeft       = action.MoveLeft
	ActionMoveRight      = action.MoveRight
	ActionMoveUp         = action.MoveUp
	ActionMoveDown       = action.MoveDown
	ActionPageUp         = action.PageUp
	ActionPageDown       = action.PageDown
	ActionHome           = action.Home
	ActionEnd            = action.End
	ActionDeleteBackward = action.DeleteBackward
	ActionDeleteForward  = action.DeleteForward
	ActionPaste          = action.Paste
	ActionSelectAll      = action.SelectAll
	ActionCopy           = action.Copy
	ActionCut            = action.Cut
)

// ActionHandler is implemented by views that can handle semantic actions.
type ActionHandler interface {
	HandleAction(act Action, ctx *rovel.Ctx) bool
}
