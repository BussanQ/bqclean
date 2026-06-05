package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceScansAndCleansUserTemp(t *testing.T) {
	temp := t.TempDir()
	localAppData := filepath.Join(temp, "Local")
	userTemp := filepath.Join(localAppData, "Temp")
	target := filepath.Join(userTemp, "cache.tmp")

	if err := os.MkdirAll(userTemp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LOCALAPPDATA", localAppData)
	service := NewService()

	scan, err := service.Scan(context.Background(), ScanOptions{Categories: []CleanCategory{CategoryUserTemp}})
	if err != nil {
		t.Fatal(err)
	}
	if scan.TotalCount != 1 {
		t.Fatalf("expected one scanned item, got %d: %#v", scan.TotalCount, scan.Items)
	}
	if scan.Items[0].Path != target {
		t.Fatalf("expected target path %q, got %q", target, scan.Items[0].Path)
	}

	clean, err := service.Clean(context.Background(), CleanRequest{
		TaskID:  scan.TaskID,
		ItemIDs: []string{scan.Items[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if clean.DeletedCount != 1 || clean.DeletedBytes != 5 {
		t.Fatalf("unexpected clean result: %#v", clean)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target to be deleted, stat err: %v", err)
	}
}
