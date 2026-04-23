# go-pathfinder

![go-pathfinder demo](./demo.gif)

Model on-screen TV-remote keyboards as a state graph, find the cheapest click
sequence to type a string, with three layouts (QWERTY, alphabetical, Apple TV),
three wrap policies (none, row, grid), a sticky-caps FSM, both Dijkstra and A\*
solvers, and an animated `-sim` mode for watching the cursor work in the
terminal. The fundamental insight is that the cursor state is not just `(row,
col)` — it is `(layer, row, col, capsMode)`, so layer switches and caps-lock
toggles are first-class edges in the graph, not special cases bolted on after
the fact.

The math, the modelling decisions, and the typing-complexity metric Ψ (H ·
T̃ / cost per char) are explained in detail in the companion blog post at
<https://madeddu.xyz/posts/go-pathfinder/>. This repo is the implementation;
the post is where the motivation and analysis live.

---

## Quick start

```sh
make build

# Type "Hello" on the default QWERTY layout with Dijkstra
./go-pathfinder -text "Hello"

# Apple TV layout, A* solver, animated at 80 ms per step
./go-pathfinder -layout appletv -algo astar -text "hello" -sim -speed 80

# Alphabetical layout, wrap=grid, print every move
./go-pathfinder -layout alphabetical -wrap grid -text "abc" -v

# QWERTY with full metrics (H, T, T̃, Ψ alongside the click count)
./go-pathfinder -text "qwerty" -metrics
```

---

## Flags

| Flag | Default | Purpose | Accepted values |
|------|---------|---------|-----------------|
| `-layout` | `qwerty` | Keyboard layout to use | `qwerty`, `alphabetical`, `appletv` |
| `-algo` | `dijkstra` | Pathfinding algorithm | `dijkstra`, `astar` |
| `-wrap` | _(layout default)_ | Override the wrap policy | `none`, `row`, `grid` |
| `-text` | _(empty)_ | String to type; renders the layout only when empty | any string |
| `-v` | `false` | Print every move with running cursor state | boolean flag |
| `-sim` | `false` | Animate the cursor in-place (requires `-text`) | boolean flag |
| `-speed` | `250` | Per-step delay in milliseconds for `-sim` mode | integer ≥ 0 |
| `-metrics` | `false` | Print entropy H, dispersion T, T̃ and Ψ alongside the click count | boolean flag |

---

## Project layout

```
cmd/
  go-pathfinder/        main.go           — CLI entry: flag parsing + dispatch only
internal/
  keyboard/             layout.go         — Layout, Layer, Key, WrapMode, presets,
                                            ParseWrap, LoadLayout, Render
                        graph.go          — State, Step, Successor, Pathfinder
                                            interface, (layer,row,col,capsMode)
                                            state graph, sticky-caps FSM, Type
  solver/               solver.go         — Dijkstra + A* implementing Pathfinder,
                                            shared search core, inline min-heap
  metrics/              metrics.go        — Entropy, Dispersion, Diameter, Psi,
                                            PrintMetrics
  sim/                  sim.go            — in-terminal animation via ANSI escapes,
                                            non-TTY static fallback
Makefile                                  — build/test/fmt/lint/coverage targets
README.md
demo.gif
```

Dependency direction, clean and one-way:

```
keyboard   → (nothing)
solver     → keyboard
metrics    → keyboard
sim        → keyboard
cmd/...    → keyboard, solver, metrics, sim
```

The `Pathfinder` interface lives in `keyboard` (alongside `Layout`) so
`Layout.Type(text, finder)` can take it without creating a `solver →
keyboard → solver` cycle. `solver.Dijkstra` and `solver.AStar` are the two
concrete implementations shipped today.

---

## Build / test

```sh
make build          # compile ./cmd/go-pathfinder to ./go-pathfinder
make test           # go test ./... -count=1
make fmt            # gofmt -s -w .
make lint           # golangci-lint run, or falls back to go vet
make fix            # fmt + golangci-lint --fix
make clean          # remove the compiled binary
make coverage       # per-function coverage summary on all packages
make coverage-html  # same, plus open coverage.html in the browser
```

Tests live next to the code they cover, one `_test.go` per package.
`solver_test` and `metrics_test` use the external `_test` package
convention so they can import both their own package and `keyboard`
without cycles; `keyboard_test` stays in-package so it can exercise
unexported helpers.

---

## Status / scope

Toy project written to accompany the blog post linked above. No SemVer, no
stability guarantees. Everything lives under `internal/`, so nothing is
importable from outside this module by design — if you want to reuse the
types, fork and lift them into `pkg/` under your own module path.

One honest caveat: A\* with the default Manhattan-in-layer heuristic can
return a suboptimal plan on multi-layer passwords (e.g. strings that mix
letters and digits or upper/lower case on QWERTY). The heuristic is a
deliberately loose cross-layer lower bound, and `search` returns as soon as
a successor emits the target rather than waiting for the goal to be popped
from the priority queue — a textbook A\* bug that happens to work out for
Dijkstra because uniform edge weights make the early-exit optimal there.
Dijkstra always returns the optimum. If you care about exact click counts,
stick to `-algo dijkstra` (the default).
