package errors

import (
	"sync"
)

type ErrorBuffer struct {
	mu       sync.Mutex
	errors   []error
	capacity int
}

var (
	defaultBuf     *ErrorBuffer
	defaultBufInit sync.Once
)

func init() {
	defaultBufInit.Do(func() {
		defaultBuf = New(50)
	})
}

func New(capacity int) *ErrorBuffer {
	if capacity <= 0 {
		capacity = 50
	}
	return &ErrorBuffer{
		capacity: capacity,
	}
}

func (e *ErrorBuffer) Add(err error) {
	if err == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
	if len(e.errors) > e.capacity {
		e.errors = e.errors[len(e.errors)-e.capacity:]
	}
}

func (e *ErrorBuffer) Get() []error {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]error, len(e.errors))
	copy(result, e.errors)
	return result
}

func (e *ErrorBuffer) Resize(s int) {
	if s <= 0 {
		s = 50
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.capacity = s

	// only keep only the most recent errors at the end
	if len(e.errors) > s {
		e.errors = e.errors[len(e.errors)-s:]
	}
}

func (e *ErrorBuffer) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errors = nil
}

func (e *ErrorBuffer) Len() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.errors)
}

func Add(err error) {
	defaultBuf.Add(err)
}

func Get() []error {
	return defaultBuf.Get()
}

func Resize(s int) {
	defaultBuf.Resize(s)
}

func Clear() {
	defaultBuf.Clear()
}

func Len() int {
	return defaultBuf.Len()
}
