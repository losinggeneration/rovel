// Package errbuf provides a bounded, thread-safe error buffer.
package errbuf

import (
	"sync"
)

// ErrorBuffer is a bounded, thread-safe ring buffer of errors.
// When capacity is reached, the oldest errors are discarded.
//
// All methods are safe for concurrent use.
type ErrorBuffer struct {
	mu       sync.Mutex
	errors   []error
	capacity int
}

var defaultBuf = New(50)

// New creates an ErrorBuffer with the given capacity.
// If capacity is <= 0, it defaults to 50.
func New(capacity int) *ErrorBuffer {
	if capacity <= 0 {
		capacity = 50
	}

	return &ErrorBuffer{
		capacity: capacity,
	}
}

// Add appends an error to the buffer. Nil errors are ignored.
// If the buffer is at capacity, the oldest error is discarded.
// A nil receiver is a no-op.
func (e *ErrorBuffer) Add(err error) {
	if e == nil || err == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
	if len(e.errors) > e.capacity {
		e.errors = e.errors[len(e.errors)-e.capacity:]
	}
}

// Get returns a copy of all buffered errors, oldest first.
// The returned slice is safe to modify.
func (e *ErrorBuffer) Get() []error {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]error, len(e.errors))
	copy(result, e.errors)

	return result
}

// Resize changes the buffer capacity. If capacity is <= 0, it defaults to 50.
// If the current number of errors exceeds the new capacity, the oldest are discarded.
func (e *ErrorBuffer) Resize(s int) {
	if s <= 0 {
		s = 50
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.capacity = s

	if len(e.errors) > s {
		e.errors = e.errors[len(e.errors)-s:]
	}
}

// Clear removes all buffered errors.
func (e *ErrorBuffer) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = nil
}

// Len returns the number of buffered errors.
func (e *ErrorBuffer) Len() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.errors)
}

// Add appends an error to the default buffer.
func Add(err error) {
	defaultBuf.Add(err)
}

// Get returns a copy of all errors in the default buffer.
func Get() []error {
	return defaultBuf.Get()
}

// Resize changes the default buffer's capacity.
func Resize(s int) {
	defaultBuf.Resize(s)
}

// Clear removes all errors from the default buffer.
func Clear() {
	defaultBuf.Clear()
}

// Len returns the number of errors in the default buffer.
func Len() int {
	return defaultBuf.Len()
}
