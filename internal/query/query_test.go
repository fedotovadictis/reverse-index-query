package query

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   Query
		wantErr bool
	}{
		{
			name: "valid term",
			query: Query{
				Op:    Term,
				Field: "department",
				Value: "sales",
			},
			wantErr: false,
		},
		{
			name: "empty field",
			query: Query{
				Op:    Term,
				Field: "",
				Value: "sales",
			},
			wantErr: true,
		},
		{
			name: "invalid field",
			query: Query{
				Op:    Term,
				Field: "salary",
				Value: "100",
			},
			wantErr: true,
		},
		{
			name: "and missing left",
			query: Query{
				Op: And,
				Right: &Query{
					Op:    Term,
					Field: "department",
					Value: "sales",
				},
			},
			wantErr: true,
		},
		{
			name: "or missing right",
			query: Query{
				Op: Or,
				Left: &Query{
					Op:    Term,
					Field: "department",
					Value: "sales",
				},
			},
			wantErr: true,
		},
		{
			name: "unknown operator",
			query: Query{
				Op: "AAA",
			},
			wantErr: true,
		},
		{
			name: "valid and",
			query: Query{
				Op: And,
				Left: &Query{
					Op:    Term,
					Field: "department",
					Value: "sales",
				},
				Right: &Query{
					Op:    Term,
					Field: "channel",
					Value: "email",
				},
			},
			wantErr: false,
		},
		{
			name: "valid or",
			query: Query{
				Op: Or,
				Left: &Query{
					Op:    Term,
					Field: "department",
					Value: "sales",
				},
				Right: &Query{
					Op:    Term,
					Field: "channel",
					Value: "email",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
func TestValidateNilQuery(t *testing.T) {
	var q *Query
	err := q.Validate()
	if err == nil {
		t.Error("expected error, got nil")
	}
}
func TestReadQueryWithBOM(t *testing.T) {
	content := append(
		[]byte{0xEF, 0xBB, 0xBF},
		[]byte(`{"op":"TERM","field":"department","value":"dev"}`)...,
	)

	fileName := filepath.Join(t.TempDir(), "query.json")

	if err := os.WriteFile(fileName, content, 0644); err != nil {
		t.Fatal(err)
	}

	q, err := ReadQuery(fileName)
	if err != nil {
		t.Fatalf("ReadQuery returned error: %v", err)
	}

	if q.Op != Term || q.Field != "department" || q.Value != "dev" {
		t.Fatalf("unexpected query: %+v", q)
	}
}
