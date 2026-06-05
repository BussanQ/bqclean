package delete

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cleanapp/internal/cleaner/model"
	"cleanapp/internal/cleaner/rules"
)

func TestDeleteRejectsOutsideAllowedRoot(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "cache")
	outside := filepath.Join(temp, "outside.tmp")

	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	deleter := New(nil)
	deleted, skipped, failure := deleter.Delete(context.Background(), model.ScanItem{
		Path:      outside,
		SizeBytes: 4,
		Category:  model.CategoryUserTemp,
	}, []rules.Root{{Path: root}})

	if deleted {
		t.Fatalf("outside file must not be deleted")
	}
	if !skipped {
		t.Fatalf("outside file should be skipped")
	}
	if failure == nil {
		t.Fatalf("expected safety failure")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file should still exist: %v", err)
	}
}
