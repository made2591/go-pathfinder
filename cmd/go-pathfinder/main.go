// Command go-pathfinder is the CLI entry point. It parses flags and
// delegates to the keyboard, solver, metrics, and sim packages.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/made2591/go-pathfinder/internal/keyboard"
	"github.com/made2591/go-pathfinder/internal/metrics"
	"github.com/made2591/go-pathfinder/internal/sim"
	"github.com/made2591/go-pathfinder/internal/solver"
)

func main() {
	layoutName := flag.String("layout", "qwerty", "keyboard layout (qwerty, alphabetical, appletv)")
	wrapName := flag.String("wrap", "", "wrap policy override: none, row, grid (default: use layout's own policy)")
	algoName := flag.String("algo", "dijkstra", "pathfinding algorithm (dijkstra, astar)")
	text := flag.String("text", "", "text to type; if empty only the layout is rendered")
	verbose := flag.Bool("v", false, "print every move with running cursor state")
	simFlag := flag.Bool("sim", false, "animate the cursor typing the text in-place (requires -text)")
	speedMs := flag.Int("speed", 250, "per-step delay in milliseconds for -sim mode (0 = no delay)")
	showMetrics := flag.Bool("metrics", false, "print entropy, dispersion, diameter and typing-complexity metrics alongside -text output")
	flag.Parse()

	if *simFlag {
		speed := time.Duration(*speedMs) * time.Millisecond
		if err := runSimCLI(*layoutName, *wrapName, *algoName, *text, speed); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(*layoutName, *wrapName, *algoName, *text, *verbose, *showMetrics); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(layoutName, wrapName, algoName, text string, verbose, showMetrics bool) error {
	layout, err := keyboard.LoadLayout(layoutName)
	if err != nil {
		return err
	}
	wrap, override, err := keyboard.ParseWrap(wrapName)
	if err != nil {
		return err
	}
	if override {
		layout.Wrap = wrap
	}
	finder, err := solver.LoadFinder(algoName)
	if err != nil {
		return err
	}

	if err := layout.Render(os.Stdout); err != nil {
		return err
	}
	if text == "" {
		return nil
	}

	plan, err := layout.Type(text, finder)
	if err != nil {
		return err
	}

	fmt.Printf("\nlayout: %s | algo: %s | text: %q\n", layout.Name, finder.Name(), text)
	fmt.Printf("clicks (= password cost): %d  (%.2f per character)\n",
		len(plan), float64(len(plan))/float64(utf8.RuneCountInString(text)))
	if showMetrics {
		metrics.PrintMetrics(text, layout, len(plan))
	}

	if !verbose {
		return nil
	}
	state := layout.Start()
	for i, step := range plan {
		su, _ := layout.Apply(state, step.Move)
		state = su.Next
		fmt.Printf("  %3d %-2s  layer=%d row=%d col=%d caps=%s",
			i+1, step.Move, state.Layer, state.Row, state.Col, state.Caps)
		if step.Emitted != 0 {
			fmt.Printf("  emit=%q", step.Emitted)
		}
		fmt.Println()
	}
	return nil
}

func runSimCLI(layoutName, wrapName, algoName, text string, speed time.Duration) error {
	if text == "" {
		return fmt.Errorf("-sim requires -text to be non-empty")
	}
	layout, err := keyboard.LoadLayout(layoutName)
	if err != nil {
		return err
	}
	wrap, override, err := keyboard.ParseWrap(wrapName)
	if err != nil {
		return err
	}
	if override {
		layout.Wrap = wrap
	}
	finder, err := solver.LoadFinder(algoName)
	if err != nil {
		return err
	}
	return sim.Run(layout, finder, text, speed)
}
