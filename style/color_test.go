package style

import "testing"

func TestColorConstructors(t *testing.T) {
	// Test ColorBasic
	c := ColorBasic(1) // Red
	if kind, _ := c.BasicIndex(); kind != 1 {
		t.Errorf("ColorBasic(1) BasicIndex() = %d, want 1", kind)
	}

	if c.Kind() != ColorKindBasic {
		t.Errorf("ColorBasic(1) Kind() = %d, want %d", c.Kind(), ColorKindBasic)
	}

	// Test ColorIndex
	idx := ColorIndex(232)
	if i, _ := idx.Index(); i != 232 {
		t.Errorf("ColorIndex(232) Index() = %d, want 232", i)
	}

	if idx.Kind() != ColorKindIndexed {
		t.Errorf("ColorIndex(232) Kind() = %d, want %d", idx.Kind(), ColorKindIndexed)
	}

	// Test ColorRGB
	rgb := ColorRGB(255, 128, 0)

	r, g, b, ok := rgb.RGB()
	if !ok || r != 255 || g != 128 || b != 0 {
		t.Errorf("ColorRGB(255,128,0) RGB() = %d,%d,%d,%v, want 255,128,0,true", r, g, b, ok)
	}

	if rgb.Kind() != ColorKindRGB {
		t.Errorf("ColorRGB(255,128,0) Kind() = %d, want %d", rgb.Kind(), ColorKindRGB)
	}
}

func TestColorResolve_Basic(t *testing.T) {
	c := Capability{HasBasic: true}

	// Basic colors should pass through unchanged
	color := ColorRed

	resolved := color.Resolve(c)
	if resolved != color {
		t.Errorf("Basic color changed after Resolve: got %d, want %d", resolved, color)
	}

	// Default should pass through
	d := ColorDefault

	resolved = d.Resolve(c)
	if resolved != d {
		t.Errorf("Default color changed after Resolve")
	}
}

func TestColorResolve_RGBToIndexed(t *testing.T) {
	c := Capability{HasBasic: true, Has256Color: true}

	// RGB should map to nearest indexed when no truecolor
	color := ColorRGB(255, 0, 0)
	resolved := color.Resolve(c)

	if resolved.Kind() != ColorKindIndexed {
		t.Errorf("RGB didn't resolve to indexed, got kind=%d", resolved.Kind())
	}

	// Pure red should map to a red-ish indexed color
	idx, ok := resolved.Index()
	if !ok {
		t.Errorf("failed to get index from resolved color")
	}

	_ = idx
}

func TestColorResolve_RGBToBasic(t *testing.T) {
	c := Capability{HasBasic: true}

	// RGB should map to basic when no 256/truecolor
	color := ColorRGB(255, 0, 0)
	resolved := color.Resolve(c)

	if resolved.Kind() != ColorKindBasic {
		t.Errorf("RGB didn't resolve to basic, got kind=%d", resolved.Kind())
	}

	// Red RGB should map to ColorRed or similar
	if resolved != ColorRed && resolved != ColorBrightRed {
		// This is OK as long as it's some kind of red
		idx, _ := resolved.BasicIndex()
		if idx != 1 && idx != 9 {
			t.Errorf("RGB(255,0,0) resolved to unexpected basic color: %d", idx)
		}
	}
}

func TestColorResolve_IndexedToBasic(t *testing.T) {
	c := Capability{HasBasic: true}

	// Indexed should map to basic when no 256-color support
	color := ColorIndex(200) // Some color in the 256-color palette
	resolved := color.Resolve(c)

	if resolved.Kind() != ColorKindBasic {
		t.Errorf("Indexed didn't resolve to basic, got kind=%d", resolved.Kind())
	}
}

func TestColorConstants(t *testing.T) {
	// Verify constants produce correct kind/index
	tests := []struct {
		color    Color
		wantKind ColorKind
		wantIdx  uint8
	}{
		{ColorBlack, ColorKindBasic, 0},
		{ColorRed, ColorKindBasic, 1},
		{ColorBrightWhite, ColorKindBasic, 15},
	}

	for _, tt := range tests {
		kind := tt.color.Kind()
		if kind != tt.wantKind {
			t.Errorf("%v.Kind() = %d, want %d", tt.color, kind, tt.wantKind)
		}

		if idx, ok := tt.color.BasicIndex(); !ok || idx != tt.wantIdx {
			t.Errorf("%v.BasicIndex() = %d,%v, want %d,true", tt.color, idx, ok, tt.wantIdx)
		}
	}
}

func TestColorResolve_NoColorFloor(t *testing.T) {
	c := Capability{} // no basic color

	// Every color kind degrades to the terminal default.
	for _, color := range []Color{
		ColorDefault,
		ColorRed,
		ColorIndex(200),
		ColorRGB(255, 0, 0),
	} {
		if resolved := color.Resolve(c); resolved != ColorDefault {
			t.Errorf("%v.Resolve(no-color) = %v, want ColorDefault", color, resolved)
		}
	}
}

func TestDetectCapabilityFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		noColor   *string // nil = unset
		term      string
		colorterm string
		want      Capability
	}{
		{
			name: "plain terminal has basic only",
			term: "xterm",
			want: Capability{HasBasic: true},
		},
		{
			name: "256color term",
			term: "xterm-256color",
			want: Capability{HasBasic: true, Has256Color: true},
		},
		{
			name:      "truecolor via COLORTERM",
			term:      "xterm",
			colorterm: "truecolor",
			want:      Capability{HasBasic: true, Has256Color: true, HasTrueColor: true},
		},
		{
			name:    "NO_COLOR disables everything",
			noColor: strptr("1"),
			term:    "xterm-256color",
			want:    Capability{},
		},
		{
			name:      "NO_COLOR wins over COLORTERM=truecolor",
			noColor:   strptr("anything"),
			colorterm: "truecolor",
			want:      Capability{},
		},
		{
			name:    "empty NO_COLOR does not disable",
			noColor: strptr(""),
			term:    "xterm-256color",
			want:    Capability{HasBasic: true, Has256Color: true},
		},
		{
			name: "TERM=dumb disables everything",
			term: "dumb",
			want: Capability{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noColor := ""
			if tt.noColor != nil {
				noColor = *tt.noColor
			}

			// Empty NO_COLOR is treated as unset by DetectCapabilityFromEnv,
			// so setting "" here is equivalent to the variable being absent.
			t.Setenv("NO_COLOR", noColor)
			t.Setenv("TERM", tt.term)
			t.Setenv("COLORTERM", tt.colorterm)

			if got := DetectCapabilityFromEnv(); got != tt.want {
				t.Errorf("DetectCapabilityFromEnv() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func strptr(s string) *string { return &s }

func TestStyleDerivation(t *testing.T) {
	base := Style{
		FG:   ColorDefault,
		BG:   ColorDefault,
		Attr: 0,
	}

	// Test WithFG
	withFG := base.WithFG(ColorRed)
	if withFG.FG != ColorRed {
		t.Errorf("WithFG() failed: got %v, want FG=Red", withFG)
	}

	if withFG.BG != base.BG {
		t.Errorf("WithFG() changed BG")
	}

	// Test WithAttr
	withAttr := base.WithAttr(AttrBold)
	if withAttr.Attr != AttrBold {
		t.Errorf("WithAttr() failed: got %v, want Bold", withAttr)
	}

	// Test Merge
	override := Style{FG: ColorBlue, Attr: AttrUnderline}

	merged := Merge(base, override)
	if merged.FG != ColorBlue {
		t.Errorf("Merge() didn't apply FG override")
	}

	if merged.Attr != (base.Attr | override.Attr) {
		t.Errorf("Merge() didn't merge attrs")
	}
}
