package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// isTTY reports whether w is an interactive terminal.
func isTTY(w *os.File) bool {
	info, err := w.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// simFrame holds all the data needed to render one animation frame.
type simFrame struct {
	layout  *Layout
	state   State
	typed   string
	target  string
	clicks  int
	flash   flashMode
}

type flashMode int

const (
	flashNone  flashMode = iota
	flashCaps            // highlight caps key (yellow-ish inverse)
	flashLayer           // highlight layer-switch key
	flashEmit            // highlight emitted key (green)
)

// cellWidth is the fixed character width used for every key cell in sim mode.
// renderCell already produces exactly 3 runes (" x " or "[x]"), so this must
// stay in sync with renderCell's output width. Changing it is a breaking
// assumption.
const cellWidth = 3

// renderSimLayer writes the active layer to w, highlighting the cursor at
// (curRow, curCol) with ANSI inverse video and optionally applying a flash
// style to the same cell.
func renderSimLayer(w io.Writer, layer *Layer, curRow, curCol int, flash flashMode) error {
	for r, row := range layer.Keys {
		var sb strings.Builder
		sb.WriteString("  ")
		for c, k := range row {
			cell := renderCell(k)
			if r == curRow && c == curCol {
				switch flash {
				case flashEmit:
					// bright green background
					fmt.Fprintf(&sb, "\x1b[42;1m%s\x1b[0m", cell)
				case flashCaps, flashLayer:
					// bright yellow background
					fmt.Fprintf(&sb, "\x1b[43;1m%s\x1b[0m", cell)
				default:
					// inverse video
					fmt.Fprintf(&sb, "\x1b[7m%s\x1b[0m", cell)
				}
			} else {
				sb.WriteString(cell)
			}
			if c < len(row)-1 {
				sb.WriteByte(' ')
			}
		}
		// pad to fixed width so stale characters from a previous (wider) layer
		// are overwritten
		line := sb.String()
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

// renderStatus writes the single-line status bar.
func renderStatus(w io.Writer, sf simFrame) error {
	capsStr := sf.state.Caps.String()
	layerName := sf.layout.Layers[sf.state.Layer].Name

	// determine next rune to type
	remaining := []rune(sf.target[len([]rune(sf.typed)):])
	nextStr := "-"
	if len(remaining) > 0 {
		nextStr = fmt.Sprintf("%q", remaining[0])
	}

	_, err := fmt.Fprintf(w,
		"target: %-20s  typed: %-20s  next: %-6s  layer: %-8s  caps: %-3s  clicks: %d\n",
		fmt.Sprintf("%q", sf.target),
		fmt.Sprintf("%q", sf.typed),
		nextStr,
		layerName,
		capsStr,
		sf.clicks,
	)
	return err
}

// planStep associates a Step with the resulting State after that step so the
// simulator can replay the full sequence without re-running pathfinding.
type planStep struct {
	step  Step
	after State // cursor state after this step
}

// buildPlan runs the pathfinder once and returns an annotated step list
// alongside the full typed string for verification. This is a pure function
// useful for testing independently of the TTY animation loop.
func buildPlan(layout *Layout, finder Pathfinder, text string) ([]planStep, error) {
	state := layout.Start()
	var out []planStep
	for _, ch := range text {
		steps, end, err := finder.Find(layout, state, ch)
		if err != nil {
			return nil, err
		}
		for i, s := range steps {
			var after State
			if i == len(steps)-1 {
				after = end
			} else {
				su, _ := layout.apply(state, s.Move)
				after = su.Next
			}
			out = append(out, planStep{step: s, after: after})
			state = after
		}
	}
	return out, nil
}

// resetTerminal restores visible cursor and default colours.
func resetTerminal(w *os.File) {
	fmt.Fprint(w, "\x1b[?25h\x1b[0m")
}

// runSim animates the cursor moving on the keyboard one click at a time.
// If stdout is not a TTY it falls back to the static render + verbose log.
func runSim(layout *Layout, finder Pathfinder, text string, speed time.Duration) error {
	tty := isTTY(os.Stdout)

	plan, err := buildPlan(layout, finder, text)
	if err != nil {
		return err
	}

	if !tty {
		// Non-TTY fallback: static render then step log.
		if err := layout.Render(os.Stdout); err != nil {
			return err
		}
		fmt.Printf("\nclicks: %d\n", len(plan))
		typed := ""
		for i, ps := range plan {
			fmt.Printf("  %3d %-2s  layer=%d row=%d col=%d caps=%s",
				i+1, ps.step.Move,
				ps.after.Layer, ps.after.Row, ps.after.Col, ps.after.Caps)
			if ps.step.Emitted != 0 {
				typed += string(ps.step.Emitted)
				fmt.Printf("  emit=%q", ps.step.Emitted)
			}
			fmt.Println()
		}
		fmt.Printf("typed: %q\n", typed)
		return nil
	}

	// TTY path: hide cursor, set up cleanup on exit and SIGINT.
	fmt.Fprint(os.Stdout, "\x1b[?25l") // hide cursor
	defer resetTerminal(os.Stdout)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		<-sigCh
		resetTerminal(os.Stdout)
		os.Exit(130)
	}()

	// First frame: clear screen.
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	state := layout.Start()
	typed := ""
	clicks := 0

	drawFrame := func(flash flashMode) error {
		fmt.Fprint(os.Stdout, "\x1b[H") // move cursor to top-left
		sf := simFrame{
			layout: layout,
			state:  state,
			typed:  typed,
			target: text,
			clicks: clicks,
			flash:  flash,
		}
		if err := renderStatus(os.Stdout, sf); err != nil {
			return err
		}
		layer := &layout.Layers[state.Layer]
		return renderSimLayer(os.Stdout, layer, state.Row, state.Col, flash)
	}

	// Draw initial frame before any moves.
	if err := drawFrame(flashNone); err != nil {
		return err
	}

	flashMs := speed / 3
	if flashMs < 30*time.Millisecond {
		flashMs = 30 * time.Millisecond
	}
	if speed == 0 {
		flashMs = 0
	}

	for _, ps := range plan {
		clicks++

		// Determine flash style for this OK press.
		fm := flashNone
		if ps.step.Move == MoveOK {
			k := layout.Layers[state.Layer].Keys[state.Row][state.Col]
			switch k.Action {
			case ActionEmit:
				fm = flashEmit
			case ActionToggleCaps:
				fm = flashCaps
			case ActionSwitchLayer:
				fm = flashLayer
			}
		}

		// Update logical state first.
		state = ps.after
		if ps.step.Emitted != 0 {
			typed += string(ps.step.Emitted)
		}

		if fm != flashNone && speed > 0 {
			// Flash: draw with highlight, wait, then redraw normal.
			if err := drawFrame(fm); err != nil {
				return err
			}
			time.Sleep(flashMs)
		}

		if err := drawFrame(flashNone); err != nil {
			return err
		}

		if speed > 0 {
			time.Sleep(speed)
		}
	}

	// Final status line below the keyboard.
	fmt.Fprintf(os.Stdout, "\ndone — typed: %q  total clicks: %d\n", typed, clicks)
	return nil
}
