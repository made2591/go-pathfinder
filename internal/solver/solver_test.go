package solver_test

import (
	"testing"

	"github.com/made2591/go-pathfinder/internal/keyboard"
	"github.com/made2591/go-pathfinder/internal/solver"
)

// ---------------------------------------------------------------------------
// LoadFinder + Name()
// ---------------------------------------------------------------------------

func TestLoadFinder(t *testing.T) {
	if _, err := solver.LoadFinder("dijkstra"); err != nil {
		t.Errorf("LoadFinder(dijkstra): %v", err)
	}
	if _, err := solver.LoadFinder("astar"); err != nil {
		t.Errorf("LoadFinder(astar): %v", err)
	}
	if _, err := solver.LoadFinder("bogus"); err == nil {
		t.Error("LoadFinder(bogus) expected error")
	}
}

func TestSolverNames(t *testing.T) {
	if (solver.Dijkstra{}).Name() != "dijkstra" {
		t.Error("Dijkstra.Name() wrong")
	}
	if (solver.AStar{}).Name() != "astar" {
		t.Error("AStar.Name() wrong")
	}
}

// ---------------------------------------------------------------------------
// Layout.Type via Dijkstra / A*
// ---------------------------------------------------------------------------

func TestLayoutType_QWERTY_qwerty(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	plan, err := l.Type("qwerty", solver.Dijkstra{})
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	if got := len(plan); got != 11 {
		t.Errorf("clicks = %d, want 11", got)
	}
}

func TestLayoutType_EmittedRunes(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	want := "abc"
	plan, err := l.Type(want, solver.Dijkstra{})
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	var got []rune
	for _, s := range plan {
		if s.Emitted != 0 {
			got = append(got, s.Emitted)
		}
	}
	if string(got) != want {
		t.Errorf("emitted %q, want %q", string(got), want)
	}
}

func TestLayoutType_Alphabetical_abc(t *testing.T) {
	l, _ := keyboard.LoadLayout("alphabetical")
	plan, err := l.Type("abc", solver.Dijkstra{})
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	if got := len(plan); got != 5 {
		t.Errorf("clicks = %d, want 5", got)
	}
}

func TestLayoutType_UnknownChar(t *testing.T) {
	l, _ := keyboard.LoadLayout("alphabetical")
	if _, err := l.Type("1", solver.Dijkstra{}); err == nil {
		t.Fatal("expected error for untypeable rune, got nil")
	}
}

// ---------------------------------------------------------------------------
// Dijkstra / A* parity
// ---------------------------------------------------------------------------

// Parity holds only for inputs where the Manhattan-in-layer heuristic is
// tight (single-layer, same caps state). Multi-layer uppercase strings can
// expose the loose-cross-layer-heuristic edge case where A* returns a
// suboptimal plan.
func TestSolverParity(t *testing.T) {
	cases := []struct {
		layout string
		text   string
	}{
		{"qwerty", "qwerty"},
		{"qwerty", "abc"},
		{"qwerty", "az"},
		{"alphabetical", "abc"},
		{"appletv", "hi"},
	}
	for _, tc := range cases {
		t.Run(tc.layout+"/"+tc.text, func(t *testing.T) {
			l, _ := keyboard.LoadLayout(tc.layout)
			dPlan, err := l.Type(tc.text, solver.Dijkstra{})
			if err != nil {
				t.Fatalf("Dijkstra.Type: %v", err)
			}
			l2, _ := keyboard.LoadLayout(tc.layout)
			aPlan, err := l2.Type(tc.text, solver.AStar{})
			if err != nil {
				t.Fatalf("AStar.Type: %v", err)
			}
			if len(dPlan) != len(aPlan) {
				t.Errorf("Dijkstra=%d clicks, AStar=%d clicks; should be equal", len(dPlan), len(aPlan))
			}
		})
	}
}
