package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeDiskGrowthComparesAgainstPreviousSnapshot(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("CLEANAPP_GROWTH_STORE", filepath.Join(temp, "growth.json"))

	root := filepath.Join(temp, "drive")
	appData := filepath.Join(root, "Users", "tester", "AppData", "Roaming")
	cacheDir := filepath.Join(appData, "Code", "CachedExtensionVSIXs")
	t.Setenv("APPDATA", appData)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "first.bin"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	first, err := service.AnalyzeDiskGrowth(context.Background(), DiskGrowthOptions{Root: root, MaxDepth: 8, MinGrowthBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.HasBaseline {
		t.Fatal("first growth scan should create a baseline")
	}

	if err := os.WriteFile(filepath.Join(cacheDir, "second.bin"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := service.AnalyzeDiskGrowth(context.Background(), DiskGrowthOptions{Root: root, MaxDepth: 8, MinGrowthBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !second.HasBaseline {
		t.Fatal("second growth scan should compare against baseline")
	}

	found := false
	for _, entry := range second.Entries {
		if filepath.Clean(entry.Path) == filepath.Clean(cacheDir) {
			found = true
			if entry.GrowthBytes != 10 {
				t.Fatalf("expected cache growth of 10 bytes, got %d", entry.GrowthBytes)
			}
		}
	}
	if !found {
		t.Fatalf("expected cache directory in growth entries: %#v", second.Entries)
	}

	store, err := loadGrowthStore()
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Snapshots) != 2 {
		t.Fatalf("expected two snapshots, got %#v", store.Snapshots)
	}
	oldID := store.Snapshots[0].ID
	newID := store.Snapshots[1].ID
	children, err := service.CompareSnapshotPath(oldID, newID, filepath.Join(root, "Users"))
	if err != nil {
		t.Fatal(err)
	}
	foundUser := false
	for _, diff := range children.Diffs {
		if filepath.Clean(diff.Path) == filepath.Clean(filepath.Join(root, "Users", "tester")) {
			foundUser = true
			if diff.DeltaBytes != 10 {
				t.Fatalf("expected tester growth of 10 bytes, got %d", diff.DeltaBytes)
			}
		}
	}
	if !foundUser {
		t.Fatalf("expected tester child diff under Users: %#v", children.Diffs)
	}
}

func TestCleanGrowthPathsClearsDirectoryContents(t *testing.T) {
	temp := t.TempDir()
	appData := filepath.Join(temp, "Roaming")
	cacheDir := filepath.Join(appData, "Code", "CachedExtensionVSIXs")
	target := filepath.Join(cacheDir, "extension.vsix")
	t.Setenv("APPDATA", appData)

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := NewService().CleanGrowthPaths(context.Background(), GrowthCleanRequest{Paths: []string{cacheDir}})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeletedCount != 1 || result.DeletedBytes != 5 {
		t.Fatalf("unexpected clean result: %#v", result)
	}
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("expected cache directory to remain, stat err: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected cache file to be deleted, stat err: %v", err)
	}
}
