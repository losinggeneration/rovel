package tui

import "testing"

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	if id1 == id2 {
		t.Error("NewID() should generate unique IDs")
	}

	if id1 == 0 {
		t.Error("NewID() should not return zero")
	}
}
