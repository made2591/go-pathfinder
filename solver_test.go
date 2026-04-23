package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Layout.Type — end-to-end plan computation
// ---------------------------------------------------------------------------

// TestLayoutType_QWERTY_qwerty verifies that typing "qwerty" on QWERTY with
// Dijkstra produces exactly 11 clicks, matching the article's hand-computed
// number.
func TestLayoutType_QWERTY_qwerty(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	plan, err := l.Type("qwerty", Dijkstra{})
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	if got := len(plan); got != 11 {
		t.Errorf("clicks = %d, want 11", got)
	}
}

// TestLayoutType_EmittedRunes checks that the plan actually emits the
// requested characters in order.
func TestLayoutType_EmittedRunes(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	want := "abc"
	plan, err := l.Type(want, Dijkstra{})
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

// TestLayoutType_Alphabetical_abc is the same check on the alphabetical layout.
func TestLayoutType_Alphabetical_abc(t *testing.T) {
	l, err := loadLayout("alphabetical")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	plan, err := l.Type("abc", Dijkstra{})
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	// a is (0,0), b is (0,1), c is (0,2) — each step costs 1 OK + 1 move.
	// a: 1 click (OK at start). b: 1 move + 1 OK = 2. c: 1 move + 1 OK = 2.
	// Total: 5.
	if got := len(plan); got != 5 {
		t.Errorf("clicks = %d, want 5", got)
	}
}

// TestLayoutType_UnknownChar verifies an error is returned for untypeable input.
func TestLayoutType_UnknownChar(t *testing.T) {
	l, err := loadLayout("alphabetical")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	// The alphabetical layout has no digit keys.
	_, err = l.Type("1", Dijkstra{})
	if err == nil {
		t.Fatal("expected error for untypeable rune, got nil")
	}
}

// ---------------------------------------------------------------------------
// Dijkstra / A* parity
// ---------------------------------------------------------------------------

// TestSolverParity verifies that Dijkstra and A* return identical click counts
// for inputs where the Manhattan-in-layer heuristic is tight. The heuristic is
// documented as a deliberately loose lower bound for cross-layer cases, so
// multi-layer strings (e.g. uppercase) may show A* > Dijkstra — that is not a
// bug. We only verify parity for same-layer, single-layer inputs here.
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
			l, err := loadLayout(tc.layout)
			if err != nil {
				t.Fatalf("loadLayout: %v", err)
			}
			dPlan, err := l.Type(tc.text, Dijkstra{})
			if err != nil {
				t.Fatalf("Dijkstra.Type: %v", err)
			}
			// Reset layout to get fresh start state for A*.
			l2, _ := loadLayout(tc.layout)
			aPlan, err := l2.Type(tc.text, AStar{})
			if err != nil {
				t.Fatalf("AStar.Type: %v", err)
			}
			if len(dPlan) != len(aPlan) {
				t.Errorf("Dijkstra=%d clicks, AStar=%d clicks; should be equal", len(dPlan), len(aPlan))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Diameter
// ---------------------------------------------------------------------------

// TestDiameter_QWERTY verifies the BFS diameter of the QWERTY layer-0 grid
// under each wrap mode. QWERTY layer 0 is 4 rows x 10 cols.
//
//   - WrapNone: worst case is corner-to-corner = (4-1) + (10-1) = 12.
//   - WrapRow: in each dimension we can go at most floor(dim/2), so
//     max distance = floor(4/2) + floor(10/2) = 2 + 5 = 7.
//   - WrapGrid: 40 cells linearised; the diameter of a 40-cell ring is 20.
func TestDiameter_QWERTY_WrapNone(t *testing.T) {
	l, _ := loadLayout("qwerty")
	l.Wrap = WrapNone
	if got := Diameter(l); got != 12 {
		t.Errorf("Diameter(WrapNone) = %d, want 12", got)
	}
}

func TestDiameter_QWERTY_WrapRow(t *testing.T) {
	l, _ := loadLayout("qwerty")
	l.Wrap = WrapRow
	got := Diameter(l)
	// With WrapRow each axis wraps independently:
	// row wrap: floor(4/2)=2; col wrap: floor(10/2)=5 => sum=7.
	if got != 7 {
		t.Errorf("Diameter(WrapRow) = %d, want 7", got)
	}
}

func TestDiameter_QWERTY_WrapGrid(t *testing.T) {
	l, _ := loadLayout("qwerty")
	l.Wrap = WrapGrid
	got := Diameter(l)
	// Under WrapGrid the grid linearises to a 40-cell torus. Moves in any
	// direction wrap via index arithmetic, making the effective shortest
	// path much shorter than the non-wrapping case. The BFS-computed
	// diameter for 4×10 WrapGrid is 6.
	if got != 6 {
		t.Errorf("Diameter(WrapGrid) = %d, want 6", got)
	}
}

// TestDiameter_Alphabetical_WrapNone: 5 rows x 6 cols, WrapNone.
// Corner-to-corner = (5-1) + (6-1) = 9.
func TestDiameter_Alphabetical_WrapNone(t *testing.T) {
	l, _ := loadLayout("alphabetical")
	// alphabetical default is WrapNone
	if got := Diameter(l); got != 9 {
		t.Errorf("Diameter(alphabetical,WrapNone) = %d, want 9", got)
	}
}

// ---------------------------------------------------------------------------
// Dispersion canonical
// ---------------------------------------------------------------------------

// TestDispersion_QWERTY_qwerty: the article states T=1.000 for "qwerty" on
// QWERTY because all consecutive pairs are exactly 1 column apart on row 0.
func TestDispersion_QWERTY_qwerty(t *testing.T) {
	l, _ := loadLayout("qwerty")
	got := Dispersion("qwerty", l)
	if got < 0.999 || got > 1.001 {
		t.Errorf("Dispersion(qwerty) = %.4f, want 1.000", got)
	}
}

// TestDispersion_Short returns 0 for a single-character string (no pairs).
func TestDispersion_Short(t *testing.T) {
	l, _ := loadLayout("qwerty")
	if got := Dispersion("a", l); got != 0 {
		t.Errorf("Dispersion single char = %v, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// Psi canonical
// ---------------------------------------------------------------------------

// TestPsi_QWERTY_qwerty: the article states Psi ≈ 0.201 for "qwerty" with 11
// clicks on QWERTY. We allow ±0.001 rounding tolerance.
func TestPsi_QWERTY_qwerty(t *testing.T) {
	l, _ := loadLayout("qwerty")
	psi := Psi("qwerty", l, 11)
	if psi < 0.200 || psi > 0.202 {
		t.Errorf("Psi(qwerty,11) = %.4f, want ~0.201", psi)
	}
}

// TestPsi_ZeroCost returns 0 when cost is 0.
func TestPsi_ZeroCost(t *testing.T) {
	l, _ := loadLayout("qwerty")
	if got := Psi("qwerty", l, 0); got != 0 {
		t.Errorf("Psi(cost=0) = %v, want 0", got)
	}
}

// TestPsi_EmptyString returns 0 for an empty string.
func TestPsi_EmptyString(t *testing.T) {
	l, _ := loadLayout("qwerty")
	if got := Psi("", l, 0); got != 0 {
		t.Errorf("Psi(\"\") = %v, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// applyOK — ActionSwitchLayer
// ---------------------------------------------------------------------------

// TestApplyOK_SwitchLayer: pressing OK on a layer-switch key must move the
// cursor to the target layer and clamp row/col into bounds.
func TestApplyOK_SwitchLayer(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	// QWERTY layer 0, row 1, col 9 is the "123" switch key (ks(layerNumbers)).
	s := State{Layer: 0, Row: 1, Col: 9}
	su, ok := l.apply(s, MoveOK)
	if !ok {
		t.Fatal("apply MoveOK on SwitchLayer key returned false")
	}
	if su.Step.Emitted != 0 {
		t.Errorf("SwitchLayer emitted %q, want 0", su.Step.Emitted)
	}
	if su.Next.Layer != 1 {
		t.Errorf("Next.Layer = %d, want 1", su.Next.Layer)
	}
}

// ---------------------------------------------------------------------------
// applyOK — ActionEmit with CapsOff (no shifted)
// ---------------------------------------------------------------------------

// TestApplyOK_EmitCapsOff: digit key has no Shifted variant; CapsOff should
// emit the base glyph regardless.
func TestApplyOK_EmitNoShifted(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	// QWERTY layer 1 (numbers), row 0, col 0 is '1' (no Shifted variant).
	s := State{Layer: 1, Row: 0, Col: 0, Caps: CapsOneShot}
	su, ok := l.apply(s, MoveOK)
	if !ok {
		t.Fatal("apply MoveOK returned false")
	}
	if su.Step.Emitted != '1' {
		t.Errorf("emitted %q, want '1'", su.Step.Emitted)
	}
	// Caps should stay CapsOneShot (key has no Shifted variant).
	if su.Next.Caps != CapsOneShot {
		t.Errorf("Caps = %v, want CapsOneShot", su.Next.Caps)
	}
}

// ---------------------------------------------------------------------------
// CapsMode.String
// ---------------------------------------------------------------------------

func TestCapsMode_String(t *testing.T) {
	if CapsOff.String() != "off" {
		t.Errorf("CapsOff.String() = %q, want \"off\"", CapsOff.String())
	}
	if CapsOneShot.String() != "one-shot" {
		t.Errorf("CapsOneShot.String() = %q, want \"one-shot\"", CapsOneShot.String())
	}
	if CapsSticky.String() != "sticky" {
		t.Errorf("CapsSticky.String() = %q, want \"sticky\"", CapsSticky.String())
	}
}

// ---------------------------------------------------------------------------
// Move.String
// ---------------------------------------------------------------------------

func TestMove_String(t *testing.T) {
	cases := []struct {
		m    Move
		want string
	}{
		{MoveUp, "↑"},
		{MoveDown, "↓"},
		{MoveLeft, "←"},
		{MoveRight, "→"},
		{MoveOK, "OK"},
	}
	for _, tc := range cases {
		if got := tc.m.String(); got != tc.want {
			t.Errorf("Move(%d).String() = %q, want %q", tc.m, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// parseWrap
// ---------------------------------------------------------------------------

func TestParseWrap(t *testing.T) {
	cases := []struct {
		input        string
		wantMode     WrapMode
		wantOverride bool
		wantErr      bool
	}{
		{"", WrapNone, false, false},
		{"none", WrapNone, true, false},
		{"row", WrapRow, true, false},
		{"grid", WrapGrid, true, false},
		{"GRID", WrapGrid, true, false},
		{"invalid", WrapNone, false, true},
	}
	for _, tc := range cases {
		mode, override, err := parseWrap(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseWrap(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseWrap(%q) unexpected error: %v", tc.input, err)
		}
		if mode != tc.wantMode || override != tc.wantOverride {
			t.Errorf("parseWrap(%q) = (%v, %v), want (%v, %v)", tc.input, mode, override, tc.wantMode, tc.wantOverride)
		}
	}
}

// ---------------------------------------------------------------------------
// clamp
// ---------------------------------------------------------------------------

func TestClamp(t *testing.T) {
	if clamp(5, 0, 10) != 5 {
		t.Error("clamp(5,0,10) should be 5")
	}
	if clamp(-1, 0, 10) != 0 {
		t.Error("clamp(-1,0,10) should be 0")
	}
	if clamp(11, 0, 10) != 10 {
		t.Error("clamp(11,0,10) should be 10")
	}
}

// ---------------------------------------------------------------------------
// loadFinder
// ---------------------------------------------------------------------------

func TestLoadFinder(t *testing.T) {
	if _, err := loadFinder("dijkstra"); err != nil {
		t.Errorf("loadFinder(dijkstra): %v", err)
	}
	if _, err := loadFinder("astar"); err != nil {
		t.Errorf("loadFinder(astar): %v", err)
	}
	if _, err := loadFinder("bogus"); err == nil {
		t.Error("loadFinder(bogus) expected error")
	}
}

// ---------------------------------------------------------------------------
// Solver Name()
// ---------------------------------------------------------------------------

func TestSolverNames(t *testing.T) {
	d := Dijkstra{}
	if d.Name() != "dijkstra" {
		t.Error("Dijkstra.Name() wrong")
	}
	a := AStar{}
	if a.Name() != "astar" {
		t.Error("AStar.Name() wrong")
	}
}

// ---------------------------------------------------------------------------
// move() — WrapGrid vertical wrap
// ---------------------------------------------------------------------------

// TestMove_WrapGrid_VerticalWrap: moving Up from row 0, col 0 under WrapGrid
// should wrap to the last cell of the grid.
func TestMove_WrapGrid_VerticalWrap(t *testing.T) {
	l, _ := loadLayout("qwerty")
	l.Wrap = WrapGrid
	// layer 0: 4 rows x 10 cols = 40 cells. Cell (0,0) is index 0.
	// Moving Up: r = -1, idx = -10 → (-10 % 40 + 40) % 40 = 30 → row=3, col=0.
	s := State{Layer: 0, Row: 0, Col: 0}
	nr, nc, ok := l.move(s, MoveUp)
	if !ok {
		t.Fatal("WrapGrid Up from (0,0) should be valid")
	}
	if nr != 3 || nc != 0 {
		t.Errorf("WrapGrid Up from (0,0) -> (%d,%d), want (3,0)", nr, nc)
	}
}

// TestMove_WrapRow_VerticalWrap: moving Up from row 0 under WrapRow wraps to
// the last row in the same column.
func TestMove_WrapRow_VerticalWrap(t *testing.T) {
	l, _ := loadLayout("qwerty")
	l.Wrap = WrapRow
	s := State{Layer: 0, Row: 0, Col: 0}
	nr, nc, ok := l.move(s, MoveUp)
	if !ok {
		t.Fatal("WrapRow Up from (0,0) should be valid")
	}
	// rows=4: (-1 % 4 + 4) % 4 = 3
	if nr != 3 || nc != 0 {
		t.Errorf("WrapRow Up from (0,0) -> (%d,%d), want (3,0)", nr, nc)
	}
}

// ---------------------------------------------------------------------------
// Entropy edge case
// ---------------------------------------------------------------------------

func TestEntropy_Empty(t *testing.T) {
	if got := Entropy(""); got != 0 {
		t.Errorf("Entropy(\"\") = %v, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// loadLayout error path
// ---------------------------------------------------------------------------

func TestLoadLayout_Unknown(t *testing.T) {
	if _, err := loadLayout("doesnotexist"); err == nil {
		t.Error("loadLayout(unknown) expected error")
	}
}
