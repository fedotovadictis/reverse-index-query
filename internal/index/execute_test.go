package index

import (
	"project_cat_reverse/internal/event"
	"slices"
	"testing"

	"project_cat_reverse/internal/query"
)

func TestExecuteTerm(t *testing.T) {
	idx := NewIndex()

	idx.AddEvent(event.Event{
		ID:         1,
		Department: "sales",
	})

	idx.AddEvent(event.Event{
		ID:         2,
		Department: "it",
	})

	idx.AddEvent(event.Event{
		ID:         3,
		Department: "sales",
	})
	// добавить несколько событий

	q := &query.Query{
		Op:    query.Term,
		Field: "department",
		Value: "sales",
	}

	got, err := idx.Execute(q)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	expected := []uint64{1, 3}

	if !slices.Equal(got, expected) {
		t.Errorf("Execute() = %v, want %v", got, expected)
	}
}
