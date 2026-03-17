package ui

import "github.com/losinggeneration/tui"

// Type aliases for ergonomic UI code.
// These make writing widgets more convenient without importing tui everywhere.
type (
	View    = tui.View
	ID      = tui.ID
	Rect    = tui.Rect
	Size    = tui.Size
	Painter = tui.Painter
	Ctx     = tui.Ctx
	Event   = tui.Event
)

type PreferredSizer interface {
	PreferredSize() tui.Size
}
