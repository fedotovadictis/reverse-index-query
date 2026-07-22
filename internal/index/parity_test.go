package index

import (
	"slices"
	"testing"

	"project_cat_reverse/internal/query"
	"project_cat_reverse/internal/reader"
	"project_cat_reverse/internal/scan"
)

func TestIndexParity(t *testing.T) {
	events, err := reader.ReadEvents("../../testdata/control/events.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	idx := NewIndex()
	idx.Build(events)
	idx.Sort()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "term",
			query: "department=dev",
		},
		{
			name:  "and",
			query: "department=dev AND file_ext=pdf",
		},
		{
			name:  "or",
			query: "department=dev OR department=qa",
		},
		{
			name:  "parentheses",
			query: "(department=dev OR department=qa) AND file_ext=pdf",
		},
		{
			name:  "no matches",
			query: "department=unknown",
		},
		{
			name:  "complex",
			query: "department=dev AND (file_ext=pdf OR channel=email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.ParseString(tt.query)
			if err != nil {
				t.Fatal(err)
			}

			scanIDs, err := scan.Execute(events, &q)
			if err != nil {
				t.Fatal(err)
			}

			indexIDs, err := idx.Execute(&q)
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(scanIDs, indexIDs) {
				t.Fatalf(
					"query %q\nscan=%v\nindex=%v",
					tt.query,
					scanIDs,
					indexIDs,
				)
			}
		})
	}
}
