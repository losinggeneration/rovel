// Example cbreak demonstrates cbreak (character-at-a-time, non-raw) terminal
// mode.
//
// The rendering pipeline works identically to raw mode: styled text, borders,
// and cursor positioning all function normally. The difference is that the
// terminal stays in cbreak (non-canonical) mode, so:
//
//   - Text on screen is selectable with the mouse
//   - Ctrl+C delivers SIGINT and is translated to a KeyCtrlC event
//   - Output is drawn inline at the current cursor position — the screen is
//     not cleared, and content above the region is preserved
//
// This mode is suitable for interactive CLI tools — dialog boxes, prompts,
// and script-driven TUI components (think dialog(1), gum, or Claude Code) —
// that draw inline without taking over the entire terminal.
package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/widgets"
)

// dialogRoot wraps a Dialog and maps Ctrl+C to Quit so the example can be
// dismissed without selecting a button.
type dialogRoot struct {
	*widgets.Dialog
}

func (d *dialogRoot) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if ke, ok := e.(rovel.KeyEvent); ok && ke.Key == event.KeyCtrlC {
		ctx.Quit()

		return true
	}

	return d.Dialog.Handle(e, ctx)
}

func main() {
	os.Exit(run())
}

func run() int {
	appOpts := rovel.AppOpts{
		TerminalMode:  rovel.ModeCBreak,
		ClearOnExit:   true,
		ResolveAction: ui.DefaultAppResolver(),
	}

	app, err := rovel.New(appOpts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	var choice string

	dialog := widgets.NewDialog(widgets.DialogOpts{
		Title:   "Deploy to production?",
		Message: "This will push v2.4.1 to all regions.\nRollback requires a manual intervention.",
		Buttons: []widgets.DialogButton{
			{Label: "Cancel", OnPress: func(ctx *rovel.Ctx) {
				choice = "cancel"
				ctx.Quit()
			}},
			{Label: "Deploy", OnPress: func(ctx *rovel.Ctx) {
				choice = "deploy"
				ctx.Quit()
			}},
		},
		Width: 48,
	})

	app.SetRoot(&dialogRoot{Dialog: dialog})

	if err := app.Enable(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	// Restore the terminal on panic or early return so the user's shell
	// isn't left in cbreak. Guarded so the normal exit path — which
	// restores before writing results — doesn't double-restore.
	restored := false
	defer func() {
		if !restored {
			if err := app.Restore(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)
			}
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	// Restore BEFORE printing so the shell output lands below the rendered
	// region — not on top of the dialog's last painted row.
	if err := app.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)

		return 2
	}
	restored = true

	// Non-zero exit on cancel (or Ctrl+C) so shell flows like
	// `cbreak && next-command` short-circuit when the user declines.
	if choice == "cancel" || choice == "" {
		return 1
	}

	return 0
}
