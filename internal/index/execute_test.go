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
func TestExecuteAnd(t *testing.T) {
	idx := NewIndex()

	idx.AddEvent(event.Event{
		ID:         1,
		Department: "sales",
		Channel:    "email",
	})
	idx.AddEvent(event.Event{
		ID:         2,
		Department: "sales",
		Channel:    "local",
	})
	idx.AddEvent(event.Event{
		ID:         3,
		Department: "hr",
		Channel:    "email",
	})

	q := &query.Query{
		Op: query.And,
		Left: &query.Query{
			Op:    query.Term,
			Field: "department",
			Value: "sales",
		},
		Right: &query.Query{
			Op:    query.Term,
			Field: "channel",
			Value: "email",
		},
	}

	got, err := idx.Execute(q)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	expected := []uint64{1}

	if !slices.Equal(got, expected) {
		t.Errorf("Execute() = %v, want %v", got, expected)
	}
}

func TestExecuteOr(t *testing.T) {
	idx := NewIndex()

	idx.AddEvent(event.Event{
		ID:         1,
		Department: "sales",
		Channel:    "email",
	})
	idx.AddEvent(event.Event{
		ID:         2,
		Department: "sales",
		Channel:    "local",
	})
	idx.AddEvent(event.Event{
		ID:         3,
		Department: "hr",
		Channel:    "email",
	})

	q := &query.Query{
		Op: query.Or,
		Left: &query.Query{
			Op:    query.Term,
			Field: "department",
			Value: "sales",
		},
		Right: &query.Query{
			Op:    query.Term,
			Field: "channel",
			Value: "email",
		},
	}

	got, err := idx.Execute(q)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	expected := []uint64{1, 2, 3}
	if !slices.Equal(got, expected) {
		t.Errorf("Execute() = %v, want %v", got, expected)
	}
}
func TestSortPostingListsByLength(t *testing.T) {
	postingLists := [][]uint64{
		{1, 2, 3, 4, 5},
		{10},
		{20, 21, 22},
		{30, 31},
	}

	sortPostingListsByLength(postingLists)

	gotLengths := make([]int, len(postingLists))
	for i, ids := range postingLists {
		gotLengths[i] = len(ids)
	}

	wantLengths := []int{1, 2, 3, 5}

	if !slices.Equal(gotLengths, wantLengths) {
		t.Fatalf("posting list lengths = %v, want %v", gotLengths, wantLengths)
	}
}
