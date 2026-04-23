package keyboard

import "testing"

// ---------------------------------------------------------------------------
// Layout presets + load
// ---------------------------------------------------------------------------

func TestLoadLayout_Presets(t *testing.T) {
	cases := []struct {
		name      string
		wantGlyph rune
	}{
		{"qwerty", 'q'},
		{"alphabetical", 'a'},
		{"appletv", 'a'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, err := LoadLayout(tc.name)
			if err != nil {
				t.Fatalf("LoadLayout(%q): %v", tc.name, err)
			}
			if len(l.Layers) == 0 {
				t.Fatal("no layers")
			}
			layer0 := &l.Layers[0]
			if layer0.Rows() == 0 || layer0.Cols() == 0 {
				t.Fatal("layer 0 has no keys")
			}
			if got := layer0.Keys[0][0].Glyph; got != tc.wantGlyph {
				t.Errorf("layer0[0][0].Glyph = %q, want %q", got, tc.wantGlyph)
			}
		})
	}
}

func TestLoadLayout_Unknown(t *testing.T) {
	if _, err := LoadLayout("doesnotexist"); err == nil {
		t.Error("LoadLayout(unknown) expected error")
	}
}

// ---------------------------------------------------------------------------
// Wrap policies + edge moves
// ---------------------------------------------------------------------------

func TestWrapPolicies_EdgeMoves(t *testing.T) {
	base, err := LoadLayout("qwerty")
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	lastCol := base.Layers[0].Cols() - 1 // 9

	cases := []struct {
		wrap    WrapMode
		hasSucc bool
		wantRow int
		wantCol int
	}{
		{WrapNone, false, 0, 0},
		{WrapRow, true, 0, 0},
		{WrapGrid, true, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.wrap.String(), func(t *testing.T) {
			l, _ := LoadLayout("qwerty")
			l.Wrap = tc.wrap
			s := State{Layer: 0, Row: 0, Col: lastCol}
			su, ok := l.Apply(s, MoveRight)
			if ok != tc.hasSucc {
				t.Fatalf("Apply Right ok=%v, want %v", ok, tc.hasSucc)
			}
			if !tc.hasSucc {
				return
			}
			if su.Next.Row != tc.wantRow || su.Next.Col != tc.wantCol {
				t.Errorf("successor=(%d,%d), want (%d,%d)", su.Next.Row, su.Next.Col, tc.wantRow, tc.wantCol)
			}
		})
	}
}

func TestMove_WrapGrid_VerticalWrap(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	l.Wrap = WrapGrid
	s := State{Layer: 0, Row: 0, Col: 0}
	nr, nc, ok := l.MoveCursor(s, MoveUp)
	if !ok {
		t.Fatal("WrapGrid Up from (0,0) should be valid")
	}
	if nr != 3 || nc != 0 {
		t.Errorf("WrapGrid Up from (0,0) -> (%d,%d), want (3,0)", nr, nc)
	}
}

func TestMove_WrapRow_VerticalWrap(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	l.Wrap = WrapRow
	s := State{Layer: 0, Row: 0, Col: 0}
	nr, nc, ok := l.MoveCursor(s, MoveUp)
	if !ok {
		t.Fatal("WrapRow Up from (0,0) should be valid")
	}
	if nr != 3 || nc != 0 {
		t.Errorf("WrapRow Up from (0,0) -> (%d,%d), want (3,0)", nr, nc)
	}
}

// ---------------------------------------------------------------------------
// ParseWrap
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
		mode, override, err := ParseWrap(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseWrap(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseWrap(%q) unexpected error: %v", tc.input, err)
		}
		if mode != tc.wantMode || override != tc.wantOverride {
			t.Errorf("ParseWrap(%q) = (%v, %v), want (%v, %v)", tc.input, mode, override, tc.wantMode, tc.wantOverride)
		}
	}
}

// ---------------------------------------------------------------------------
// String methods
// ---------------------------------------------------------------------------

func TestCapsMode_String(t *testing.T) {
	if CapsOff.String() != "off" {
		t.Errorf("CapsOff = %q, want \"off\"", CapsOff.String())
	}
	if CapsOneShot.String() != "one-shot" {
		t.Errorf("CapsOneShot = %q, want \"one-shot\"", CapsOneShot.String())
	}
	if CapsSticky.String() != "sticky" {
		t.Errorf("CapsSticky = %q, want \"sticky\"", CapsSticky.String())
	}
}

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
			t.Errorf("Move(%d) = %q, want %q", tc.m, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// applyOK paths
// ---------------------------------------------------------------------------

func TestApplyOK_SwitchLayer(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	// QWERTY layer 0, row 1, col 9 is the "123" switch key.
	s := State{Layer: 0, Row: 1, Col: 9}
	su, ok := l.Apply(s, MoveOK)
	if !ok {
		t.Fatal("Apply MoveOK on SwitchLayer key returned false")
	}
	if su.Step.Emitted != 0 {
		t.Errorf("SwitchLayer emitted %q, want 0", su.Step.Emitted)
	}
	if su.Next.Layer != 1 {
		t.Errorf("Next.Layer = %d, want 1", su.Next.Layer)
	}
}

func TestApplyOK_EmitNoShifted(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	// QWERTY layer 1 (numbers), row 0, col 0 is '1' (no Shifted variant).
	s := State{Layer: 1, Row: 0, Col: 0, Caps: CapsOneShot}
	su, ok := l.Apply(s, MoveOK)
	if !ok {
		t.Fatal("Apply MoveOK returned false")
	}
	if su.Step.Emitted != '1' {
		t.Errorf("emitted %q, want '1'", su.Step.Emitted)
	}
	if su.Next.Caps != CapsOneShot {
		t.Errorf("Caps = %v, want CapsOneShot", su.Next.Caps)
	}
}

// ---------------------------------------------------------------------------
// Caps FSM transitions
// ---------------------------------------------------------------------------

func findActionKey(l *Layout, action Action) (State, bool) {
	layer := &l.Layers[0]
	for r, row := range layer.Keys {
		for c, k := range row {
			if k.Action == action {
				return State{Layer: 0, Row: r, Col: c}, true
			}
		}
	}
	return State{}, false
}

func findGlyphKey(l *Layout, r rune) (State, bool) {
	layer := &l.Layers[0]
	for row, keys := range layer.Keys {
		for col, k := range keys {
			if k.Action == ActionEmit && k.Glyph == r {
				return State{Layer: 0, Row: row, Col: col}, true
			}
		}
	}
	return State{}, false
}

func pressOK(t *testing.T, l *Layout, s State) (State, rune) {
	t.Helper()
	su, ok := l.Apply(s, MoveOK)
	if !ok {
		t.Fatal("Apply MoveOK returned false")
	}
	return su.Next, su.Step.Emitted
}

func TestCaps_OneShotTransitions(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	capsSt, ok := findActionKey(l, ActionToggleCaps)
	if !ok {
		t.Fatal("no ActionToggleCaps key on layer 0")
	}
	capsSt.Caps = CapsOff
	afterToggle, _ := pressOK(t, l, capsSt)
	if afterToggle.Caps != CapsOneShot {
		t.Fatalf("after first toggle: Caps=%v, want CapsOneShot", afterToggle.Caps)
	}

	aSt, ok := findGlyphKey(l, 'a')
	if !ok {
		t.Fatal("no 'a' key on layer 0")
	}
	aSt.Caps = afterToggle.Caps

	afterEmit, emitted := pressOK(t, l, aSt)
	if emitted != 'A' {
		t.Errorf("emitted %q, want 'A'", emitted)
	}
	if afterEmit.Caps != CapsOff {
		t.Errorf("after one-shot emit: Caps=%v, want CapsOff", afterEmit.Caps)
	}
}

func TestCaps_StickyPromotion(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	capsSt, _ := findActionKey(l, ActionToggleCaps)
	capsSt.Caps = CapsOff

	after1, _ := pressOK(t, l, capsSt)
	if after1.Caps != CapsOneShot {
		t.Fatalf("after 1st toggle: %v, want CapsOneShot", after1.Caps)
	}
	capsSt.Caps = after1.Caps

	after2, _ := pressOK(t, l, capsSt)
	if after2.Caps != CapsSticky {
		t.Fatalf("after 2nd toggle: %v, want CapsSticky", after2.Caps)
	}

	aSt, _ := findGlyphKey(l, 'a')
	aSt.Caps = after2.Caps
	afterEmit, emitted := pressOK(t, l, aSt)
	if emitted != 'A' {
		t.Errorf("emitted %q, want 'A'", emitted)
	}
	if afterEmit.Caps != CapsSticky {
		t.Errorf("after sticky emit: Caps=%v, want CapsSticky", afterEmit.Caps)
	}
}

func TestCaps_StickyTurnsOff(t *testing.T) {
	l, _ := LoadLayout("qwerty")
	capsSt, _ := findActionKey(l, ActionToggleCaps)
	capsSt.Caps = CapsSticky
	after, _ := pressOK(t, l, capsSt)
	if after.Caps != CapsOff {
		t.Errorf("after toggle from Sticky: Caps=%v, want CapsOff", after.Caps)
	}
}

// ---------------------------------------------------------------------------
// clamp (private helper)
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
