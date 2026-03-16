package tui_test

import (
	"testing"

	tui "github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/ui"
)

func TestActionCancelSync(t *testing.T) {
	if tui.ActionCancelForTest != int(ui.ActionCancel) {
		t.Fatalf("actionCancel = %d, ui.ActionCancel = %d — keep in sync",
			tui.ActionCancelForTest, int(ui.ActionCancel))
	}
}
