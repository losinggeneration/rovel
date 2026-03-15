package event

// MouseButton represents a mouse button.
type MouseButton uint8

const (
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonWheelUp
	MouseButtonWheelDown
)

// MouseAction represents the type of mouse action.
type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
)

// MouseEvent represents a mouse event.
type MouseEvent struct {
	X      int
	Y      int
	Button MouseButton
	Action MouseAction
	Mod    ModMask
}

func (MouseEvent) isEvent() {}
