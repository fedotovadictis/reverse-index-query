package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateToFileInvalidCount(t *testing.T) {
	err := GenerateToFile(0, "events.jsonl", 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
func TestGenerateToFileEmptyName(t *testing.T) {
	err := GenerateToFile(3, "", 42)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
func TestGenerateToFileCreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	fileName := filepath.Join(tempDir, "events.jsonl")

	err := GenerateToFile(3, fileName, 42)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
}
