package sdl

import (
	"os"
	"testing"

	"github.com/losinggeneration/tui/backend"
)

func TestNew(t *testing.T) {
	b, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, ok := b.(backend.Backend); !ok {
		t.Fatalf("New() result does not satisfy backend.Backend: %T", b)
	}
}

func TestBackendEnablePresentRestore_DummyVideo(t *testing.T) {
	if err := os.Setenv("SDL_VIDEODRIVER", "dummy"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}

	bi, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	b := bi.(*Backend)
	if _, err := b.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	err = b.PresentCellFrame(backend.CellFrame{
		W: 2,
		H: 1,
		Cells: []backend.FrameCell{
			{R: 'O'},
			{R: 'K'},
		},
	})
	if err != nil {
		t.Fatalf("PresentCellFrame: %v", err)
	}

	if err := b.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
}
