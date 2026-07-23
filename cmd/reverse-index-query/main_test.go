package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"project_cat_reverse/internal/event"
	"project_cat_reverse/internal/result"
)

func TestLimitMatchedIDsBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		count         int
		wantCount     int
		wantTruncated bool
	}{
		{
			name:          "zero",
			count:         0,
			wantCount:     0,
			wantTruncated: false,
		},
		{
			name:          "below limit",
			count:         999,
			wantCount:     999,
			wantTruncated: false,
		},
		{
			name:          "exact limit",
			count:         1000,
			wantCount:     1000,
			wantTruncated: false,
		},
		{
			name:          "above limit",
			count:         1001,
			wantCount:     1000,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := makeIDs(tt.count)

			got, truncated := limitMatchedIDs(ids, maxMatchedIDs)

			if truncated != tt.wantTruncated {
				t.Fatalf(
					"truncated = %v, want %v",
					truncated,
					tt.wantTruncated,
				)
			}

			if len(got) != tt.wantCount {
				t.Fatalf(
					"len(got) = %d, want %d",
					len(got),
					tt.wantCount,
				)
			}

			want := ids[:tt.wantCount]
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got IDs do not match the expected prefix")
			}
		})
	}
}

func TestLimitMatchedIDsDoesNotModifyInput(t *testing.T) {
	ids := []uint64{1, 2, 3, 4}
	original := append([]uint64(nil), ids...)

	_, _ = limitMatchedIDs(ids, 2)

	if !reflect.DeepEqual(ids, original) {
		t.Fatalf("input changed: got %v, want %v", ids, original)
	}
}

func makeIDs(count int) []uint64 {
	ids := make([]uint64, count)

	for i := range ids {
		ids[i] = uint64(i + 1)
	}

	return ids
}
func TestRunQueryStringForScanAndIndex(t *testing.T) {
	events := []event.Event{
		{
			ID:         1,
			Department: "sales",
			Channel:    "email",
			Severity:   "low",
		},
		{
			ID:         2,
			Department: "hr",
			Channel:    "email",
			Severity:   "high",
		},
		{
			ID:         3,
			Department: "sales",
			Channel:    "slack",
			Severity:   "high",
		},
		{
			ID:         4,
			Department: "it",
			Channel:    "slack",
			Severity:   "low",
		},
	}

	tests := []struct {
		name       string
		expression string
		wantIDs    []uint64
	}{
		{
			name:       "or",
			expression: "department=sales OR channel=email",
			wantIDs:    []uint64{1, 2, 3},
		},
		{
			name:       "and has higher priority than or",
			expression: "department=sales OR channel=email AND severity=high",
			wantIDs:    []uint64{1, 2, 3},
		},
		{
			name:       "parentheses change priority",
			expression: "(department=sales OR channel=email) AND severity=high",
			wantIDs:    []uint64{2, 3},
		},
		{
			name:       "mixed case operators and spaces",
			expression: "  department=sales aNd severity=high  ",
			wantIDs:    []uint64{3},
		},
		{
			name:       "no matches",
			expression: "department=finance",
			wantIDs:    []uint64{},
		},
	}

	methods := []string{"scan", "index"}

	for _, tt := range tests {
		for _, method := range methods {
			t.Run(tt.name+"/"+method, func(t *testing.T) {
				eventsPath := writeEventsFile(t, events)
				outputPath := filepath.Join(t.TempDir(), "result.json")

				err := run([]string{
					"run",
					"--events", eventsPath,
					"--query-string", tt.expression,
					"--method", method,
					"--out", outputPath,
				})
				if err != nil {
					t.Fatalf("run() returned error: %v", err)
				}

				got := readResultFile(t, outputPath)

				if got.Method != method {
					t.Fatalf("method = %q, want %q", got.Method, method)
				}

				if got.MatchedCount != len(tt.wantIDs) {
					t.Fatalf(
						"matched_count = %d, want %d",
						got.MatchedCount,
						len(tt.wantIDs),
					)
				}

				if !reflect.DeepEqual(got.MatchedIDs, tt.wantIDs) {
					t.Fatalf(
						"matched_ids = %v, want %v",
						got.MatchedIDs,
						tt.wantIDs,
					)
				}

				if got.Truncated {
					t.Fatal("truncated = true, want false")
				}
			})
		}
	}
}

func TestRunQueryStringErrors(t *testing.T) {
	eventsPath := writeEventsFile(t, []event.Event{
		{
			ID:         1,
			Department: "sales",
		},
	})

	tests := []struct {
		name       string
		expression string
		wantError  string
	}{
		{
			name:       "empty expression",
			expression: "",
			wantError:  "either --query or --query-string is required",
		},
		{
			name:       "missing right operand",
			expression: "department=sales AND",
			wantError:  "expected expression after",
		},
		{
			name:       "missing right operand after or",
			expression: "department=sales OR",
			wantError:  "expected expression after",
		},
		{
			name:       "unknown field",
			expression: "unknown=value",
			wantError:  "unknown field",
		},
		{
			name:       "unclosed parenthesis",
			expression: "(department=sales OR channel=email",
			wantError:  "parenthes",
		},
	}

	for _, tt := range tests {
		for _, method := range []string{"scan", "index"} {
			t.Run(tt.name+"/"+method, func(t *testing.T) {
				outputPath := filepath.Join(t.TempDir(), "result.json")

				err := run([]string{
					"run",
					"--events", eventsPath,
					"--query-string", tt.expression,
					"--method", method,
					"--out", outputPath,
				})
				if err == nil {
					t.Fatal("run() returned nil error")
				}

				if !strings.Contains(
					strings.ToLower(err.Error()),
					strings.ToLower(tt.wantError),
				) {
					t.Fatalf(
						"error = %q, want substring %q",
						err,
						tt.wantError,
					)
				}

				if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
					t.Fatalf(
						"output file was created after error: %v",
						statErr,
					)
				}
			})
		}
	}
}

func writeEventsFile(t *testing.T, events []event.Event) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "events.jsonl")

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create events file: %v", err)
	}

	encoder := json.NewEncoder(file)

	for _, evt := range events {
		if err := encoder.Encode(evt); err != nil {
			_ = file.Close()
			t.Fatalf("encode event: %v", err)
		}
	}

	if err := file.Close(); err != nil {
		t.Fatalf("close events file: %v", err)
	}

	return path
}

func readResultFile(t *testing.T, path string) result.Result {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result file: %v", err)
	}

	var got result.Result

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode result file: %v", err)
	}

	if got.MatchedIDs == nil {
		got.MatchedIDs = []uint64{}
	}

	return got
}
