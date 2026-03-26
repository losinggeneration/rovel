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
	MouseDrag
)

// MouseEvent represents a mouse event.
type MouseEvent struct {
	X          int
	Y          int
	Button     MouseButton
	Action     MouseAction
	Mod        ModMask
	ClickCount int // 1 = single, 2 = double, 3 = triple, etc.
	WheelDelta int // wheel step count; 0 means 1 for wheel events
}

func (MouseEvent) isEvent() {}
