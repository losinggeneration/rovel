package layout

import (
	"github.com/losinggeneration/tui"
)

type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

type SizePolicy struct {
	GrowX    bool
	GrowY    bool
	ShrinkX  bool
	ShrinkY  bool
	StretchX int
	StretchY int
	AlignX   Align
	AlignY   Align
}

type Child struct {
	View tui.View
	Opts SizePolicy
}

func NewChild(v tui.View) Child {
	return Child{View: v}
}

func GrowChild(v tui.View, sx, sy int) Child {
	return Child{
		View: v,
		Opts: SizePolicy{
			GrowX:    true,
			GrowY:    true,
			StretchX: sx,
			StretchY: sy,
		},
	}
}

func GrowXChild(v tui.View, stretch int) Child {
	return Child{
		View: v,
		Opts: SizePolicy{
			GrowX:    true,
			StretchX: stretch,
		},
	}
}

func GrowYChild(v tui.View, stretch int) Child {
	return Child{
		View: v,
		Opts: SizePolicy{
			GrowY:    true,
			StretchY: stretch,
		},
	}
}

func AlignChild(v tui.View, ax, ay Align) Child {
	return Child{
		View: v,
		Opts: SizePolicy{
			AlignX: ax,
			AlignY: ay,
		},
	}
}

var DefaultSizePolicy = SizePolicy{}
