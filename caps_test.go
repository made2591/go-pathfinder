package main

import "testing"

// findKey searches layer 0 for the first key with the given action and returns
// a State positioned at that key.
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

// findGlyphKey searches layer 0 for an ActionEmit key whose Glyph matches r.
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

// pressOK applies MoveOK from state s and returns the resulting state.
func pressOK(t *testing.T, l *Layout, s State) (State, rune) {
	t.Helper()
	su, ok := l.apply(s, MoveOK)
	if !ok {
		t.Fatal("apply MoveOK returned false")
	}
	return su.Next, su.Step.Emitted
}

// TestCaps_OneShotTransitions: Off → toggle → OneShot; then emit 'a' → 'A',
// state transitions to CapsOff (one-shot consumed).
func TestCaps_OneShotTransitions(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}

	capsSt, ok := findActionKey(l, ActionToggleCaps)
	if !ok {
		t.Fatal("no ActionToggleCaps key on layer 0")
	}

	// Ensure we start from CapsOff.
	capsSt.Caps = CapsOff
	afterToggle, _ := pressOK(t, l, capsSt)
	if afterToggle.Caps != CapsOneShot {
		t.Fatalf("after first toggle: Caps=%v, want CapsOneShot", afterToggle.Caps)
	}

	// Move the cursor to 'a' key, keep caps from afterToggle.
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

// TestCaps_StickyPromotion: Off → toggle → OneShot → toggle → Sticky;
// then emit 'a' → 'A', state stays CapsSticky.
func TestCaps_StickyPromotion(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}

	capsSt, ok := findActionKey(l, ActionToggleCaps)
	if !ok {
		t.Fatal("no ActionToggleCaps key on layer 0")
	}
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

	aSt, ok := findGlyphKey(l, 'a')
	if !ok {
		t.Fatal("no 'a' key on layer 0")
	}
	aSt.Caps = after2.Caps

	afterEmit, emitted := pressOK(t, l, aSt)
	if emitted != 'A' {
		t.Errorf("emitted %q, want 'A'", emitted)
	}
	if afterEmit.Caps != CapsSticky {
		t.Errorf("after sticky emit: Caps=%v, want CapsSticky", afterEmit.Caps)
	}
}

// TestCaps_StickyTurnsOff: Sticky → toggle → CapsOff.
func TestCaps_StickyTurnsOff(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}

	capsSt, ok := findActionKey(l, ActionToggleCaps)
	if !ok {
		t.Fatal("no ActionToggleCaps key on layer 0")
	}
	capsSt.Caps = CapsSticky

	after, _ := pressOK(t, l, capsSt)
	if after.Caps != CapsOff {
		t.Errorf("after toggle from Sticky: Caps=%v, want CapsOff", after.Caps)
	}
}
