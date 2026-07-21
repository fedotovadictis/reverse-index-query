package query

import "testing"

func TestParseString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantField string
		wantValue string
	}{
		{
			name:      "without spaces",
			input:     "department=dev",
			wantField: "department",
			wantValue: "dev",
		},
		{
			name:      "with spaces",
			input:     "department = dev",
			wantField: "department",
			wantValue: "dev",
		},
		{
			name:      "user id",
			input:     "user_id=42",
			wantField: "user_id",
			wantValue: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			if q.Op != Term {
				t.Errorf("Op = %q, want %q", q.Op, Term)
			}

			if q.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", q.Field, tt.wantField)
			}

			if q.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", q.Value, tt.wantValue)
			}

			if q.Left != nil {
				t.Errorf("Left = %#v, want nil", q.Left)
			}

			if q.Right != nil {
				t.Errorf("Right = %#v, want nil", q.Right)
			}
		})
	}
}

func TestParseStringErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty query",
			input: "",
		},
		{
			name:  "missing equals",
			input: "department",
		},
		{
			name:  "empty field",
			input: "=dev",
		},
		{
			name:  "empty value",
			input: "department=",
		},
		{
			name:  "unknown field",
			input: "unknown=dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseString(tt.input)
			if err == nil {
				t.Fatalf("ParseString(%q) expected error", tt.input)
			}
		})
	}
}

func TestParseStringAnd(t *testing.T) {
	q, err := ParseString(
		"department=dev AND file_ext=pdf",
	)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if q.Op != And {
		t.Fatalf("Op = %q, want %q", q.Op, And)
	}

	if q.Left == nil {
		t.Fatal("Left is nil")
	}

	if q.Right == nil {
		t.Fatal("Right is nil")
	}

	if q.Left.Op != Term {
		t.Errorf("Left.Op = %q, want %q", q.Left.Op, Term)
	}

	if q.Left.Field != "department" {
		t.Errorf(
			"Left.Field = %q, want %q",
			q.Left.Field,
			"department",
		)
	}

	if q.Left.Value != "dev" {
		t.Errorf(
			"Left.Value = %q, want %q",
			q.Left.Value,
			"dev",
		)
	}

	if q.Right.Op != Term {
		t.Errorf("Right.Op = %q, want %q", q.Right.Op, Term)
	}

	if q.Right.Field != "file_ext" {
		t.Errorf(
			"Right.Field = %q, want %q",
			q.Right.Field,
			"file_ext",
		)
	}

	if q.Right.Value != "pdf" {
		t.Errorf(
			"Right.Value = %q, want %q",
			q.Right.Value,
			"pdf",
		)
	}
}

func TestParseStringMultipleAnd(t *testing.T) {
	q, err := ParseString(
		"department=dev AND file_ext=pdf AND channel=email",
	)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if q.Op != And {
		t.Fatalf("Op = %q, want %q", q.Op, And)
	}

	if q.Left == nil || q.Right == nil {
		t.Fatal("AND operands must not be nil")
	}

	if q.Right.Op != Term {
		t.Fatalf("Right.Op = %q, want %q", q.Right.Op, Term)
	}

	if q.Right.Field != "channel" {
		t.Errorf(
			"Right.Field = %q, want %q",
			q.Right.Field,
			"channel",
		)
	}

	if q.Right.Value != "email" {
		t.Errorf(
			"Right.Value = %q, want %q",
			q.Right.Value,
			"email",
		)
	}
}
