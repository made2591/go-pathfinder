package metrics_test

import (
	"math"
	"testing"

	"github.com/made2591/go-pathfinder/internal/keyboard"
	"github.com/made2591/go-pathfinder/internal/metrics"
)

// ---------------------------------------------------------------------------
// Entropy
// ---------------------------------------------------------------------------

func TestEntropy_AllSame(t *testing.T) {
	if got := metrics.Entropy("aaaaaa"); got != 0 {
		t.Errorf("Entropy(all-same) = %v, want 0", got)
	}
}

func TestEntropy_TwoSymbols(t *testing.T) {
	if got := metrics.Entropy("ababab"); math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Entropy(two equal symbols) = %v, want 1.0", got)
	}
}

func TestEntropy_Uniform(t *testing.T) {
	got := metrics.Entropy("abcdef")
	want := math.Log2(6)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Entropy(uniform 6) = %v, want %v", got, want)
	}
}

func TestEntropy_Empty(t *testing.T) {
	if got := metrics.Entropy(""); got != 0 {
		t.Errorf("Entropy(\"\") = %v, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// Dispersion
// ---------------------------------------------------------------------------

func TestDispersion_AllSame(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	if got := metrics.Dispersion("aaaaaa", l); got != 0 {
		t.Errorf("Dispersion(all-same) = %v, want 0", got)
	}
}

func TestDispersion_Adjacent(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	if got := metrics.Dispersion("qwerty", l); math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Dispersion(qwerty) = %v, want ~1.0", got)
	}
}

func TestDispersion_Short(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	if got := metrics.Dispersion("a", l); got != 0 {
		t.Errorf("Dispersion single char = %v, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// Diameter
// ---------------------------------------------------------------------------

func TestDiameter_QWERTY_WrapNone(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	l.Wrap = keyboard.WrapNone
	if got := metrics.Diameter(l); got != 12 {
		t.Errorf("Diameter(WrapNone) = %d, want 12", got)
	}
}

func TestDiameter_QWERTY_WrapRow(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	l.Wrap = keyboard.WrapRow
	if got := metrics.Diameter(l); got != 7 {
		t.Errorf("Diameter(WrapRow) = %d, want 7", got)
	}
}

func TestDiameter_QWERTY_WrapGrid(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	l.Wrap = keyboard.WrapGrid
	if got := metrics.Diameter(l); got != 6 {
		t.Errorf("Diameter(WrapGrid) = %d, want 6", got)
	}
}

func TestDiameter_Alphabetical_WrapNone(t *testing.T) {
	l, _ := keyboard.LoadLayout("alphabetical")
	if got := metrics.Diameter(l); got != 9 {
		t.Errorf("Diameter(alphabetical, WrapNone) = %d, want 9", got)
	}
}

// ---------------------------------------------------------------------------
// Psi
// ---------------------------------------------------------------------------

func TestPsi_QWERTY_qwerty(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	psi := metrics.Psi("qwerty", l, 11)
	if psi < 0.200 || psi > 0.202 {
		t.Errorf("Psi(qwerty,11) = %.4f, want ~0.201", psi)
	}
}

func TestPsi_ZeroCost(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	if got := metrics.Psi("qwerty", l, 0); got != 0 {
		t.Errorf("Psi(cost=0) = %v, want 0", got)
	}
}

func TestPsi_EmptyString(t *testing.T) {
	l, _ := keyboard.LoadLayout("qwerty")
	if got := metrics.Psi("", l, 0); got != 0 {
		t.Errorf("Psi(\"\") = %v, want 0", got)
	}
}
