//go:build unix

package ansi

import (
	"encoding/base64"
	"errors"
	"fmt"
)

var ErrClipboardPayloadTooLarge = errors.New("clipboard payload too large")
var ErrClipboardReadNotSupported = errors.New("clipboard read not supported synchronously; use ClipboardReadRequest")

// maxClipboardBytes is the maximum payload size for OSC 52 clipboard write.
const maxClipboardBytes = 64 * 1024

// ClipboardWrite writes text to the system clipboard via OSC 52.
func (b *Backend) ClipboardWrite(text string) error {
	if len(text) > maxClipboardBytes {
		return fmt.Errorf("clipboard payload too large: %d bytes (max %d): %w", len(text), maxClipboardBytes, ErrClipboardPayloadTooLarge)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// OSC 52 ; c ; <base64> ST
	seq := fmt.Sprintf("\x1b]52;c;%s\x1b\\", encoded)

	_, err := b.w.WriteString(seq)
	if err != nil {
		return err
	}

	return b.w.Flush()
}

// ClipboardRead is not supported synchronously by the ANSI backend.
// Use ClipboardReadRequest for async clipboard reading via OSC 52.
func (b *Backend) ClipboardRead() (string, error) {
	return "", ErrClipboardReadNotSupported
}

// ClipboardReadRequest sends an OSC 52 read request to the terminal.
// The response arrives asynchronously as a ClipboardResponseEvent.
func (b *Backend) ClipboardReadRequest() error {
	// OSC 52 ; c ; ? ST — query clipboard contents
	_, err := b.w.WriteString("\x1b]52;c;?\x1b\\")
	if err != nil {
		return err
	}

	return b.w.Flush()
}
