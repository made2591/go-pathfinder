package sim

import (
	"bytes"
	"strings"
	"testing"

	"github.com/made2591/go-pathfinder/internal/keyboard"
	"github.com/made2591/go-pathfinder/internal/solver"
)

func TestBuildPlan_HiExclaim(t *testing.T) {
	layout, _ := keyboard.LoadLayout("qwerty")
	finder, _ := solver.LoadFinder("dijkstra")

	text := "Hi!"
	plan, err := BuildPlan(layout, finder, text)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	var sb strings.Builder
	for _, ps := range plan {
		if ps.Step.Emitted != 0 {
			sb.WriteRune(ps.Step.Emitted)
		}
	}
	if got := sb.String(); got != text {
		t.Errorf("typed %q, want %q", got, text)
	}
	if len(plan) == 0 {
		t.Error("plan is empty")
	}
	for i, ps := range plan {
		if ps.After.Layer < 0 || ps.After.Layer >= len(layout.Layers) {
			t.Errorf("step %d: After.Layer %d out of range", i, ps.After.Layer)
		}
		layer := &layout.Layers[ps.After.Layer]
		if ps.After.Row < 0 || ps.After.Row >= layer.Rows() {
			t.Errorf("step %d: After.Row %d out of range", i, ps.After.Row)
		}
		if ps.After.Col < 0 || ps.After.Col >= layer.Cols() {
			t.Errorf("step %d: After.Col %d out of range", i, ps.After.Col)
		}
	}
}

func TestBuildPlan_ZeroSpeed(t *testing.T) {
	layout, _ := keyboard.LoadLayout("qwerty")
	finder, _ := solver.LoadFinder("dijkstra")

	for _, text := range []string{"Hi!", "abc", "Hello"} {
		plan, err := BuildPlan(layout, finder, text)
		if err != nil {
			t.Fatalf("text=%q BuildPlan: %v", text, err)
		}
		var sb strings.Builder
		for _, ps := range plan {
			if ps.Step.Emitted != 0 {
				sb.WriteRune(ps.Step.Emitted)
			}
		}
		if got := sb.String(); got != text {
			t.Errorf("text=%q got typed=%q", text, got)
		}
	}
}

func TestRenderSimLayer_NoPanic(t *testing.T) {
	layout, _ := keyboard.LoadLayout("qwerty")
	layer := &layout.Layers[0]

	for _, tc := range []struct{ row, col int }{{0, 0}, {layer.Rows() - 1, layer.Cols() - 1}, {1, 5}} {
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

func TestRenderSimLayer_FlashModes(t *testing.T) {
	layout, _ := keyboard.LoadLayout("qwerty")
	layer := &layout.Layers[0]

	for _, fm := range []flashMode{flashNone, flashEmit, flashCaps, flashLayer} {
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

func TestRenderStatus_Fields(t *testing.T) {
	layout, _ := keyboard.LoadLayout("qwerty")
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
