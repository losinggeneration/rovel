package tui

import "errors"

var ErrClosed = errors.New("tui: app is closing or closed")

// ErrSuspendUnsupported is returned by App.Suspend when the active backend does
// not implement backend.SignalController (e.g. a custom backend without
// terminal-suspend support).
var ErrSuspendUnsupported = errors.New("tui: backend does not support suspend")
