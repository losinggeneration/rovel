//go:build unix

package ansi

import (
	"bufio"
	"strings"
	"testing"

	"github.com/losinggeneration/rovel/backend"
)

func TestSetInputFeaturesModifiedKeysWritesKittyKeyboardProtocol(t *testing.T) {
	t.Setenv("TMUX", "")

	var out strings.Builder
	b := &Backend{w: bufio.NewWriter(&out)}

	if err := b.SetInputFeatures(backend.InputFeatures{ModifiedKeys: true}); err != nil {
		t.Fatalf("enable modified keys: %v", err)
	}
	if got, want := out.String(), "\x1b[>1u\x1b[>4;2m"; got != want {
		t.Fatalf("enable sequence = %q, want %q", got, want)
	}
	if !b.inputFeats.modifiedKeys {
		t.Fatal("modifiedKeys feature was not marked enabled")
	}

	out.Reset()
	if err := b.SetInputFeatures(backend.InputFeatures{}); err != nil {
		t.Fatalf("disable modified keys: %v", err)
	}
	if got, want := out.String(), "\x1b[<u\x1b[>4;0m"; got != want {
		t.Fatalf("disable sequence = %q, want %q", got, want)
	}
	if b.inputFeats.modifiedKeys {
		t.Fatal("modifiedKeys feature was not marked disabled")
	}
}

func TestSetInputFeaturesModifiedKeysWritesTmuxPassthroughWhenInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")

	var out strings.Builder
	b := &Backend{w: bufio.NewWriter(&out)}

	if err := b.SetInputFeatures(backend.InputFeatures{ModifiedKeys: true}); err != nil {
		t.Fatalf("enable modified keys: %v", err)
	}
	if got, want := out.String(), "\x1bPtmux;\x1b\x1b[>1u\x1b\x1b[>4;2m\x1b\\"; got != want {
		t.Fatalf("enable sequence = %q, want %q", got, want)
	}

	out.Reset()
	if err := b.SetInputFeatures(backend.InputFeatures{}); err != nil {
		t.Fatalf("disable modified keys: %v", err)
	}
	if got, want := out.String(), "\x1bPtmux;\x1b\x1b[<u\x1b\x1b[>4;0m\x1b\\"; got != want {
		t.Fatalf("disable sequence = %q, want %q", got, want)
	}
}
