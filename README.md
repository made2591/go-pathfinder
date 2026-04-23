# go-pathfinder - Find the cheapest click sequence to type on a TV remote

[![Go Report Card](https://goreportcard.com/badge/github.com/made2591/go-pathfinder)](https://goreportcard.com/report/github.com/made2591/go-pathfinder)
[![Go Version](https://img.shields.io/github/go-mod/go-version/made2591/go-pathfinder.svg)](https://golang.org/)
[![License](https://img.shields.io/github/license/made2591/go-pathfinder.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/made2591/go-pathfinder.svg)](https://pkg.go.dev/github.com/made2591/go-pathfinder)

![demo](./demo.gif)

**go-pathfinder** is a CLI that models on-screen TV-remote keyboards as a state graph and finds the cheapest click sequence to type any given string. A related [blog-post](https://madeddu.xyz/posts/go-pathfinder/) explains the motivation, the modelling decisions, and the typing-complexity metric Ψ in detail.

The problem is the one you meet every time you set up a Wi-Fi password on a smart TV: you have four arrow keys and an OK button, an on-screen keyboard with a QWERTY-ish grid of letters, maybe a `123` button that switches to a digits layer, a `#+=` button for symbols, a caps-lock toggle — and a password that contains at least one of each. The "type one character" cost is not *one click per glyph*; it is *one to fifteen clicks, depending on where the cursor sits and where you need it to go*.

Formally, the remote exposes five inputs,

```sh
↑ ↓ ← → OK
```

and the cursor lives on a keyboard described as a layered grid. The natural-but-wrong model is a graph of keys. The right model is a graph of cursor states,

```sh
s = (layer, row, col, capsMode)
```

with five edges leaving each state: four directional moves (constrained by the layout's wrap policy) and one OK edge whose effect depends on the key under the cursor — emit a glyph, toggle caps, or jump to another layer. Each edge has uniform cost 1 — a single physical click. Typing a string of length `n` is `n` shortest-path searches stitched together.

> *Given a layout L, a password s, and a cursor starting state s₀, find the click sequence c₁, c₂, …, cₖ that emits s in order while minimising k.*

Under the hood the repo ships three layouts (QWERTY, alphabetical, Apple-TV single row), three wrap policies (none, row, grid), a sticky-caps FSM, both Dijkstra and A* solvers, the typing-complexity metric Ψ (H · T̃ / cost per char), and an animated `-sim` mode that replays the optimal plan in the terminal.

## Table of Contents

- [go-pathfinder - Find the cheapest click sequence to type on a TV remote](#go-pathfinder---find-the-cheapest-click-sequence-to-type-on-a-tv-remote)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Project layout](#project-layout)
  - [Build and test](#build-and-test)
  - [Support](#support)
  - [Contributing](#contributing)

## Installation

Clone and build from source:

```sh
git clone https://github.com/made2591/go-pathfinder.git
cd go-pathfinder
make build
```

This produces a `./go-pathfinder` binary at the repo root. No third-party dependencies — standard library only.

## Usage

Type a password on the default QWERTY layout with Dijkstra,

```sh
./go-pathfinder -text "Hello"
```

Watch the cursor work in animated mode on Apple TV's single-row keyboard,

```sh
./go-pathfinder -layout appletv -text "hello" -sim -speed 80
```

Try the alphabetical grid with a non-default wrap policy and print every move,

```sh
./go-pathfinder -layout alphabetical -wrap grid -text "abc" -v
```

Print the typing-complexity metrics alongside the click count,

```sh
./go-pathfinder -text "qwerty" -metrics
```

which produces

```sh
layout: QWERTY | algo: dijkstra | text: "qwerty"
clicks (= password cost): 11  (1.83 per character)
H  (Shannon entropy):          2.585 bits
T  (topological dispersion):   1.000
T̃  (normalised = T / D_L):     0.143   (D_L = 7)
Ψ  (typing complexity):        0.201
```

These are the same numbers the [blog post](https://madeddu.xyz/posts/go-pathfinder/) derives by hand — the CLI is what I used to validate them.

The full flag reference:

| Flag | Default | Purpose | Values |
|------|---------|---------|--------|
| `-layout` | `qwerty` | Keyboard layout | `qwerty`, `alphabetical`, `appletv` |
| `-algo` | `dijkstra` | Pathfinding algorithm | `dijkstra`, `astar` |
| `-wrap` | _(layout default)_ | Override the wrap policy | `none`, `row`, `grid` |
| `-text` | _(empty)_ | String to type; renders the layout only when empty | any string |
| `-v` | `false` | Print every move with running cursor state | flag |
| `-sim` | `false` | Animate the cursor in-place (requires `-text`) | flag |
| `-speed` | `250` | Per-step delay in milliseconds for `-sim` | integer ≥ 0 |
| `-metrics` | `false` | Print H, T, T̃, Ψ alongside the click count | flag |

A honest caveat: A* with the default Manhattan-in-layer heuristic can return a suboptimal plan on multi-layer inputs (e.g. strings that mix letters and digits on QWERTY). The blog post discusses exactly why. If you need exact click counts, stick to the default `-algo dijkstra`.

## Project layout

```sh
cmd/
  go-pathfinder/    main.go        — CLI entry: flag parsing and dispatch only
internal/
  keyboard/         layout.go      — Layout, Layer, Key, WrapMode, presets,
                                     ParseWrap, LoadLayout, Render
                    graph.go       — State, Step, Successor, Pathfinder interface,
                                     (layer, row, col, capsMode) state graph,
                                     sticky-caps FSM, Type
  solver/           solver.go      — Dijkstra and A* implementing Pathfinder,
                                     shared search core, inline min-heap
  metrics/          metrics.go     — Entropy, Dispersion, Diameter, Psi,
                                     PrintMetrics
  sim/              sim.go         — in-terminal animation via ANSI escapes,
                                     non-TTY static fallback
```

Dependency direction, one-way:

```sh
keyboard  →  (no internal deps)
solver    →  keyboard
metrics   →  keyboard
sim       →  keyboard
cmd/...   →  keyboard, solver, metrics, sim
```

The `Pathfinder` interface lives in `keyboard` so `Layout.Type(text, finder)` can take it without creating a `solver → keyboard → solver` cycle. Everything outside `cmd/` is under `internal/`, so nothing is importable from outside this module by design — fork and lift into `pkg/` under your own module path if you need to reuse the types.

## Build and test

```sh
make build          # compile ./cmd/go-pathfinder to ./go-pathfinder
make test           # go test ./... -count=1
make fmt            # gofmt -s -w .
make lint           # golangci-lint run, or falls back to go vet
make fix            # fmt plus golangci-lint --fix
make clean          # remove the compiled binary
make coverage       # per-function coverage summary on all packages
make coverage-html  # same, plus open coverage.html in the browser
```

Tests live next to the code they cover. `solver_test` and `metrics_test` use the external `_test` package convention so they can import both their own package and `keyboard` without import cycles; `keyboard_test` stays in-package so it can reach the unexported helpers.

## Support

Please [open an issue](https://github.com/made2591/go-pathfinder/issues/new) for support.

## Contributing

Please contribute using [Github Flow](https://guides.github.com/introduction/flow/). Create a branch, add commits, and [open a pull request](https://github.com/made2591/go-pathfinder/compare/).
