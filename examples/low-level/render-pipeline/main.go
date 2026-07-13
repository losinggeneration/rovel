// Package main implements a pipeline demo that tests the rendering infrastructure.
// This demo shows the render + diff + flush pipeline with incremental updates.
//
// NOTE: This is a low-level demo that directly uses the render package to test
// the base infrastructure. With widgets, they will use ctx.Invalidate() to mark
// dirty regions, and Paint Contract A will automatically clear damaged areas
// before painting.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/render"
)

func main() {
	// Create a buffer for our "screen"
	buf := render.NewBuffer(80, 24)

	// Create a front buffer to track what's been rendered
	front := render.NewBuffer(80, 24)

	// Create a damage tracker
	damage := render.NewDamage(80, 24)

	// Create a clip rect for the full buffer
	clip := rovel.Rect{X: 0, Y: 0, W: 80, H: 24}

	// Create a painter (no damage tracking — damage is explicit)
	baseStyle := rovel.Style{FG: rovel.ColorDefault, BG: rovel.ColorDefault}
	painter := render.NewPainter(buf, clip, baseStyle)

	// Create an ANSI flusher that writes to stdout
	flusher := render.NewANSIFlusher(os.Stdout)

	// Clear screen and hide cursor
	err := flusher.ClearScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ClearScreen error: %v\n", err)
	}

	err = flusher.HideCursor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "HideCursor error: %v\n", err)
	}

	defer func() {
		err := flusher.ShowCursor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ShowCursor error: %v\n", err)
		}
	}()

	err = flusher.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Flush error: %v\n", err)
	}

	// Frame 1: Header box with "loading" message
	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	// Frame 2 & 3: Add the rendering pipeline box and diff engine box
	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	drawPipelineBox(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	drawPipelineBox(painter)
	drawDiffBox(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	// Frame 4 & Wide chars: Add the filled box and wide character test
	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	drawPipelineBox(painter)
	drawDiffBox(painter)
	drawFilledBox(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	drawPipelineBox(painter)
	drawDiffBox(painter)
	drawFilledBox(painter)
	drawWideChars(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	// Update frame 3 (diff box): Add text and change size
	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "Loading...")
	drawPipelineBox(painter)
	drawDiffBoxUpdated(painter)
	drawFilledBox(painter)
	drawWideChars(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	// Update frames 1 & 4: Update header to DONE, add text to filled box
	damage.Clear()
	damage.AddRect(clip)

	clearText := rovel.Rect{X: 4, Y: 3, W: 14, H: 1}
	painter.Fill(clearText, ' ', whiteBlack)
	drawHeaderBox(painter, "DONE!")
	drawPipelineBox(painter)
	drawDiffBoxUpdated(painter)
	drawFilledBoxWithText(painter)
	drawWideChars(painter)
	flushAndShowDiff(buf, front, damage, flusher)
	time.Sleep(750 * time.Millisecond)

	// Add "Press Enter to exit" text
	damage.Clear()
	damage.AddRect(clip)
	drawHeaderBox(painter, "DONE!")
	drawPipelineBox(painter)
	drawDiffBoxUpdated(painter)
	drawFilledBoxWithText(painter)
	drawWideChars(painter)
	drawExitText(painter)
	flushAndShowDiff(buf, front, damage, flusher)

	// Wait for user input
	_, _ = fmt.Scanln()
}

func flushAndShowDiff(buf, front *render.Buffer, damage *render.Damage, flusher *render.ANSIFlusher) {
	runs := render.DiffRuns(buf, front, damage)

	err := flusher.FlushRuns(buf, front, runs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Flush error: %v\n", err)
		os.Exit(1)
	}
}

// Styles
var (
	cyanBold    = rovel.Style{FG: rovel.ColorCyan, BG: rovel.ColorBlack, Attr: rovel.AttrBold}
	yellowBlue  = rovel.Style{FG: rovel.ColorYellow, BG: rovel.ColorBlue, Attr: rovel.AttrBold}
	greenNormal = rovel.Style{FG: rovel.ColorGreen, BG: rovel.ColorBlack}
	magentaNorm = rovel.Style{FG: rovel.ColorMagenta, BG: rovel.ColorBlack}
	brightMag   = rovel.Style{FG: rovel.ColorBrightMagenta, BG: rovel.ColorBlack, Attr: rovel.AttrBold}
	blackGreen  = rovel.Style{FG: rovel.ColorBlack, BG: rovel.ColorGreen, Attr: rovel.AttrBold}
	whiteBlack  = rovel.Style{FG: rovel.ColorWhite, BG: rovel.ColorBlack}
	brightGreen = rovel.Style{FG: rovel.ColorBrightGreen, BG: rovel.ColorBlack, Attr: rovel.AttrBold}
	brightWhite = rovel.Style{FG: rovel.ColorBrightWhite, BG: rovel.ColorBlack}
	brightBlack = rovel.Style{FG: rovel.ColorBrightBlack, BG: rovel.ColorBlack}
	grayBg      = rovel.Style{FG: rovel.ColorWhite, BG: rovel.ColorBrightBlack}
)

func drawHeaderBox(p *render.Painter, title string) {
	box := rovel.Rect{X: 2, Y: 2, W: 40, H: 5}
	p.Box(box, cyanBold)
	p.Text(4, 3, "  "+title+"  ", yellowBlue)
}

func drawPipelineBox(p *render.Painter) {
	box := rovel.Rect{X: 2, Y: 10, W: 30, H: 8}
	p.Box(box, greenNormal)
	p.Text(4, 11, "The rendering pipeline", whiteBlack)
	p.Text(4, 12, "is working!", brightGreen)
}

func drawDiffBox(p *render.Painter) {
	box := rovel.Rect{X: 45, Y: 10, W: 30, H: 8}
	p.Box(box, magentaNorm)
	p.Text(47, 11, "Diff engine:", brightMag)
}

func drawDiffBoxUpdated(p *render.Painter) {
	box := rovel.Rect{X: 45, Y: 10, W: 30, H: 8}
	p.Box(box, magentaNorm)
	p.Text(47, 11, "Diff engine:", brightMag)
	p.Text(47, 13, "  [ OK ]  ", blackGreen)
}

func drawFilledBox(p *render.Painter) {
	box := rovel.Rect{X: 50, Y: 2, W: 25, H: 5}
	p.Fill(box, '░', brightBlack)
}

func drawFilledBoxWithText(p *render.Painter) {
	box := rovel.Rect{X: 50, Y: 2, W: 25, H: 5}
	p.Fill(box, '░', brightBlack)
	p.Text(52, 3, "Pattern fill", grayBg)
}

func drawWideChars(p *render.Painter) {
	p.Text(4, 20, "Wide chars: 你好世界 🌍", brightWhite)
}

func drawExitText(p *render.Painter) {
	p.Text(2, 23, "Press Enter to exit...", whiteBlack)
}
