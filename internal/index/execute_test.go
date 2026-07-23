package index

import (
	"project_cat_reverse/internal/event"
	"project_cat_reverse/internal/scan"
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
func TestScanAndIndexParityOnNonStandardData(t *testing.T) {
	tests := []struct {
		name   string
		events []event.Event
		query  *query.Query
		want   []uint64
	}{
		{
			name: "unsorted input",
			events: []event.Event{
				{ID: 30, Department: "sales"},
				{ID: 10, Department: "sales"},
				{ID: 20, Department: "hr"},
				{ID: 5, Department: "sales"},
			},
			query: &query.Query{
				Op:    query.Term,
				Field: "department",
				Value: "sales",
			},
			want: []uint64{5, 10, 30},
		},
		{
			name: "duplicate ids",
			events: []event.Event{
				{ID: 7, Department: "sales"},
				{ID: 3, Department: "sales"},
				{ID: 7, Department: "sales"},
				{ID: 3, Department: "sales"},
			},
			query: &query.Query{
				Op:    query.Term,
				Field: "department",
				Value: "sales",
			},
			want: []uint64{3, 7},
		},
		{
			name: "empty result",
			events: []event.Event{
				{ID: 2, Department: "sales"},
				{ID: 1, Department: "hr"},
			},
			query: &query.Query{
				Op:    query.Term,
				Field: "department",
				Value: "dev",
			},
			want: []uint64{},
		},
		{
			name: "or with duplicate ids",
			events: []event.Event{
				{ID: 4, Department: "sales", Channel: "email"},
				{ID: 2, Department: "sales", Channel: "local"},
				{ID: 4, Department: "sales", Channel: "email"},
				{ID: 1, Department: "hr", Channel: "email"},
			},
			query: &query.Query{
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
			},
			want: []uint64{1, 2, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanIDs, err := scan.Execute(tt.events, tt.query)
			if err != nil {
				t.Fatalf("scan.Execute() returned error: %v", err)
			}

			idx := NewIndex()
			idx.Build(tt.events)
			idx.Sort()

			indexIDs, err := idx.Execute(tt.query)
			if err != nil {
				t.Fatalf("index.Execute() returned error: %v", err)
			}

			if !slices.Equal(scanIDs, indexIDs) {
				t.Fatalf(
					"scan and index differ:\nscan:  %v\nindex: %v",
					scanIDs,
					indexIDs,
				)
			}

			if !slices.Equal(scanIDs, tt.want) {
				t.Fatalf("result = %v, want %v", scanIDs, tt.want)
			}
		})
	}
}
