package render

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

func FuzzPainterText(f *testing.F) {
	f.Add("hello world")
	f.Add("日本語テスト")
	f.Add("🌍🎉👨‍👩‍👧‍👦")
	f.Add("a\tb\nc")
	f.Add("")
	f.Add("café résumé naïve")
	f.Add(string([]byte{0xc3, 0x28})) // invalid UTF-8
	f.Add("a\x00b")                   // embedded null

	f.Fuzz(func(t *testing.T, s string) {
		buf := NewBuffer(40, 10)
		clip := geom.Rect{X: 0, Y: 0, W: 40, H: 10}
		p := NewPainter(buf, clip, style.Style{})

		// Should not panic.
		p.Text(0, 0, s, style.Style{})

		// Verify no cell was written outside buffer bounds.
		for y := range buf.H {
			for x := range buf.W {
				cell := buf.At(x, y)
				if cell.Wide && x+1 < buf.W {
					cont := buf.At(x+1, y)
					if !cont.WideCont {
						t.Errorf("wide cell at (%d,%d) without continuation", x, y)
					}
				}
			}
		}
	})
}
