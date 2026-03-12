package tui

import "github.com/losinggeneration/tui/geom"

type UpdateCtx struct {
	Invalidate       func(r geom.Rect)
	InvalidateAll    func()
	InvalidateLayout func(id ID)
	RequestFocus     func(id ID)
	Quit             func()
}
