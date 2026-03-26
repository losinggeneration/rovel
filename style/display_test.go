package style

import "testing"

func TestColorDisplayRGBA(t *testing.T) {
	fallback := RGBA{R: 1, G: 2, B: 3, A: 4}

	if got := ColorDefault.DisplayRGBA(fallback); got != fallback {
		t.Fatalf("default DisplayRGBA = %+v, want %+v", got, fallback)
	}

	got := ColorRed.DisplayRGBA(fallback)
	if got.A != 0xFF || got.R == 0 && got.G == 0 && got.B == 0 {
		t.Fatalf("basic DisplayRGBA = %+v, want opaque non-zero red-ish color", got)
	}

	got = ColorRGB(9, 8, 7).DisplayRGBA(fallback)
	if got != (RGBA{R: 9, G: 8, B: 7, A: 0xFF}) {
		t.Fatalf("rgb DisplayRGBA = %+v, want {9 8 7 255}", got)
	}
}

func TestStyleResolvedForDisplay(t *testing.T) {
	base := Style{
		FG: ColorWhite,
		BG: ColorBlue,
	}

	fg, bg := (Style{}).ResolvedForDisplay(base)
	if fg != ColorWhite || bg != ColorBlue {
		t.Fatalf("zero style resolved colors = %v/%v, want base fg/bg", fg, bg)
	}

	fg, bg = (Style{FG: ColorRed, Attr: AttrReverse}).ResolvedForDisplay(base)
	if fg != ColorBlue || bg != ColorRed {
		t.Fatalf("reverse resolved colors = %v/%v, want bg/base swap", fg, bg)
	}
}

func TestStyleDisplayRGBA(t *testing.T) {
	base := Style{
		FG: ColorWhite,
		BG: ColorBlue,
	}

	defaultFG := RGBA{R: 1, G: 1, B: 1, A: 0xFF}
	defaultBG := RGBA{R: 2, G: 2, B: 2, A: 0xFF}

	fg, bg := (Style{FG: ColorRed}).DisplayRGBA(base, defaultFG, defaultBG)
	if fg == defaultFG {
		t.Fatalf("fg fell back to default unexpectedly: %+v", fg)
	}
	if bg == defaultBG {
		t.Fatalf("bg fell back to default unexpectedly: %+v", bg)
	}

	fg, bg = (Style{Attr: AttrReverse}).DisplayRGBA(base, defaultFG, defaultBG)
	wantFG := base.BG.DisplayRGBA(defaultFG)
	wantBG := base.FG.DisplayRGBA(defaultBG)
	if fg != wantFG || bg != wantBG {
		t.Fatalf("reverse DisplayRGBA = %+v/%+v, want %+v/%+v", fg, bg, wantFG, wantBG)
	}
}
