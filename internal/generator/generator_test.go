package generator

import (
	"bytes"
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
func TestGenerateToFileSameSeedProducesSameOutput(t *testing.T) {
	tempDir := t.TempDir()
	fileName1 := filepath.Join(tempDir, "events1.jsonl")
	fileName2 := filepath.Join(tempDir, "events2.jsonl")
	err := GenerateToFile(3, fileName1, 42)
	if err != nil {
		t.Fatal(err)
	}
	err = GenerateToFile(3, fileName2, 42)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := os.ReadFile(fileName1)
	if err != nil {
		t.Fatal(err)
	}

	data2, err := os.ReadFile(fileName2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data1, data2) {
		t.Fatal("GenerateToFile produced different output for the same seed")
	}

}
