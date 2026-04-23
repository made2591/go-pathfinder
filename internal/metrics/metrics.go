// Package metrics implements the typing-complexity numbers from the
// companion blog post: Shannon entropy H, topological dispersion T, layer
// diameter D_L, and the combined Psi score H * (T / D_L) / (cost / |s|).
package metrics

import (
	"fmt"
	"math"

	"github.com/made2591/go-pathfinder/internal/keyboard"
)

// Entropy returns the Shannon entropy of the empirical character distribution
// in s, measured in bits: -Σ p_c * log2(p_c).
// Returns 0 for strings with 0 or 1 unique characters.
func Entropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[rune]int)
	total := 0
	for _, r := range s {
		freq[r]++
		total++
	}
	var h float64
	for _, count := range freq {
		p := float64(count) / float64(total)
		h -= p * math.Log2(p)
	}
	return h
}

// Dispersion returns the mean graph distance on layer 0 between consecutive
// characters of s, using l.Wrap for movement edges (weight 1 per step).
//
// For a character not present on layer 0 (e.g. a digit on QWERTY's letters
// layer), the character pair is skipped and does not contribute to the mean.
// Only Glyph is considered; Shifted is intentionally ignored.
func Dispersion(s string, l *keyboard.Layout) float64 {
	runes := []rune(s)
	if len(runes) < 2 {
		return 0
	}

	layer := &l.Layers[0]
	pos := make(map[rune][2]int)
	for r, row := range layer.Keys {
		for c, k := range row {
			if k.Action == keyboard.ActionEmit {
				if _, exists := pos[k.Glyph]; !exists {
					pos[k.Glyph] = [2]int{r, c}
				}
			}
		}
	}

	var total float64
	count := 0
	for i := 0; i < len(runes)-1; i++ {
		a, b := runes[i], runes[i+1]
		pa, aOk := pos[a]
		pb, bOk := pos[b]
		if !aOk || !bOk {
			continue
		}
		d := math.Abs(float64(pa[0]-pb[0])) + math.Abs(float64(pa[1]-pb[1]))
		total += d
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// Diameter returns the BFS diameter of layer 0 under the given wrap mode.
// Only movement edges (Up/Down/Left/Right) are used; OK is excluded.
// Each edge has weight 1.
func Diameter(l *keyboard.Layout) int {
	layer := &l.Layers[0]
	rows, cols := layer.Rows(), layer.Cols()
	total := rows * cols

	idx := func(r, c int) int { return r*cols + c }

	bfs := func(startR, startC int) int {
		dist := make([]int, total)
		for i := range dist {
			dist[i] = -1
		}
		queue := make([]int, 0, total)
		start := idx(startR, startC)
		dist[start] = 0
		queue = append(queue, start)

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			cr, cc := cur/cols, cur%cols
			s := keyboard.State{Layer: 0, Row: cr, Col: cc}
			for m := keyboard.MoveUp; m < keyboard.MoveOK; m++ {
				nr, nc, ok := l.MoveCursor(s, m)
				if !ok {
					continue
				}
				ni := idx(nr, nc)
				if dist[ni] < 0 {
					dist[ni] = dist[cur] + 1
					queue = append(queue, ni)
				}
			}
		}

		max := 0
		for _, d := range dist {
			if d > max {
				max = d
			}
		}
		return max
	}

	diameter := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if e := bfs(r, c); e > diameter {
				diameter = e
			}
		}
	}
	return diameter
}

// Psi computes the typing complexity metric: H * (T / D_L) / (cost / |s|).
func Psi(s string, l *keyboard.Layout, cost int) float64 {
	h := Entropy(s)
	t := Dispersion(s, l)
	dL := float64(Diameter(l))
	if dL == 0 || cost == 0 || len([]rune(s)) == 0 {
		return 0
	}
	tNorm := t / dL
	costPerChar := float64(cost) / float64(len([]rune(s)))
	return h * tNorm / costPerChar
}

// PrintMetrics writes the article-format metrics block to stdout.
func PrintMetrics(text string, l *keyboard.Layout, cost int) {
	h := Entropy(text)
	t := Dispersion(text, l)
	dL := Diameter(l)
	tNorm := 0.0
	if dL > 0 {
		tNorm = t / float64(dL)
	}
	psi := Psi(text, l, cost)

	fmt.Printf("H  (Shannon entropy):          %.3f bits\n", h)
	fmt.Printf("T  (topological dispersion):   %.3f\n", t)
	fmt.Printf("T̃  (normalised = T / D_L):     %.3f   (D_L = %d)\n", tNorm, dL)
	fmt.Printf("Ψ  (typing complexity):        %.3f\n", psi)
}
