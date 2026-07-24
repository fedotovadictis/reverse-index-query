package reader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadEventsMalformedJSONIncludesLineNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	data := strings.Join([]string{
		`{"id":1,"department":"sales"}`,
		`{"id":2,"department":`,
		`{"id":3,"department":"dev"}`,
	}, "\n")

	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write events file: %v", err)
	}

	_, err := ReadEvents(path)
	if err == nil {
		t.Fatal("ReadEvents() returned nil error")
	}

	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf(
			"error = %q, want line number 2",
			err,
		)
	}
}

func TestReadEventsTooLongLineIncludesLineNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	longValue := strings.Repeat("x", 70*1024)

	data := strings.Join([]string{
		`{"id":1,"department":"sales"}`,
		`{"id":2,"department":"` + longValue + `"}`,
	}, "\n")

	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write events file: %v", err)
	}

	_, err := ReadEvents(path)
	if err == nil {
		t.Fatal("ReadEvents() returned nil error")
	}

	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf(
			"error = %q, want line number 2",
			err,
		)
	}
}
