// Package sim animates the cursor moving on the keyboard one click at a time
// using ANSI escape codes. Falls back to a static render + verbose log when
// stdout is not a TTY.
package sim

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/made2591/go-pathfinder/internal/keyboard"
)

func isTTY(w *os.File) bool {
	info, err := w.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

type simFrame struct {
	layout *keyboard.Layout
	state  keyboard.State
	typed  string
	target string
	clicks int
	flash  flashMode
}

type flashMode int

const (
	flashNone  flashMode = iota
	flashCaps            // highlight caps key (yellow)
	flashLayer           // highlight layer-switch key
	flashEmit            // highlight emitted key (green)
)

// renderSimLayer writes the active layer to w, highlighting the cursor with
// ANSI inverse video and optionally applying a flash style to the same cell.
func renderSimLayer(w io.Writer, layer *keyboard.Layer, curRow, curCol int, flash flashMode) error {
	for r, row := range layer.Keys {
		var sb strings.Builder
		sb.WriteString("  ")
		for c, k := range row {
			cell := keyboard.RenderCell(k)
			if r == curRow && c == curCol {
				switch flash {
				case flashEmit:
					fmt.Fprintf(&sb, "\x1b[42;1m%s\x1b[0m", cell)
				case flashCaps, flashLayer:
					fmt.Fprintf(&sb, "\x1b[43;1m%s\x1b[0m", cell)
				default:
					fmt.Fprintf(&sb, "\x1b[7m%s\x1b[0m", cell)
				}
			} else {
				sb.WriteString(cell)
			}
			if c < len(row)-1 {
				sb.WriteByte(' ')
			}
		}
		line := sb.String()
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

func renderStatus(w io.Writer, sf simFrame) error {
	capsStr := sf.state.Caps.String()
	layerName := sf.layout.Layers[sf.state.Layer].Name

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

// PlanStep associates a Step with the resulting State after that step so the
// simulator can replay the full sequence without re-running pathfinding.
type PlanStep struct {
	Step  keyboard.Step
	After keyboard.State
}

// BuildPlan runs the pathfinder once and returns an annotated step list. Pure
// function, useful for testing independently of the TTY animation loop.
func BuildPlan(layout *keyboard.Layout, finder keyboard.Pathfinder, text string) ([]PlanStep, error) {
	state := layout.Start()
	var out []PlanStep
	for _, ch := range text {
		steps, end, err := finder.Find(layout, state, ch)
		if err != nil {
			return nil, err
		}
		for i, s := range steps {
			var after keyboard.State
			if i == len(steps)-1 {
				after = end
			} else {
				su, _ := layout.Apply(state, s.Move)
				after = su.Next
			}
			out = append(out, PlanStep{Step: s, After: after})
			state = after
		}
	}
	return out, nil
}

func resetTerminal(w *os.File) {
	fmt.Fprint(w, "\x1b[?25h\x1b[0m")
}

// Run animates the cursor moving on the keyboard one click at a time. If
// stdout is not a TTY it falls back to the static render + verbose log.
func Run(layout *keyboard.Layout, finder keyboard.Pathfinder, text string, speed time.Duration) error {
	tty := isTTY(os.Stdout)

	plan, err := BuildPlan(layout, finder, text)
	if err != nil {
		return err
	}

	if !tty {
		if err := layout.Render(os.Stdout); err != nil {
			return err
		}
		fmt.Printf("\nclicks: %d\n", len(plan))
		typed := ""
		for i, ps := range plan {
			fmt.Printf("  %3d %-2s  layer=%d row=%d col=%d caps=%s",
				i+1, ps.Step.Move,
				ps.After.Layer, ps.After.Row, ps.After.Col, ps.After.Caps)
			if ps.Step.Emitted != 0 {
				typed += string(ps.Step.Emitted)
				fmt.Printf("  emit=%q", ps.Step.Emitted)
			}
			fmt.Println()
		}
		fmt.Printf("typed: %q\n", typed)
		return nil
	}

	fmt.Fprint(os.Stdout, "\x1b[?25l")
	defer resetTerminal(os.Stdout)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		<-sigCh
		resetTerminal(os.Stdout)
		os.Exit(130)
	}()

	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	state := layout.Start()
	typed := ""
	clicks := 0

	drawFrame := func(flash flashMode) error {
		fmt.Fprint(os.Stdout, "\x1b[H")
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

		fm := flashNone
		if ps.Step.Move == keyboard.MoveOK {
			k := layout.Layers[state.Layer].Keys[state.Row][state.Col]
			switch k.Action {
			case keyboard.ActionEmit:
				fm = flashEmit
			case keyboard.ActionToggleCaps:
				fm = flashCaps
			case keyboard.ActionSwitchLayer:
				fm = flashLayer
			}
		}

		state = ps.After
		if ps.Step.Emitted != 0 {
			typed += string(ps.Step.Emitted)
		}

		if fm != flashNone && speed > 0 {
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

	fmt.Fprintf(os.Stdout, "\ndone — typed: %q  total clicks: %d\n", typed, clicks)
	return nil
}
