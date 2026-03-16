//go:build unix

package ansi

import "github.com/losinggeneration/tui/backend"

// inputFeatures tracks which input features are currently enabled.
type inputFeatures struct {
	mouse          bool
	bracketedPaste bool
}

// InputCapabilities reports which input features this backend supports.
func (b *Backend) InputCapabilities() backend.InputCapabilities {
	return backend.InputCapabilities{
		Mouse:          true,
		MouseMotion:    true,
		BracketedPaste: true,
		ClipboardWrite: true,
		ClipboardRead:  false,
	}
}

// SetInputFeatures enables or disables input features.
func (b *Backend) SetInputFeatures(f backend.InputFeatures) error {
	// Mouse: basic (1000) + button-event / motion (1002) + SGR (1006)
	if f.Mouse && !b.inputFeats.mouse {
		if _, err := b.w.WriteString("\x1b[?1000h\x1b[?1002h\x1b[?1006h"); err != nil {
			return err
		}
		b.inputFeats.mouse = true
	} else if !f.Mouse && b.inputFeats.mouse {
		if _, err := b.w.WriteString("\x1b[?1006l\x1b[?1002l\x1b[?1000l"); err != nil {
			return err
		}
		b.inputFeats.mouse = false
	}

	// Bracketed paste
	if f.BracketedPaste && !b.inputFeats.bracketedPaste {
		if _, err := b.w.WriteString("\x1b[?2004h"); err != nil {
			return err
		}
		b.inputFeats.bracketedPaste = true
	} else if !f.BracketedPaste && b.inputFeats.bracketedPaste {
		if _, err := b.w.WriteString("\x1b[?2004l"); err != nil {
			return err
		}
		b.inputFeats.bracketedPaste = false
	}

	return b.w.Flush()
}

// disableInputFeatures disables all enabled input features.
// Called during Restore() to clean up terminal state.
func (b *Backend) disableInputFeatures() error {
	return b.SetInputFeatures(backend.InputFeatures{})
}
