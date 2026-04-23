package main

import "testing"

// TestLoadLayout_Presets verifies that each built-in layout loads successfully
// and has the expected character at (row=0, col=0) of layer 0.
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
			l, err := loadLayout(tc.name)
			if err != nil {
				t.Fatalf("loadLayout(%q): %v", tc.name, err)
			}
			if len(l.Layers) == 0 {
				t.Fatal("no layers")
			}
			layer0 := &l.Layers[0]
			if layer0.Rows() == 0 || layer0.Cols() == 0 {
				t.Fatal("layer 0 has no keys")
			}
			got := layer0.Keys[0][0].Glyph
			if got != tc.wantGlyph {
				t.Errorf("layer0[0][0].Glyph = %q, want %q", got, tc.wantGlyph)
			}
		})
	}
}

// TestWrapPolicies_EdgeMoves verifies that moving Right from the last column
// of row 0 on QWERTY behaves correctly under each WrapMode.
func TestWrapPolicies_EdgeMoves(t *testing.T) {
	base, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	// QWERTY layer 0 has 4 rows × 10 cols.
	lastCol := base.Layers[0].Cols() - 1 // 9

	cases := []struct {
		wrap     WrapMode
		hasSucc  bool
		wantRow  int
		wantCol  int
	}{
		{WrapNone, false, 0, 0},   // off-edge → no successor
		{WrapRow, true, 0, 0},    // wraps within same row → (0, 0)
		{WrapGrid, true, 1, 0},   // linearises → next row first cell → (1, 0)
	}

	for _, tc := range cases {
		t.Run(tc.wrap.String(), func(t *testing.T) {
			l, _ := loadLayout("qwerty")
			l.Wrap = tc.wrap
			s := State{Layer: 0, Row: 0, Col: lastCol}
			su, ok := l.apply(s, MoveRight)
			if ok != tc.hasSucc {
				t.Fatalf("apply Right ok=%v, want %v", ok, tc.hasSucc)
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
