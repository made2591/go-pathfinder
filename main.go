package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	layoutName := flag.String("layout", "qwerty", "keyboard layout (qwerty)")
	algoName := flag.String("algo", "dijkstra", "pathfinding algorithm (dijkstra, astar)")
	text := flag.String("text", "", "text to type; if empty only the layout is rendered")
	verbose := flag.Bool("v", false, "print every move with running cursor state")
	sim := flag.Bool("sim", false, "animate the cursor typing the text in-place (requires -text)")
	speedMs := flag.Int("speed", 250, "per-step delay in milliseconds for -sim mode (0 = no delay)")
	flag.Parse()

	if *sim {
		speed := time.Duration(*speedMs) * time.Millisecond
		if err := runSimCLI(*layoutName, *algoName, *text, speed); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(*layoutName, *algoName, *text, *verbose); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(layoutName, algoName, text string, verbose bool) error {
	layout, err := loadLayout(layoutName)
	if err != nil {
		return err
	}
	finder, err := loadFinder(algoName)
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
	fmt.Printf("clicks: %d  (%.2f per character)\n", len(plan), float64(len(plan))/float64(countRunes(text)))

	if !verbose {
		return nil
	}
	state := layout.Start()
	for i, step := range plan {
		su, _ := layout.apply(state, step.Move)
		state = su.Next
		fmt.Printf("  %3d %-2s  layer=%d row=%d col=%d caps=%v", i+1, step.Move, state.Layer, state.Row, state.Col, state.Caps)
		if step.Emitted != 0 {
			fmt.Printf("  emit=%q", step.Emitted)
		}
		fmt.Println()
	}
	return nil
}

func countRunes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// runSimCLI resolves the layout and finder then delegates to runSim.
func runSimCLI(layoutName, algoName, text string, speed time.Duration) error {
	if text == "" {
		return fmt.Errorf("-sim requires -text to be non-empty")
	}
	layout, err := loadLayout(layoutName)
	if err != nil {
		return err
	}
	finder, err := loadFinder(algoName)
	if err != nil {
		return err
	}
	return runSim(layout, finder, text, speed)
}
