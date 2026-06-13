package cleaner

import (
	"context"
	"fmt"
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

func TestPruneGrowthSnapshotsKeepsNewestPerRoot(t *testing.T) {
	snapshots := make([]growthSnapshot, 0, maxGrowthSnapshotsPerRoot+4)
	for i := 0; i < maxGrowthSnapshotsPerRoot+2; i++ {
		snapshots = append(snapshots, growthSnapshot{ID: fmt.Sprintf("c-%d", i), Root: `C:\`})
	}
	snapshots = append(snapshots, growthSnapshot{ID: "d-0", Root: `D:\`}, growthSnapshot{ID: "d-1", Root: `d:\`})

	pruned := pruneGrowthSnapshots(snapshots)

	cCount, dCount := 0, 0
	for _, snapshot := range pruned {
		if snapshot.Root == `C:\` {
			cCount++
		} else {
			dCount++
		}
	}
	if cCount != maxGrowthSnapshotsPerRoot {
		t.Fatalf("expected %d C: snapshots, got %d", maxGrowthSnapshotsPerRoot, cCount)
	}
	if dCount != 2 {
		t.Fatalf("expected both D: snapshots to survive, got %d", dCount)
	}
	for _, snapshot := range pruned {
		if snapshot.ID == "c-0" || snapshot.ID == "c-1" {
			t.Fatalf("oldest C: snapshots should be pruned, found %s", snapshot.ID)
		}
	}
	if pruned[len(pruned)-1].ID != "d-1" {
		t.Fatalf("pruning must preserve order, last = %s", pruned[len(pruned)-1].ID)
	}
}

func TestCompactGrowthSnapshotKeepsDeepLargeDirs(t *testing.T) {
	// Use a synthetic path outside Temp/LocalAppData so isAllowedGrowthCleanupPath
	// does not treat deep dirs as cleanable. compactGrowthSnapshot does not touch disk.
	base := filepath.Join("Z:", "snaptest")
	u := filepath.Join(base, "u")
	big := filepath.Join(u, "big")
	small := filepath.Join(u, "small")
	deep := filepath.Join(big, "a", "b", "c", "d", "e", "f", "deep")

	snapshot := growthSnapshot{
		Root: base,
		Dirs: []growthDirectoryRecord{
			{Path: base, Depth: 0, SizeBytes: 100 * 1024 * 1024},
			{Path: u, Depth: 1, SizeBytes: 50 * 1024},
			{Path: big, Depth: 2, SizeBytes: 2 * 1024 * 1024},
			{Path: small, Depth: 2, SizeBytes: 500 * 1024},
			{Path: deep, Depth: 9, SizeBytes: 5 * 1024 * 1024},
		},
	}

	kept := map[string]bool{}
	for _, dir := range compactGrowthSnapshot(snapshot).Dirs {
		kept[dir.Path] = true
	}

	for _, want := range []string{base, u, big} {
		if !kept[want] {
			t.Fatalf("expected %q to be retained", want)
		}
	}
	for _, drop := range []string{small, deep} {
		if kept[drop] {
			t.Fatalf("expected %q to be pruned (deep+small or too deep)", drop)
		}
	}
}

func TestCompareGrowthSnapshotChildrenMultiLevel(t *testing.T) {
	base := t.TempDir()
	d1 := filepath.Join(base, "a")
	d2 := filepath.Join(d1, "b")
	d3 := filepath.Join(d2, "c")

	mk := func(s1, s2, s3 int64) growthSnapshot {
		return growthSnapshot{
			Root: base,
			Dirs: []growthDirectoryRecord{
				{Path: base, Depth: 0, SizeBytes: s1 + 1},
				{Path: d1, Depth: 1, SizeBytes: s1},
				{Path: d2, Depth: 2, SizeBytes: s2},
				{Path: d3, Depth: 3, SizeBytes: s3},
			},
		}
	}
	oldSnap := mk(10*1024*1024, 6*1024*1024, 3*1024*1024)
	newSnap := mk(15*1024*1024, 9*1024*1024, 5*1024*1024)

	steps := []struct {
		parent    string
		wantChild string
		wantDelta int64
	}{
		{base, d1, 5 * 1024 * 1024},
		{d1, d2, 3 * 1024 * 1024},
		{d2, d3, 2 * 1024 * 1024},
	}
	for _, step := range steps {
		diffs := compareGrowthSnapshotChildren(oldSnap, newSnap, step.parent)
		if len(diffs) != 1 {
			t.Fatalf("parent %q: expected 1 direct child, got %d", step.parent, len(diffs))
		}
		if diffs[0].Path != step.wantChild {
			t.Fatalf("parent %q: expected child %q, got %q", step.parent, step.wantChild, diffs[0].Path)
		}
		if diffs[0].DeltaBytes != step.wantDelta {
			t.Fatalf("parent %q: expected delta %d, got %d", step.parent, step.wantDelta, diffs[0].DeltaBytes)
		}
	}
}
