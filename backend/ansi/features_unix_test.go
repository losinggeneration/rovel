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

func TestSetInputFeaturesMouseWritesTrackingModes(t *testing.T) {
	var out strings.Builder
	b := &Backend{w: bufio.NewWriter(&out)}

	if err := b.SetInputFeatures(backend.InputFeatures{Mouse: true}); err != nil {
		t.Fatalf("enable mouse: %v", err)
	}
	if got, want := out.String(), "\x1b[?1000h\x1b[?1002h\x1b[?1006h"; got != want {
		t.Fatalf("enable sequence = %q, want %q", got, want)
	}
	if !b.inputFeats.mouse {
		t.Fatal("mouse feature was not marked enabled")
	}

	out.Reset()
	if err := b.SetInputFeatures(backend.InputFeatures{}); err != nil {
		t.Fatalf("disable mouse: %v", err)
	}
	if got, want := out.String(), "\x1b[?1006l\x1b[?1002l\x1b[?1000l"; got != want {
		t.Fatalf("disable sequence = %q, want %q", got, want)
	}
	if b.inputFeats.mouse {
		t.Fatal("mouse feature was not marked disabled")
	}
}

func TestSetInputFeaturesBracketedPasteWritesMode(t *testing.T) {
	var out strings.Builder
	b := &Backend{w: bufio.NewWriter(&out)}

	if err := b.SetInputFeatures(backend.InputFeatures{BracketedPaste: true}); err != nil {
		t.Fatalf("enable bracketed paste: %v", err)
	}
	if got, want := out.String(), "\x1b[?2004h"; got != want {
		t.Fatalf("enable sequence = %q, want %q", got, want)
	}
	if !b.inputFeats.bracketedPaste {
		t.Fatal("bracketedPaste feature was not marked enabled")
	}

	out.Reset()
	if err := b.SetInputFeatures(backend.InputFeatures{}); err != nil {
		t.Fatalf("disable bracketed paste: %v", err)
	}
	if got, want := out.String(), "\x1b[?2004l"; got != want {
		t.Fatalf("disable sequence = %q, want %q", got, want)
	}
	if b.inputFeats.bracketedPaste {
		t.Fatal("bracketedPaste feature was not marked disabled")
	}
}

func TestSetInputFeaturesIsIdempotentWhileEnabled(t *testing.T) {
	var out strings.Builder
	b := &Backend{w: bufio.NewWriter(&out)}

	if err := b.SetInputFeatures(backend.InputFeatures{Mouse: true, BracketedPaste: true}); err != nil {
		t.Fatalf("enable features: %v", err)
	}
	out.Reset()
	if err := b.SetInputFeatures(backend.InputFeatures{Mouse: true, BracketedPaste: true}); err != nil {
		t.Fatalf("re-enable features: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("re-enabling emitted sequences again: %q", out.String())
	}
}
