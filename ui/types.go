package ui

import "github.com/losinggeneration/rovel"

// Type aliases for ergonomic UI code.
// These make writing widgets more convenient without importing tui everywhere.
type (
	View    = rovel.View
	ID      = rovel.ID
	Rect    = rovel.Rect
	Size    = rovel.Size
	Painter = rovel.Painter
	Ctx     = rovel.Ctx
	Event   = rovel.Event
)

type PreferredSizer interface {
	PreferredSize() rovel.Size
}
