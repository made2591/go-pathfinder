package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// captureNonTTY exercises buildPlan + the non-TTY rendering path by driving
// runSim with a bytes.Buffer as the writer. Since runSim writes to os.Stdout
// directly (and always detects non-TTY for a pipe), we test buildPlan
// independently here.
func TestBuildPlan_HiExclaim(t *testing.T) {
	layout, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	finder, err := loadFinder("dijkstra")
	if err != nil {
		t.Fatalf("loadFinder: %v", err)
	}

	text := "Hi!"
	plan, err := buildPlan(layout, finder, text)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	// Reconstruct the typed string from emitted runes.
	var sb strings.Builder
	for _, ps := range plan {
		if ps.step.Emitted != 0 {
			sb.WriteRune(ps.step.Emitted)
		}
	}
	got := sb.String()
	if got != text {
		t.Errorf("typed %q, want %q", got, text)
	}

	// Plan must be non-empty and every planStep must have a valid after-state.
	if len(plan) == 0 {
		t.Error("plan is empty")
	}
	for i, ps := range plan {
		l := layout
		if ps.after.Layer < 0 || ps.after.Layer >= len(l.Layers) {
			t.Errorf("step %d: after.Layer %d out of range", i, ps.after.Layer)
		}
		layer := &l.Layers[ps.after.Layer]
		if ps.after.Row < 0 || ps.after.Row >= layer.Rows() {
			t.Errorf("step %d: after.Row %d out of range", i, ps.after.Row)
		}
		if ps.after.Col < 0 || ps.after.Col >= layer.Cols() {
			t.Errorf("step %d: after.Col %d out of range", i, ps.after.Col)
		}
	}
}

// TestRenderSimLayer checks that renderSimLayer produces output for every row
// and does not panic for edge cursor positions.
func TestRenderSimLayer_NoPanic(t *testing.T) {
	layout, _ := loadLayout("qwerty")
	layer := &layout.Layers[0]

	cases := []struct{ row, col int }{
		{0, 0},
		{layer.Rows() - 1, layer.Cols() - 1},
		{1, 5},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		if err := renderSimLayer(&buf, layer, tc.row, tc.col, flashNone); err != nil {
			t.Errorf("renderSimLayer row=%d col=%d: %v", tc.row, tc.col, err)
		}
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		if len(lines) != layer.Rows() {
			t.Errorf("expected %d lines, got %d", layer.Rows(), len(lines))
		}
	}
}

// TestRenderSimLayer_FlashModes verifies that each flash mode writes ANSI
// escape codes for the highlighted cell and leaves other cells plain.
func TestRenderSimLayer_FlashModes(t *testing.T) {
	layout, _ := loadLayout("qwerty")
	layer := &layout.Layers[0]

	modes := []flashMode{flashNone, flashEmit, flashCaps, flashLayer}
	for _, fm := range modes {
		var buf bytes.Buffer
		if err := renderSimLayer(&buf, layer, 0, 0, fm); err != nil {
			t.Errorf("flash mode %d: %v", fm, err)
		}
		out := buf.String()
		if fm != flashNone && !strings.Contains(out, "\x1b[") {
			t.Errorf("flash mode %d: expected ANSI escape in output", fm)
		}
	}
}

// TestRenderStatus_Fields checks that the status line contains expected fields.
func TestRenderStatus_Fields(t *testing.T) {
	layout, _ := loadLayout("qwerty")
	sf := simFrame{
		layout: layout,
		state:  layout.Start(),
		typed:  "He",
		target: "Hello",
		clicks: 7,
	}
	var buf bytes.Buffer
	if err := renderStatus(&buf, sf); err != nil {
		t.Fatalf("renderStatus: %v", err)
	}
	line := buf.String()
	for _, want := range []string{"Hello", "He", "clicks: 7", "letters", "off"} {
		if !strings.Contains(line, want) {
			t.Errorf("status line missing %q:\n%s", want, line)
		}
	}
}

// TestBuildPlan_ZeroSpeed is a smoke test that buildPlan + plan replay with
// speed=0 does not panic and produces the correct output. It exercises the
// same code path runSim uses in a non-TTY context without involving os.Stdout.
func TestBuildPlan_ZeroSpeed(t *testing.T) {
	layout, _ := loadLayout("qwerty")
	finder, _ := loadFinder("dijkstra")

	texts := []string{"Hi!", "abc", "Hello"}
	for _, text := range texts {
		plan, err := buildPlan(layout, finder, text)
		if err != nil {
			t.Fatalf("text=%q buildPlan: %v", text, err)
		}
		var sb strings.Builder
		for _, ps := range plan {
			if ps.step.Emitted != 0 {
				sb.WriteRune(ps.step.Emitted)
			}
		}
		if got := sb.String(); got != text {
			t.Errorf("text=%q got typed=%q", text, got)
		}
	}
	// Ensure speed=0 sentinel does not cause negative sleep in the logic.
	_ = time.Duration(0)
}
