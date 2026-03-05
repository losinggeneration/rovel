package ui

// FocusAction represents a focus navigation action.
type FocusAction int

const (
	FocusNone FocusAction = iota
	FocusNext
	FocusPrev
	FocusFirst
	FocusLast
)

// FocusCycle is implemented by containers that handle focus navigation.
// It is defined but not used in M4 - containers handle Tab traversal
// directly in their Handle method. Reserved for future expansion.
type FocusCycle interface {
	FocusAction(act FocusAction, ctx *Ctx) bool
}

// Focusable is implemented by views that can receive focus.
// Containers check this interface to determine traversal targets.
// Note: Does NOT embed View - cleaner for adapters/wrappers.
type Focusable interface {
	Focusable() bool
}

// Composite is implemented by containers that expose children.
// Used by FocusRing to detect subtree-focus.
// Note: Does NOT embed View - cleaner for adapters/wrappers.
type Composite interface {
	Children() []View
}
