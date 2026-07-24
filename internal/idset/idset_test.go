package idset

import (
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	got := Normalize([]uint64{7, 3, 7, 1, 3})
	want := []uint64{1, 3, 7}

	if !slices.Equal(got, want) {
		t.Fatalf("Normalize() = %v, want %v", got, want)
	}
}

func TestUniqueSorted(t *testing.T) {
	got := UniqueSorted([]uint64{1, 1, 3, 3, 7})
	want := []uint64{1, 3, 7}

	if !slices.Equal(got, want) {
		t.Fatalf("UniqueSorted() = %v, want %v", got, want)
	}
}
