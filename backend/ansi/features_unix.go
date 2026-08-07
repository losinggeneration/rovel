//go:build unix

package ansi

import (
	"os"
	"strings"

	"github.com/losinggeneration/rovel/backend"
)

// inputFeatures tracks which input features are currently enabled.
type inputFeatures struct {
	mouse          bool
	bracketedPaste bool
	modifiedKeys   bool
}

// InputCapabilities reports which input features this backend supports.
// In cbreak mode all capabilities are disabled: mouse tracking and bracketed
// paste are suppressed so text in the terminal remains selectable.
func (b *Backend) InputCapabilities() backend.InputCapabilities {
	if b.mode == backend.ModeCBreak {
		return backend.InputCapabilities{}
	}

	return backend.InputCapabilities{
		Mouse:          true,
		MouseMotion:    true,
		BracketedPaste: true,
		ModifiedKeys:   true,
		ClipboardWrite: true,
		ClipboardRead:  true,
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

	// Modified key reporting. This asks terminals that support the CSI-u / Kitty
	// keyboard protocol to encode otherwise ambiguous modified keys such as
	// Shift+Enter as CSI 13;2u. We also enable xterm's modifyOtherKeys level 2,
	// used by terminals that report Shift+Enter as CSI 27;2;13~ instead of CSI-u.
	//
	// tmux filters extended-key request sequences unless the pane has passthrough
	// enabled. Wrap the requests in a tmux Device Control String so the outer
	// terminal receives them; tmux itself already decodes many modified keys when
	// extended-keys is enabled and will ignore the payload otherwise. Unsupported
	// terminals ignore these private sequences, so enabling the feature is safe
	// but still best-effort.
	if f.ModifiedKeys && !b.inputFeats.modifiedKeys {
		if _, err := b.w.WriteString(tmuxPassthroughIfNeeded("\x1b[>1u\x1b[>4;2m")); err != nil {
			return err
		}

		b.inputFeats.modifiedKeys = true
	} else if !f.ModifiedKeys && b.inputFeats.modifiedKeys {
		if _, err := b.w.WriteString(tmuxPassthroughIfNeeded("\x1b[<u\x1b[>4;0m")); err != nil {
			return err
		}

		b.inputFeats.modifiedKeys = false
	}

	return b.w.Flush()
}

// tmuxPassthroughIfNeeded wraps terminal control sequences in tmux's DCS
// passthrough envelope when running inside tmux. The doubled ESC bytes are the
// escaping tmux expects inside passthrough payloads.
func tmuxPassthroughIfNeeded(seq string) string {
	if os.Getenv("TMUX") == "" {
		return seq
	}

	return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
}

// disableInputFeatures disables all enabled input features.
// Called during Restore() to clean up terminal state.
func (b *Backend) disableInputFeatures() error {
	return b.SetInputFeatures(backend.InputFeatures{})
}
