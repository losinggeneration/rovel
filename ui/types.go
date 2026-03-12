package ui

import "github.com/losinggeneration/tui"

// Type aliases for ergonomic UI code.
// These make writing widgets more convenient without importing tui everywhere.
type View = tui.View
type ID = tui.ID
type Rect = tui.Rect
type Size = tui.Size
type Painter = tui.Painter
type Ctx = tui.Ctx
type Event = tui.Event

type PreferredSizer interface {
	PreferredSize() tui.Size
}
