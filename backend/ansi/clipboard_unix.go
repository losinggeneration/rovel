//go:build unix

package ansi

import (
	"encoding/base64"
	"fmt"
)

// maxClipboardBytes is the maximum payload size for OSC 52 clipboard write.
const maxClipboardBytes = 64 * 1024

// ClipboardWrite writes text to the system clipboard via OSC 52.
func (b *Backend) ClipboardWrite(text string) error {
	if len(text) > maxClipboardBytes {
		return fmt.Errorf("clipboard payload too large: %d bytes (max %d)", len(text), maxClipboardBytes)
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

// ClipboardRead is not supported by the ANSI backend (would require async response parsing).
func (b *Backend) ClipboardRead() (string, error) {
	return "", fmt.Errorf("clipboard read not supported")
}
