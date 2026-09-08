package rovel

import (
	"testing"

	"github.com/losinggeneration/rovel/backend/memory"
	"github.com/losinggeneration/rovel/geom"
)

func collectFocusChanges(app *App) *[]FocusChange {
	changes := &[]FocusChange{}
	app.SetFocusObserver(func(change FocusChange) {
		*changes = append(*changes, change)
	})
	return changes
}

func TestFocusObserver_ReportsExplicitFocusTransitions(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}
	changes := collectFocusChanges(app)

	first := newMockNode(true)
	second := newMockNode(true)
	app.root = newMockContainer(first, second)
	app.rebuildTree()

	app.Focus(first.ID())
	if len(*changes) != 1 || (*changes)[0] != (FocusChange{From: 0, To: first.ID()}) {
		t.Fatalf("changes = %v, want one transition from 0 to first", *changes)
	}

	app.Focus(second.ID())
	if len(*changes) != 2 || (*changes)[1] != (FocusChange{From: first.ID(), To: second.ID()}) {
		t.Fatalf("changes = %v, want second transition first->second", *changes)
	}

	app.Focus(second.ID())
	if len(*changes) != 2 {
		t.Fatalf("no-op focus request reported a transition: %v", *changes)
	}
}

func TestFocusObserver_ReportsStartupAutoFocus(t *testing.T) {
	app, err := New(AppOpts{Backend: memory.New(geom.Size{W: 40, H: 10})})
	if err != nil {
		t.Fatal(err)
	}
	changes := collectFocusChanges(app)
	first := newMockNode(true)
	second := newMockNode(true)
	app.SetRoot(newMockContainer(first, second))

	if err := app.Enable(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Restore() })

	if len(*changes) != 1 {
		t.Fatalf("changes = %v, want exactly the startup auto-focus transition", *changes)
	}
	if got := (*changes)[0]; got.From != 0 || got.To != first.ID() {
		t.Fatalf("auto-focus transition = %v, want 0 -> first focusable", got)
	}
}

func TestFocusObserver_ReportsRepairAfterRemoval(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}
	changes := collectFocusChanges(app)

	removed := newMockNode(true)
	app.root = newMockContainer(removed)
	app.rebuildTree()
	app.Focus(removed.ID())
	if len(*changes) != 1 {
		t.Fatalf("changes = %v, want initial focus transition", *changes)
	}

	replacement := newMockNode(true)
	app.root = newMockContainer(replacement)
	app.rebuildTree()
	app.ensureValidFocus()

	if len(*changes) != 2 {
		t.Fatalf("changes = %v, want a repair transition after removal", *changes)
	}
	if got := (*changes)[1]; got.From != removed.ID() || got.To == removed.ID() {
		t.Fatalf("repair transition = %v, want from removed view to a different target", got)
	}
}

func TestFocusObserver_ReportsFocusClear(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}
	changes := collectFocusChanges(app)

	only := newMockNode(true)
	app.root = newMockContainer(only)
	app.rebuildTree()
	app.Focus(only.ID())

	app.root = newMockContainer(newMockNode(false))
	app.rebuildTree()
	app.ensureValidFocus()

	if len(*changes) != 2 {
		t.Fatalf("changes = %v, want a clear-focus transition", *changes)
	}
	if got := (*changes)[1]; got.From != only.ID() || got.To != 0 {
		t.Fatalf("clear transition = %v, want from focused view to 0", got)
	}
}
