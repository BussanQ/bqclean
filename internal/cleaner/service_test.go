package cleaner

import (
	"context"
	"errors"
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
	if clean.Failures == nil {
		t.Fatal("expected clean failures to be an empty slice, got nil")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target to be deleted, stat err: %v", err)
	}
}

func TestServiceEvictsOldScanSessions(t *testing.T) {
	temp := t.TempDir()
	localAppData := filepath.Join(temp, "Local")
	if err := os.MkdirAll(filepath.Join(localAppData, "Temp"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", localAppData)
	service := NewService()

	taskIDs := make([]string, 0, maxScanSessions+1)
	for i := 0; i < maxScanSessions+1; i++ {
		scan, err := service.Scan(context.Background(), ScanOptions{Categories: []CleanCategory{CategoryUserTemp}})
		if err != nil {
			t.Fatal(err)
		}
		taskIDs = append(taskIDs, scan.TaskID)
	}

	service.mu.Lock()
	sessionCount := len(service.sessions)
	service.mu.Unlock()
	if sessionCount != maxScanSessions {
		t.Fatalf("expected %d retained sessions, got %d", maxScanSessions, sessionCount)
	}

	if _, err := service.Clean(context.Background(), CleanRequest{TaskID: taskIDs[0], ItemIDs: []string{"x"}}); err == nil {
		t.Fatal("expected clean on evicted session to fail")
	}
	if _, err := service.Clean(context.Background(), CleanRequest{TaskID: taskIDs[len(taskIDs)-1], ItemIDs: nil}); err != nil {
		t.Fatalf("expected clean on latest session to succeed, got %v", err)
	}
}

func TestScanRecycleBinsAcrossDrives(t *testing.T) {
	sizes := map[string]int64{
		`C:\`: 1024,
		`D:\`: 0,
	}
	query := func(root string) (int64, int64, error) {
		if root == `E:\` {
			return 0, 0, errors.New("query failed")
		}
		size := sizes[root]
		count := int64(0)
		if size > 0 {
			count = 3
		}
		return size, count, nil
	}

	items, failures := scanRecycleBins([]string{`C:\`, `D:\`, `E:\`}, query)

	if len(items) != 1 {
		t.Fatalf("expected one non-empty recycle bin item, got %#v", items)
	}
	if items[0].Path != `C:\` || items[0].SizeBytes != 1024 || !items[0].IsVirtual {
		t.Fatalf("unexpected recycle bin item: %#v", items[0])
	}
	if len(failures) != 1 || failures[0].Path != `Recycle Bin (E:\)` {
		t.Fatalf("expected one failure for E:, got %#v", failures)
	}
}
