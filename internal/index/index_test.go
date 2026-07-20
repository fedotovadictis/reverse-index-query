package index

import (
	"project_cat_reverse/internal/event"
	"slices"
	"testing"
)

func TestIntersect(t *testing.T) {
	tests := []struct {
		name     string
		left     []uint64
		right    []uint64
		expected []uint64
	}{
		{
			name:     "partial overlap",
			left:     []uint64{1, 3, 5, 7},
			right:    []uint64{2, 3, 6, 7},
			expected: []uint64{3, 7},
		},
		{
			name:     "no common elements",
			left:     []uint64{1, 3, 5},
			right:    []uint64{2, 4, 6},
			expected: []uint64{},
		},
		{
			name:     "equal lists",
			left:     []uint64{1, 2, 3},
			right:    []uint64{1, 2, 3},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "empty left",
			left:     []uint64{},
			right:    []uint64{1, 2, 3},
			expected: []uint64{},
		},
		{
			name:     "empty right",
			left:     []uint64{1, 2, 3},
			right:    []uint64{},
			expected: []uint64{},
		},
		{
			name:     "both empty",
			left:     []uint64{},
			right:    []uint64{},
			expected: []uint64{},
		},
		{
			name:     "single common element",
			left:     []uint64{1, 4, 8},
			right:    []uint64{4},
			expected: []uint64{4},
		},
		{
			name:     "one list contained in another",
			left:     []uint64{1, 2, 3, 4, 5},
			right:    []uint64{2, 3, 4},
			expected: []uint64{2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Intersect(tt.left, tt.right)

			if !slices.Equal(got, tt.expected) {
				t.Errorf("Intersect() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAndWithDifferentPostingListSizes(t *testing.T) {
	idx := NewIndex()

	idx.AddEvent(event.Event{
		ID:         1,
		Department: "sales",
		Channel:    "email",
	})
	idx.AddEvent(event.Event{
		ID:         2,
		Department: "sales",
		Channel:    "web",
	})
	idx.AddEvent(event.Event{
		ID:         3,
		Department: "sales",
		Channel:    "web",
	})
	idx.AddEvent(event.Event{
		ID:         4,
		Department: "sales",
		Channel:    "web",
	})

	got := idx.And(
		"department",
		"sales",
		"channel",
		"email",
	)

	expected := []uint64{1}

	if !slices.Equal(got, expected) {
		t.Errorf("And() = %v, want %v", got, expected)
	}
}

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		left     []uint64
		right    []uint64
		expected []uint64
	}{
		{
			name:     "partial overlap",
			left:     []uint64{1, 3, 5},
			right:    []uint64{2, 3, 4},
			expected: []uint64{1, 2, 3, 4, 5},
		},
		{
			name:     "no common elements",
			left:     []uint64{1, 4, 5},
			right:    []uint64{2, 3, 6},
			expected: []uint64{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "equal lists",
			left:     []uint64{1, 2, 3},
			right:    []uint64{1, 2, 3},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "empty left",
			left:     []uint64{},
			right:    []uint64{1, 2, 5},
			expected: []uint64{1, 2, 5},
		},
		{
			name:     "empty right",
			left:     []uint64{1, 2, 3},
			right:    []uint64{},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "both empty",
			left:     []uint64{},
			right:    []uint64{},
			expected: []uint64{},
		},
		{
			name:     "right inside left",
			left:     []uint64{1, 2, 3, 4, 5},
			right:    []uint64{2, 3},
			expected: []uint64{1, 2, 3, 4, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Union(tt.left, tt.right)
			if !slices.Equal(got, tt.expected) {
				t.Errorf("Union() = %v, want %v", got, tt.expected)

			}
		})
	}
}
func TestOr(t *testing.T) {
	idx := NewIndex()
	idx.AddEvent(event.Event{
		ID:         1,
		Department: "sales",
	})
	idx.AddEvent(event.Event{
		ID:      2,
		Channel: "email",
	})
	idx.AddEvent(event.Event{
		ID:         3,
		Department: "sales",
		Channel:    "email",
	})

	got := idx.Or(
		"department",
		"sales",
		"channel",
		"email",
	)

	expected := []uint64{1, 2, 3}

	if !slices.Equal(got, expected) {
		t.Errorf("Or() = %v, want %v", got, expected)
	}
}
