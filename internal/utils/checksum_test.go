package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateChecksum_FileDeterministic(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(filePath, []byte("hello shipyard"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	first, err := CalculateChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to calculate checksum: %v", err)
	}

	second, err := CalculateChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to calculate checksum on second pass: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic checksum, got %q and %q", first, second)
	}
}

func TestCalculateChecksum_DirectoryChangesWhenContentsChange(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(filePath, []byte("version-one"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	first, err := CalculateChecksum(dir)
	if err != nil {
		t.Fatalf("failed to calculate checksum: %v", err)
	}

	if err := os.WriteFile(filePath, []byte("version-two"), 0o644); err != nil {
		t.Fatalf("failed to rewrite file: %v", err)
	}

	second, err := CalculateChecksum(dir)
	if err != nil {
		t.Fatalf("failed to calculate checksum after change: %v", err)
	}

	if first == second {
		t.Fatalf("expected checksum to change when directory contents change")
	}
}
