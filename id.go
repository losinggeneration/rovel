package rovel

import "sync/atomic"

// ID is a unique identifier for views and widgets.
type ID uint64

// idCounter is the global counter for generating unique IDs.
var idCounter atomic.Uint64

// NewID generates a new unique ID.
func NewID() ID {
	return ID(idCounter.Add(1))
}
