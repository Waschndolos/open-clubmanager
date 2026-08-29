package handlers

import "testing"

func TestDatabaseDirectory(t *testing.T) {
	t.Run("relative sqlite path", func(t *testing.T) {
		dir := databaseDirectory("./data/clubmanager.db")
		if dir != "data" {
			t.Fatalf("expected data, got %q", dir)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		dir := databaseDirectory("")
		if dir != "." {
			t.Fatalf("expected '.', got %q", dir)
		}
	})
}

func TestResolveStoredDocumentPath(t *testing.T) {
	path, err := resolveStoredDocumentPath("./data", "documents/file.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "data/documents/file.pdf" {
		t.Fatalf("unexpected path: %q", path)
	}

	if _, err := resolveStoredDocumentPath("./data", "../escape.txt"); err == nil {
		t.Fatal("expected traversal path to fail")
	}
}

func TestSanitizeFilename(t *testing.T) {
	got := sanitizeFilename("../test report.pdf")
	if got != "test_report.pdf" {
		t.Fatalf("unexpected sanitized filename: %q", got)
	}
}
