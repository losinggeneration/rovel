package tui

import "errors"

var ErrClosed = errors.New("tui: app is closing or closed")
