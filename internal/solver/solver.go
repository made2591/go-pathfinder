// Package solver provides Pathfinder implementations (Dijkstra and A*) for
// the keyboard state graph. It depends on keyboard for the graph types.
package solver

import (
	"fmt"
	"slices"

	"github.com/made2591/go-pathfinder/internal/keyboard"
)

// LoadFinder returns a built-in pathfinder by name.
func LoadFinder(name string) (keyboard.Pathfinder, error) {
	switch name {
	case "dijkstra":
		return Dijkstra{}, nil
	case "astar":
		return AStar{}, nil
	default:
		return nil, fmt.Errorf("unknown algorithm %q", name)
	}
}

// Dijkstra is the uninformed baseline. Every edge has cost 1, so it is also
// equivalent to a BFS, but we keep the priority queue so the same scaffold
// supports non-uniform costs later.
type Dijkstra struct{}

func (Dijkstra) Name() string { return "dijkstra" }

func (Dijkstra) Find(l *keyboard.Layout, start keyboard.State, target rune) ([]keyboard.Step, keyboard.State, error) {
	return search(l, start, target, func(keyboard.State) int { return 0 })
}

// AStar is informed by a heuristic that lower-bounds the remaining cost. The
// default heuristic is the Manhattan distance to the nearest key in the
// current layer that emits the target (or 0 if the target isn't reachable
// without a layer switch — a deliberately loose lower bound).
type AStar struct {
	Heuristic func(l *keyboard.Layout, s keyboard.State, target rune) int
}

func (AStar) Name() string { return "astar" }

func (a AStar) Find(l *keyboard.Layout, start keyboard.State, target rune) ([]keyboard.Step, keyboard.State, error) {
	h := a.Heuristic
	if h == nil {
		h = func(l *keyboard.Layout, s keyboard.State, target rune) int { return manhattanInLayer(l, s, target) }
	}
	return search(l, start, target, func(s keyboard.State) int { return h(l, s, target) })
}

// search is the shared core for Dijkstra and A*. Passing h ≡ 0 yields
// Dijkstra; any admissible h yields A*.
func search(l *keyboard.Layout, start keyboard.State, target rune, h func(keyboard.State) int) ([]keyboard.Step, keyboard.State, error) {
	type bp struct {
		from keyboard.State
		step keyboard.Step
	}
	gScore := map[keyboard.State]int{start: 0}
	prev := map[keyboard.State]bp{}
	open := newMinHeap(func(a, b pqNode) bool { return a.f < b.f })
	open.Push(pqNode{f: h(start), state: start})

	for open.Len() > 0 {
		cur := open.Pop()
		if cur.f-h(cur.state) > gScore[cur.state] {
			continue // stale entry, a cheaper one was already processed
		}
		for _, su := range l.Successors(cur.state) {
			if su.Step.Emitted != 0 && su.Step.Emitted != target {
				continue // emitting any other rune is a dead end for this sub-search
			}
			gNew := gScore[cur.state] + 1
			if su.Step.Emitted == target {
				path := []keyboard.Step{su.Step}
				for p := cur.state; p != start; {
					b := prev[p]
					path = append(path, b.step)
					p = b.from
				}
				slices.Reverse(path)
				return path, su.Next, nil
			}
			if old, seen := gScore[su.Next]; !seen || gNew < old {
				gScore[su.Next] = gNew
				prev[su.Next] = bp{from: cur.state, step: su.Step}
				open.Push(pqNode{f: gNew + h(su.Next), state: su.Next})
			}
		}
	}
	return nil, keyboard.State{}, fmt.Errorf("rune %q is not typeable on layout %q", target, l.Name)
}

// manhattanInLayer is the default A* heuristic. It returns the Manhattan
// distance to the nearest key in the current layer that would emit target
// given the current caps state. Cross-layer cases return 0, which is
// admissible.
func manhattanInLayer(l *keyboard.Layout, s keyboard.State, target rune) int {
	layer := &l.Layers[s.Layer]
	best := -1
	for r, row := range layer.Keys {
		for c, k := range row {
			if k.Action != keyboard.ActionEmit {
				continue
			}
			emits := k.Glyph
			if s.Caps != keyboard.CapsOff && k.Shifted != 0 {
				emits = k.Shifted
			}
			if emits != target {
				continue
			}
			d := abs(r-s.Row) + abs(c-s.Col)
			if best < 0 || d < best {
				best = d
			}
		}
	}
	if best < 0 {
		return 0
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- generic min-heap --------------------------------------------------------

type pqNode struct {
	f     int // priority: g + h
	state keyboard.State
}

type minHeap[T any] struct {
	data []T
	less func(a, b T) bool
}

func newMinHeap[T any](less func(a, b T) bool) *minHeap[T] {
	return &minHeap[T]{less: less}
}

func (h *minHeap[T]) Len() int { return len(h.data) }

func (h *minHeap[T]) Push(x T) {
	h.data = append(h.data, x)
	h.up(len(h.data) - 1)
}

func (h *minHeap[T]) Pop() T {
	n := len(h.data) - 1
	h.data[0], h.data[n] = h.data[n], h.data[0]
	h.down(0, n)
	x := h.data[n]
	h.data = h.data[:n]
	return x
}

func (h *minHeap[T]) up(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if !h.less(h.data[i], h.data[p]) {
			return
		}
		h.data[i], h.data[p] = h.data[p], h.data[i]
		i = p
	}
}

func (h *minHeap[T]) down(i, n int) {
	for {
		l := 2*i + 1
		if l >= n {
			return
		}
		smallest := l
		if r := l + 1; r < n && h.less(h.data[r], h.data[l]) {
			smallest = r
		}
		if !h.less(h.data[smallest], h.data[i]) {
			return
		}
		h.data[i], h.data[smallest] = h.data[smallest], h.data[i]
		i = smallest
	}
}
