package main

// State is a node in the typing graph: the cursor position plus the caps-lock
// flag. Two states with the same (layer, row, col) but different Caps are
// distinct nodes because they emit different runes when OK is pressed.
type State struct {
	Layer int
	Row   int
	Col   int
	Caps  bool
}

// Step is a single edge traversal in the graph. Emitted is non-zero only when
// the move is OK on a key with Action == ActionEmit.
type Step struct {
	Move    Move
	Emitted rune
}

// Successor pairs an outgoing edge with the resulting state.
type Successor struct {
	Step Step
	Next State
}

// Successors returns all valid moves from s and their resulting states.
// Every edge has uniform cost 1 — a single physical click on the remote.
func (l *Layout) Successors(s State) []Successor {
	out := make([]Successor, 0, 5)
	for m := MoveUp; m <= MoveOK; m++ {
		if su, ok := l.apply(s, m); ok {
			out = append(out, su)
		}
	}
	return out
}

// apply executes a single move and returns the resulting Successor.
// The bool is false when the move is illegal (e.g. moving off-edge with WrapNone).
func (l *Layout) apply(s State, m Move) (Successor, bool) {
	if m == MoveOK {
		return l.applyOK(s)
	}
	nr, nc, ok := l.move(s, m)
	if !ok {
		return Successor{}, false
	}
	return Successor{
		Step: Step{Move: m},
		Next: State{Layer: s.Layer, Row: nr, Col: nc, Caps: s.Caps},
	}, true
}

func (l *Layout) applyOK(s State) (Successor, bool) {
	k := l.Layers[s.Layer].Keys[s.Row][s.Col]
	switch k.Action {
	case ActionEmit:
		r := k.Glyph
		if s.Caps && k.Shifted != 0 {
			r = k.Shifted
		}
		return Successor{Step: Step{Move: MoveOK, Emitted: r}, Next: s}, true
	case ActionToggleCaps:
		ns := s
		ns.Caps = !s.Caps
		return Successor{Step: Step{Move: MoveOK}, Next: ns}, true
	case ActionSwitchLayer:
		target := &l.Layers[k.Target]
		ns := State{
			Layer: k.Target,
			Row:   clamp(s.Row, 0, target.Rows()-1),
			Col:   clamp(s.Col, 0, target.Cols()-1),
			Caps:  s.Caps,
		}
		return Successor{Step: Step{Move: MoveOK}, Next: ns}, true
	}
	return Successor{}, false
}

// move returns the new (row, col) after a directional move, applying the
// layout's wrap policy. The bool is false when the move is illegal.
func (l *Layout) move(s State, m Move) (int, int, bool) {
	layer := &l.Layers[s.Layer]
	rows, cols := layer.Rows(), layer.Cols()
	r, c := s.Row, s.Col
	switch m {
	case MoveUp:
		r--
	case MoveDown:
		r++
	case MoveLeft:
		c--
	case MoveRight:
		c++
	}
	switch l.Wrap {
	case WrapNone:
		if r < 0 || r >= rows || c < 0 || c >= cols {
			return 0, 0, false
		}
	case WrapRow:
		r = (r%rows + rows) % rows
		c = (c%cols + cols) % cols
	case WrapGrid:
		// Linearise (r*cols + c) and wrap modulo rows*cols.
		idx := r*cols + c
		total := rows * cols
		idx = (idx%total + total) % total
		r, c = idx/cols, idx%cols
	}
	return r, c, true
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Type plans a click sequence for typing the given text using the supplied
// pathfinder. The plan is a flat list of Steps; concatenated emissions
// reproduce the input text.
//
// This is the greedy single-source variant: after each character we keep only
// the best end-state and start the next search from there. For the layouts
// modelled here, the optimum for the *next* character is essentially never
// hurt by choosing the cheapest path for the *current* one — but a globally
// optimal version (multi-source Dijkstra layered on the character sequence)
// would be a drop-in replacement.
func (l *Layout) Type(text string, finder Pathfinder) ([]Step, error) {
	state := l.Start()
	var plan []Step
	for _, ch := range text {
		steps, end, err := finder.Find(l, state, ch)
		if err != nil {
			return nil, err
		}
		plan = append(plan, steps...)
		state = end
	}
	return plan, nil
}
