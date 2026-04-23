package main

import (
	"math"
	"testing"
)

func TestEntropy_AllSame(t *testing.T) {
	if got := Entropy("aaaaaa"); got != 0 {
		t.Errorf("Entropy(all-same) = %v, want 0", got)
	}
}

func TestEntropy_TwoSymbols(t *testing.T) {
	got := Entropy("ababab")
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Entropy(two equal symbols) = %v, want 1.0", got)
	}
}

func TestEntropy_Uniform(t *testing.T) {
	got := Entropy("abcdef")
	want := math.Log2(6)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Entropy(uniform 6 symbols) = %v, want %v", got, want)
	}
}

func TestDispersion_AllSame(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	got := Dispersion("aaaaaa", l)
	if got != 0 {
		t.Errorf("Dispersion(all-same) = %v, want 0", got)
	}
}

// TestDispersion_Adjacent checks that "qwerty" on QWERTY layout yields a mean
// distance close to 1.0 (all consecutive pairs are horizontally adjacent on
// the top row: q-w, w-e, e-r, r-t, t-y — each 1 column apart).
func TestDispersion_Adjacent(t *testing.T) {
	l, err := loadLayout("qwerty")
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	got := Dispersion("qwerty", l)
	// All pairs on QWERTY row 0 are 1 apart, so mean distance should be exactly 1.
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Dispersion(qwerty) = %v, want ~1.0", got)
	}
}
