// Package keyboard models on-screen TV-remote keyboards as a state graph
// and exposes the types the rest of the project builds on: layouts, layers,
// keys, the cursor state, and the Pathfinder interface that solvers
// implement.
package keyboard

import (
	"fmt"
	"io"
	"strings"
)

// Move is a single user input on the remote control.
type Move int

const (
	MoveUp Move = iota
	MoveDown
	MoveLeft
	MoveRight
	MoveOK
)

func (m Move) String() string {
	return [...]string{"↑", "↓", "←", "→", "OK"}[m]
}

// Action describes what happens when OK is pressed on a key.
type Action int

const (
	ActionEmit        Action = iota // emit the rune under the cursor
	ActionToggleCaps                // flip the caps-lock flag
	ActionSwitchLayer               // jump to another layer
)

// Key is a single cell on a keyboard layer.
type Key struct {
	Glyph   rune   // emitted when caps is off (or always, if Shifted == 0)
	Shifted rune   // emitted when caps is on; 0 means "caps does not affect this key"
	Action  Action // what OK does
	Target  int    // destination layer index when Action == ActionSwitchLayer
	Label   string // short label for non-printable keys (e.g. "⇧", "123")
}

// WrapMode controls cursor behaviour at the edges of a layer.
type WrapMode int

const (
	WrapNone WrapMode = iota // movement off the edge is forbidden
	WrapRow                  // wrap stays on the same row (or column for vertical)
	WrapGrid                 // right-end wrap moves to the first cell of the next row
)

func (w WrapMode) String() string {
	return [...]string{"none", "row", "grid"}[w]
}

// Layer is a 2D grid of keys. All rows are assumed to have the same length.
type Layer struct {
	Name string
	Keys [][]Key
}

func (l *Layer) Rows() int { return len(l.Keys) }
func (l *Layer) Cols() int {
	if len(l.Keys) == 0 {
		return 0
	}
	return len(l.Keys[0])
}

// Layout is a complete on-screen keyboard.
type Layout struct {
	Name       string
	Layers     []Layer
	Wrap       WrapMode
	StartLayer int
	StartRow   int
	StartCol   int
}

// Start returns the initial cursor state.
func (l *Layout) Start() State {
	return State{Layer: l.StartLayer, Row: l.StartRow, Col: l.StartCol}
}

// Render writes a human-readable ASCII view of the layout to w.
func (l *Layout) Render(w io.Writer) error {
	for i, layer := range l.Layers {
		if _, err := fmt.Fprintf(w, "[layer %d] %s\n", i, layer.Name); err != nil {
			return err
		}
		for _, row := range layer.Keys {
			cells := make([]string, len(row))
			for c, k := range row {
				cells[c] = RenderCell(k)
			}
			if _, err := fmt.Fprintf(w, "  %s\n", strings.Join(cells, " ")); err != nil {
				return err
			}
		}
	}
	return nil
}

// RenderCell formats a single key as a 3-character cell. Exported so the
// animated simulator can reuse the exact same rendering.
func RenderCell(k Key) string {
	switch k.Action {
	case ActionToggleCaps:
		return fmt.Sprintf("[%s]", orDefault(k.Label, "⇧"))
	case ActionSwitchLayer:
		return fmt.Sprintf("[%s]", orDefault(k.Label, fmt.Sprintf("→%d", k.Target)))
	default:
		if k.Glyph == ' ' {
			return "[ ]"
		}
		return fmt.Sprintf(" %c ", k.Glyph)
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// ParseWrap converts a CLI flag string to a WrapMode. The bool is true when
// the caller explicitly asked for an override; an empty string means "use
// the layout's own default".
func ParseWrap(s string) (WrapMode, bool, error) {
	switch strings.ToLower(s) {
	case "":
		return WrapNone, false, nil
	case "none":
		return WrapNone, true, nil
	case "row":
		return WrapRow, true, nil
	case "grid":
		return WrapGrid, true, nil
	default:
		return WrapNone, false, fmt.Errorf("unknown wrap policy %q (valid: none, row, grid)", s)
	}
}

// LoadLayout returns a built-in layout by name.
func LoadLayout(name string) (*Layout, error) {
	switch strings.ToLower(name) {
	case "qwerty":
		return qwerty(), nil
	case "alphabetical":
		return alphabetical(), nil
	case "appletv":
		return appletv(), nil
	default:
		return nil, fmt.Errorf("unknown layout %q (valid: qwerty, alphabetical, appletv)", name)
	}
}

// qwerty is a smart-TV style 4×10 QWERTY with caps, numbers, and symbols layers.
func qwerty() *Layout {
	const (
		layerLetters = 0
		layerNumbers = 1
		layerSymbols = 2
	)

	letters := Layer{Name: "letters", Keys: [][]Key{
		{ke('q'), ke('w'), ke('e'), ke('r'), ke('t'), ke('y'), ke('u'), ke('i'), ke('o'), ke('p')},
		{ke('a'), ke('s'), ke('d'), ke('f'), ke('g'), ke('h'), ke('j'), ke('k'), ke('l'), ks(layerNumbers, "123")},
		{kc("⇧"), ke('z'), ke('x'), ke('c'), ke('v'), ke('b'), ke('n'), ke('m'), ke(','), ke('.')},
		{kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' ')},
	}}

	numbers := Layer{Name: "numbers", Keys: [][]Key{
		{kp('1'), kp('2'), kp('3'), kp('4'), kp('5'), kp('6'), kp('7'), kp('8'), kp('9'), kp('0')},
		{kp('-'), kp('/'), kp(':'), kp(';'), kp('('), kp(')'), kp('$'), kp('&'), kp('@'), kp('"')},
		{ks(layerSymbols, "#+="), kp('.'), kp(','), kp('?'), kp('!'), kp('\''), kp('_'), kp('+'), kp('='), ks(layerLetters, "abc")},
		{kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' ')},
	}}

	symbols := Layer{Name: "symbols", Keys: [][]Key{
		{kp('['), kp(']'), kp('{'), kp('}'), kp('#'), kp('%'), kp('^'), kp('*'), kp('+'), kp('=')},
		{kp('_'), kp('\\'), kp('|'), kp('~'), kp('<'), kp('>'), kp('€'), kp('£'), kp('¥'), kp('•')},
		{ks(layerNumbers, "123"), kp('.'), kp(','), kp('?'), kp('!'), kp('\''), kp('"'), kp('`'), kp('§'), ks(layerLetters, "abc")},
		{kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' '), kp(' ')},
	}}

	return &Layout{
		Name:       "QWERTY",
		Layers:     []Layer{letters, numbers, symbols},
		Wrap:       WrapRow,
		StartLayer: layerLetters,
		StartRow:   0,
		StartCol:   0,
	}
}

// ke is a printable letter key with a shifted (uppercase) variant.
func ke(r rune) Key {
	sh := rune(0)
	if r >= 'a' && r <= 'z' {
		sh = r - 32
	}
	return Key{Glyph: r, Shifted: sh, Action: ActionEmit}
}

// kp is a printable key without a shifted variant (digits, symbols, space).
func kp(r rune) Key { return Key{Glyph: r, Action: ActionEmit} }

// kc is a caps-lock toggle key.
func kc(label string) Key { return Key{Action: ActionToggleCaps, Label: label} }

// ks is a layer-switch key.
func ks(target int, label string) Key {
	return Key{Action: ActionSwitchLayer, Target: target, Label: label}
}

// alphabetical is a 5-row × 6-column single-layer layout found on older TVs
// and some game consoles. No caps layer, no number/symbol layers. The last
// four cells of the final row are all space keys. Default wrap: WrapNone.
func alphabetical() *Layout {
	letters := Layer{Name: "letters", Keys: [][]Key{
		{ke('a'), ke('b'), ke('c'), ke('d'), ke('e'), ke('f')},
		{ke('g'), ke('h'), ke('i'), ke('j'), ke('k'), ke('l')},
		{ke('m'), ke('n'), ke('o'), ke('p'), ke('q'), ke('r')},
		{ke('s'), ke('t'), ke('u'), ke('v'), ke('w'), ke('x')},
		{ke('y'), ke('z'), kp(' '), kp(' '), kp(' '), kp(' ')},
	}}
	return &Layout{
		Name:       "alphabetical",
		Layers:     []Layer{letters},
		Wrap:       WrapNone,
		StartLayer: 0,
		StartRow:   0,
		StartCol:   0,
	}
}

// appletv models the Apple-TV-style three-layer single-row keyboard. Default
// wrap: WrapRow (Apple TV's always-on horizontal wrap-around).
func appletv() *Layout {
	const (
		layerLetters = 0
		layerNumbers = 1
		layerSymbols = 2
	)

	letters := Layer{Name: "letters", Keys: [][]Key{{
		ke('a'), ke('b'), ke('c'), ke('d'), ke('e'), ke('f'),
		ke('g'), ke('h'), ke('i'), ke('j'), ke('k'), ke('l'),
		ke('m'), ke('n'), ke('o'), ke('p'), ke('q'), ke('r'),
		ke('s'), ke('t'), ke('u'), ke('v'), ke('w'), ke('x'),
		ke('y'), ke('z'),
		kc("⇧"),
		ks(layerNumbers, "123"),
		kp(' '),
	}}}

	numbers := Layer{Name: "numbers", Keys: [][]Key{{
		kp('0'), kp('1'), kp('2'), kp('3'), kp('4'),
		kp('5'), kp('6'), kp('7'), kp('8'), kp('9'),
		ks(layerSymbols, "#+="),
		ks(layerLetters, "abc"),
		kp(' '),
	}}}

	symbols := Layer{Name: "symbols", Keys: [][]Key{{
		kp('.'), kp(','), kp('!'), kp('?'), kp('\''), kp('"'),
		kp('-'), kp('_'), kp('('), kp(')'), kp('['), kp(']'),
		kp('{'), kp('}'), kp('<'), kp('>'), kp('/'), kp('\\'),
		kp('|'), kp('+'), kp('='), kp('*'), kp('&'), kp('^'),
		kp('%'), kp('$'), kp('#'), kp('@'), kp('~'), kp('`'),
		kp(':'),
		ks(layerNumbers, "123"),
		ks(layerLetters, "abc"),
		kp(' '),
	}}}

	return &Layout{
		Name:       "appletv",
		Layers:     []Layer{letters, numbers, symbols},
		Wrap:       WrapRow,
		StartLayer: layerLetters,
		StartRow:   0,
		StartCol:   0,
	}
}
