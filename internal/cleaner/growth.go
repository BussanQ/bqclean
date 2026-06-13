package cleaner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cleanapp/internal/cleaner/rules"
	"cleanapp/internal/cleaner/winapi"
)

const (
	defaultGrowthMaxDepth       = 6
	defaultGrowthMaxResults     = 120
	defaultGrowthMinGrowthBytes = 32 * 1024 * 1024
	snapshotDetailMaxDepth      = 8
	snapshotPathMaxResults      = 80
	// snapshotStoreMinBytes is the minimum size for a deep (depth > 1)
	// directory to be retained in a stored snapshot. Smaller deep dirs are
	// pruned to bound the snapshot file size while keeping meaningful
	// directories available for drill-down. Tune for size/detail tradeoff.
	snapshotStoreMinBytes = 1 * 1024 * 1024
)

type growthStore struct {
	Version   int              `json:"version"`
	Snapshots []growthSnapshot `json:"snapshots"`
}

type growthSnapshot struct {
	ID        string                  `json:"id"`
	Root      string                  `json:"root"`
	ScannedAt string                  `json:"scannedAt"`
	Label     string                  `json:"label"`
	Dirs      []growthDirectoryRecord `json:"dirs"`
}

type growthDirectoryRecord struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	Depth     int    `json:"depth"`
	FileCount int    `json:"fileCount"`
	DirCount  int    `json:"dirCount"`
}

type growthDirectoryStats struct {
	sizeBytes int64
	depth     int
	fileCount int
	dirCount  int
}

func (s *Service) AnalyzeDiskGrowth(ctx context.Context, options DiskGrowthOptions) (DiskGrowthResult, error) {
	ctx = fallbackContext(ctx)
	taskID := newTaskID()
	taskCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.cancels[taskID] = cancel
	s.mu.Unlock()
	defer s.removeCancel(taskID)

	root, err := resolveGrowthRoot(options.Root)
	if err != nil {
		return DiskGrowthResult{}, err
	}
	maxDepth := options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultGrowthMaxDepth
	}
	maxResults := options.MaxResults
	if maxResults <= 0 {
		maxResults = defaultGrowthMaxResults
	}
	minGrowth := options.MinGrowthBytes
	if minGrowth <= 0 {
		minGrowth = defaultGrowthMinGrowthBytes
	}

	store, _ := loadGrowthStore()
	previous := latestGrowthSnapshot(store, root)
	current, failures, cancelled := scanGrowthSnapshot(taskCtx, taskID, root)
	current.Label = time.Now().Format("2006-01-02 15:04")
	result := buildGrowthResult(taskID, current, previous, store, maxDepth, maxResults, minGrowth, failures, cancelled)

	store.Version = 1
	store.Snapshots = append(store.Snapshots, compactGrowthSnapshot(current))
	store.Snapshots = pruneGrowthSnapshots(store.Snapshots)
	if err := saveGrowthStore(store); err != nil {
		result.Failures = append(result.Failures, ScanFailure{Path: growthStorePath(), Reason: err.Error()})
	}

	return result, nil
}

func (s *Service) TakeSnapshot(ctx context.Context, drive string, label string) (DiskSnapshot, error) {
	ctx = fallbackContext(ctx)
	root, err := resolveGrowthRoot(drive)
	if err != nil {
		return DiskSnapshot{}, err
	}
	if label == "" {
		label = time.Now().Format("2006-01-02 15:04")
	}

	current, _, cancelled := scanGrowthSnapshot(ctx, newTaskID(), root)
	if cancelled {
		return DiskSnapshot{}, context.Canceled
	}
	current.Label = label

	store, _ := loadGrowthStore()
	store.Version = 1
	store.Snapshots = append(store.Snapshots, compactGrowthSnapshot(current))
	store.Snapshots = pruneGrowthSnapshots(store.Snapshots)
	if err := saveGrowthStore(store); err != nil {
		return DiskSnapshot{}, err
	}

	return diskSnapshotFromGrowth(current), nil
}

func (s *Service) ListSnapshots() ([]SnapshotInfo, error) {
	store, err := loadGrowthStore()
	if err != nil {
		return nil, err
	}
	infos := make([]SnapshotInfo, 0, len(store.Snapshots))
	for _, snapshot := range store.Snapshots {
		infos = append(infos, SnapshotInfo{
			ID:         snapshot.ID,
			CreatedAt:  snapshot.ScannedAt,
			Label:      growthSnapshotLabel(snapshot),
			TotalBytes: growthSnapshotTotal(snapshot),
			EntryCount: len(snapshot.Dirs),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, infos[i].CreatedAt)
		tj, _ := time.Parse(time.RFC3339, infos[j].CreatedAt)
		return ti.After(tj)
	})
	return infos, nil
}

func (s *Service) CompareSnapshots(oldID string, newID string) (SnapshotCompareResult, error) {
	if oldID == "" || newID == "" {
		return SnapshotCompareResult{}, fmt.Errorf("both snapshot IDs are required")
	}
	store, err := loadGrowthStore()
	if err != nil {
		return SnapshotCompareResult{}, err
	}
	oldSnap, ok := findGrowthSnapshot(store, oldID)
	if !ok {
		return SnapshotCompareResult{}, fmt.Errorf("snapshot %q was not found", oldID)
	}
	newSnap, ok := findGrowthSnapshot(store, newID)
	if !ok {
		return SnapshotCompareResult{}, fmt.Errorf("snapshot %q was not found", newID)
	}
	if !samePath(oldSnap.Root, newSnap.Root) {
		return SnapshotCompareResult{}, fmt.Errorf("snapshots are from different drives: %s vs %s", oldSnap.Root, newSnap.Root)
	}
	return compareGrowthSnapshots(oldSnap, newSnap), nil
}

func (s *Service) CompareSnapshotPath(oldID string, newID string, path string) (SnapshotPathCompareResult, error) {
	if oldID == "" || newID == "" || path == "" {
		return SnapshotPathCompareResult{}, fmt.Errorf("old snapshot ID, new snapshot ID and path are required")
	}
	store, err := loadGrowthStore()
	if err != nil {
		return SnapshotPathCompareResult{}, err
	}
	oldSnap, ok := findGrowthSnapshot(store, oldID)
	if !ok {
		return SnapshotPathCompareResult{}, fmt.Errorf("snapshot %q was not found", oldID)
	}
	newSnap, ok := findGrowthSnapshot(store, newID)
	if !ok {
		return SnapshotPathCompareResult{}, fmt.Errorf("snapshot %q was not found", newID)
	}
	if !samePath(oldSnap.Root, newSnap.Root) {
		return SnapshotPathCompareResult{}, fmt.Errorf("snapshots are from different drives: %s vs %s", oldSnap.Root, newSnap.Root)
	}
	cleanPath := filepath.Clean(path)
	if !pathInsideOrEqual(cleanPath, newSnap.Root) {
		return SnapshotPathCompareResult{}, fmt.Errorf("path %q is outside snapshot root %q", cleanPath, newSnap.Root)
	}
	return SnapshotPathCompareResult{
		OldSnapshotID: oldSnap.ID,
		NewSnapshotID: newSnap.ID,
		Path:          cleanPath,
		Diffs:         compareGrowthSnapshotChildren(oldSnap, newSnap, cleanPath),
	}, nil
}

func (s *Service) DeleteSnapshot(id string) error {
	if id == "" {
		return fmt.Errorf("snapshot ID is required")
	}
	store, err := loadGrowthStore()
	if err != nil {
		return err
	}
	next := make([]growthSnapshot, 0, len(store.Snapshots))
	for _, snapshot := range store.Snapshots {
		if snapshot.ID != id {
			next = append(next, snapshot)
		}
	}
	store.Snapshots = next
	return saveGrowthStore(store)
}

func OpenInExplorer(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	return exec.Command("explorer", abs).Start()
}

func (s *Service) CleanGrowthPaths(ctx context.Context, request GrowthCleanRequest) (CleanResult, error) {
	ctx = fallbackContext(ctx)
	result := CleanResult{Failures: make([]CleanFailure, 0)}
	seen := map[string]bool{}

	for _, rawPath := range request.Paths {
		if err := ctx.Err(); err != nil {
			result.Cancelled = true
			break
		}
		if rawPath == "" || seen[strings.ToLower(rawPath)] {
			continue
		}
		seen[strings.ToLower(rawPath)] = true

		cleanPath, err := filepath.Abs(rawPath)
		if err != nil {
			result.Failures = append(result.Failures, CleanFailure{Path: rawPath, Reason: err.Error()})
			continue
		}
		cleanPath = filepath.Clean(cleanPath)
		if !isAllowedGrowthCleanupPath(cleanPath) {
			result.SkippedCount++
			result.Failures = append(result.Failures, CleanFailure{Path: cleanPath, Reason: "path is not an allowed cache or temporary cleanup target"})
			continue
		}

		deletedBytes, skipped, failure := deleteGrowthTarget(ctx, cleanPath)
		if skipped {
			result.SkippedCount++
		}
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
			continue
		}
		result.DeletedCount++
		result.DeletedBytes += deletedBytes
	}

	return result, nil
}

func scanGrowthSnapshot(ctx context.Context, id string, root string) (growthSnapshot, []ScanFailure, bool) {
	failures := make([]ScanFailure, 0)
	cancelled := false
	root = filepath.Clean(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		return growthSnapshot{
			ID:        id,
			Root:      root,
			ScannedAt: time.Now().Format(time.RFC3339),
			Dirs:      make([]growthDirectoryRecord, 0),
		}, []ScanFailure{{Path: root, Reason: err.Error()}}, false
	}

	dirs := make([]growthDirectoryRecord, 0, len(entries)+16)
	totalSize := int64(0)
	totalFiles := 0
	seen := map[string]bool{}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			cancelled = errors.Is(err, context.Canceled)
			break
		}

		name := entry.Name()
		if isExcludedSnapshotRoot(name) {
			continue
		}
		path := filepath.Join(root, name)

		size, files, childDirs, detailRecords, itemFailures := scanGrowthEntry(ctx, root, path, entry)
		failures = append(failures, itemFailures...)
		if ctx.Err() != nil {
			cancelled = errors.Is(ctx.Err(), context.Canceled)
			break
		}
		totalSize += size
		totalFiles += files
		dirs = append(dirs, growthDirectoryRecord{
			Path:      path,
			SizeBytes: size,
			Depth:     1,
			FileCount: files,
			DirCount:  childDirs,
		})
		dirs = append(dirs, detailRecords...)
		seen[strings.ToLower(filepath.Clean(path))] = true
	}

	cacheRecords, cacheFailures, cacheCancelled := scanGrowthCleanupCandidates(ctx, root, seen)
	failures = append(failures, cacheFailures...)
	if cacheCancelled {
		cancelled = true
	}
	dirs = append(dirs, cacheRecords...)
	dirs = append(dirs, growthDirectoryRecord{
		Path:      root,
		SizeBytes: totalSize,
		Depth:     0,
		FileCount: totalFiles,
		DirCount:  len(dirs),
	})

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Path) < strings.ToLower(dirs[j].Path)
	})

	return growthSnapshot{
		ID:        id,
		Root:      root,
		ScannedAt: time.Now().Format(time.RFC3339),
		Dirs:      dirs,
	}, failures, cancelled
}

func scanGrowthEntry(ctx context.Context, root string, path string, entry fs.DirEntry) (int64, int, int, []growthDirectoryRecord, []ScanFailure) {
	failures := make([]ScanFailure, 0)
	if entry.IsDir() {
		reparse, err := winapi.EntryIsReparsePoint(path, entry)
		if err != nil {
			return 0, 0, 0, nil, []ScanFailure{{Path: path, Reason: err.Error()}}
		}
		if reparse {
			return 0, 0, 0, nil, failures
		}
		size, files, dirs, records, walkFailures := walkGrowthDir(ctx, root, path)
		return size, files, dirs, records, walkFailures
	}

	info, err := entry.Info()
	if err != nil {
		return 0, 0, 0, nil, []ScanFailure{{Path: path, Reason: err.Error()}}
	}
	return info.Size(), 1, 0, nil, failures
}

func walkGrowthDir(ctx context.Context, snapshotRoot string, root string) (int64, int, int, []growthDirectoryRecord, []ScanFailure) {
	failures := make([]ScanFailure, 0)
	stats := map[string]*growthDirectoryStats{}
	totalSize := int64(0)
	fileCount := 0
	dirCount := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			failures = append(failures, ScanFailure{Path: path, Reason: walkErr.Error()})
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != root {
			reparse, err := winapi.EntryIsReparsePoint(path, entry)
			if err != nil {
				failures = append(failures, ScanFailure{Path: path, Reason: err.Error()})
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if reparse {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if entry.IsDir() {
			if path != root {
				dirCount++
				if growthDepth(snapshotRoot, path) <= snapshotDetailMaxDepth {
					ensureGrowthDir(stats, snapshotRoot, path)
				}
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			failures = append(failures, ScanFailure{Path: path, Reason: err.Error()})
			return nil
		}
		totalSize += info.Size()
		fileCount++
		addGrowthFileWithinDepth(stats, snapshotRoot, filepath.Dir(path), info.Size())
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		failures = append(failures, ScanFailure{Path: root, Reason: err.Error()})
	}
	records := make([]growthDirectoryRecord, 0, len(stats))
	for path, stat := range stats {
		if samePath(path, root) {
			continue
		}
		records = append(records, growthDirectoryRecord{
			Path:      path,
			SizeBytes: stat.sizeBytes,
			Depth:     stat.depth,
			FileCount: stat.fileCount,
			DirCount:  stat.dirCount,
		})
	}
	return totalSize, fileCount, dirCount, records, failures
}

func scanGrowthCleanupCandidates(ctx context.Context, root string, seen map[string]bool) ([]growthDirectoryRecord, []ScanFailure, bool) {
	ruleSet := rules.Default(nil)
	records := make([]growthDirectoryRecord, 0, len(ruleSet.Roots))
	failures := make([]ScanFailure, 0)
	cancelled := false

	for _, candidate := range ruleSet.Roots {
		if err := ctx.Err(); err != nil {
			cancelled = errors.Is(err, context.Canceled)
			break
		}
		path := filepath.Clean(candidate.Path)
		if !pathInsideOrEqual(path, root) || seen[strings.ToLower(path)] {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, ScanFailure{Path: path, Reason: err.Error()})
			}
			continue
		}
		size, files, dirs, _, itemFailures := scanGrowthEntry(ctx, root, path, dirEntryInfo{name: filepath.Base(path), info: info})
		failures = append(failures, itemFailures...)
		if size <= 0 && files <= 0 {
			continue
		}
		records = append(records, growthDirectoryRecord{
			Path:      path,
			SizeBytes: size,
			Depth:     growthDepth(root, path),
			FileCount: files,
			DirCount:  dirs,
		})
		seen[strings.ToLower(path)] = true
	}
	return records, failures, cancelled
}

type dirEntryInfo struct {
	name string
	info os.FileInfo
}

func (d dirEntryInfo) Name() string               { return d.name }
func (d dirEntryInfo) IsDir() bool                { return d.info.IsDir() }
func (d dirEntryInfo) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d dirEntryInfo) Info() (os.FileInfo, error) { return d.info, nil }

func isExcludedSnapshotRoot(name string) bool {
	switch strings.ToLower(name) {
	case "windows", "program files", "program files (x86)", "programdata", "$recycle.bin", "system volume information", "pagefile.sys", "hiberfil.sys", "swapfile.sys", "config.msi", "recovery":
		return true
	default:
		return false
	}
}

func diskSnapshotFromGrowth(snapshot growthSnapshot) DiskSnapshot {
	entries := make([]DirEntry, 0, len(snapshot.Dirs))
	for _, dir := range snapshot.Dirs {
		entries = append(entries, DirEntry{
			Path:      dir.Path,
			SizeBytes: dir.SizeBytes,
			FileCount: dir.FileCount,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SizeBytes > entries[j].SizeBytes
	})
	return DiskSnapshot{
		ID:         snapshot.ID,
		CreatedAt:  snapshot.ScannedAt,
		Label:      growthSnapshotLabel(snapshot),
		Drive:      snapshot.Root,
		Entries:    entries,
		TotalBytes: growthSnapshotTotal(snapshot),
	}
}

func compareGrowthSnapshots(oldSnap growthSnapshot, newSnap growthSnapshot) SnapshotCompareResult {
	oldMap := make(map[string]growthDirectoryRecord, len(oldSnap.Dirs))
	newMap := make(map[string]growthDirectoryRecord, len(newSnap.Dirs))
	for _, entry := range oldSnap.Dirs {
		oldMap[strings.ToLower(entry.Path)] = entry
	}
	for _, entry := range newSnap.Dirs {
		newMap[strings.ToLower(entry.Path)] = entry
	}

	diffs := make([]SnapshotDiff, 0, len(newSnap.Dirs))
	for _, newEntry := range newSnap.Dirs {
		if !shouldIncludeSnapshotDiff(newEntry) {
			continue
		}
		oldEntry := oldMap[strings.ToLower(newEntry.Path)]
		delta := newEntry.SizeBytes - oldEntry.SizeBytes
		diffs = append(diffs, SnapshotDiff{
			Path:         newEntry.Path,
			OldSize:      oldEntry.SizeBytes,
			NewSize:      newEntry.SizeBytes,
			DeltaBytes:   delta,
			DeltaPercent: roundedDeltaPercent(oldEntry.SizeBytes, newEntry.SizeBytes, delta),
			Cleanable:    isAllowedGrowthCleanupPath(newEntry.Path),
		})
	}
	for _, oldEntry := range oldSnap.Dirs {
		if _, ok := newMap[strings.ToLower(oldEntry.Path)]; ok {
			continue
		}
		if !shouldIncludeSnapshotDiff(oldEntry) {
			continue
		}
		diffs = append(diffs, SnapshotDiff{
			Path:         oldEntry.Path,
			OldSize:      oldEntry.SizeBytes,
			NewSize:      0,
			DeltaBytes:   -oldEntry.SizeBytes,
			DeltaPercent: -100,
			Cleanable:    false,
		})
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].DeltaBytes > diffs[j].DeltaBytes
	})
	if len(diffs) > defaultGrowthMaxResults {
		diffs = diffs[:defaultGrowthMaxResults]
	}

	return SnapshotCompareResult{
		OldSnapshotID: oldSnap.ID,
		NewSnapshotID: newSnap.ID,
		OldLabel:      growthSnapshotLabel(oldSnap),
		NewLabel:      growthSnapshotLabel(newSnap),
		OldTotalBytes: growthSnapshotTotal(oldSnap),
		NewTotalBytes: growthSnapshotTotal(newSnap),
		Diffs:         diffs,
	}
}

func compareGrowthSnapshotChildren(oldSnap growthSnapshot, newSnap growthSnapshot, parent string) []SnapshotDiff {
	oldMap := make(map[string]growthDirectoryRecord, len(oldSnap.Dirs))
	newMap := make(map[string]growthDirectoryRecord, len(newSnap.Dirs))
	for _, entry := range oldSnap.Dirs {
		oldMap[strings.ToLower(entry.Path)] = entry
	}
	for _, entry := range newSnap.Dirs {
		newMap[strings.ToLower(entry.Path)] = entry
	}

	diffs := make([]SnapshotDiff, 0)
	for _, newEntry := range newSnap.Dirs {
		if !isDirectSnapshotChild(parent, newEntry.Path) {
			continue
		}
		oldEntry := oldMap[strings.ToLower(newEntry.Path)]
		delta := newEntry.SizeBytes - oldEntry.SizeBytes
		diffs = append(diffs, SnapshotDiff{
			Path:         newEntry.Path,
			OldSize:      oldEntry.SizeBytes,
			NewSize:      newEntry.SizeBytes,
			DeltaBytes:   delta,
			DeltaPercent: roundedDeltaPercent(oldEntry.SizeBytes, newEntry.SizeBytes, delta),
			Cleanable:    isAllowedGrowthCleanupPath(newEntry.Path),
		})
	}
	for _, oldEntry := range oldSnap.Dirs {
		if _, ok := newMap[strings.ToLower(oldEntry.Path)]; ok {
			continue
		}
		if !isDirectSnapshotChild(parent, oldEntry.Path) {
			continue
		}
		diffs = append(diffs, SnapshotDiff{
			Path:         oldEntry.Path,
			OldSize:      oldEntry.SizeBytes,
			NewSize:      0,
			DeltaBytes:   -oldEntry.SizeBytes,
			DeltaPercent: -100,
			Cleanable:    false,
		})
	}
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].DeltaBytes > diffs[j].DeltaBytes
	})
	if len(diffs) > snapshotPathMaxResults {
		diffs = diffs[:snapshotPathMaxResults]
	}
	return diffs
}

func isDirectSnapshotChild(parent string, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if samePath(parent, child) || !pathInsideOrEqual(child, parent) {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." {
		return false
	}
	return !strings.Contains(rel, string(os.PathSeparator))
}

func shouldIncludeSnapshotDiff(entry growthDirectoryRecord) bool {
	return entry.Depth <= 1 || isAllowedGrowthCleanupPath(entry.Path)
}

func roundedDeltaPercent(oldSize int64, newSize int64, delta int64) float64 {
	pct := float64(0)
	if oldSize > 0 {
		pct = float64(delta) / float64(oldSize) * 100
	} else if newSize > 0 {
		pct = 100
	}
	return math.Round(pct*100) / 100
}

func growthSnapshotLabel(snapshot growthSnapshot) string {
	if snapshot.Label != "" {
		return snapshot.Label
	}
	parsed, err := time.Parse(time.RFC3339, snapshot.ScannedAt)
	if err != nil {
		return snapshot.ScannedAt
	}
	return parsed.Format("2006-01-02 15:04")
}

func growthSnapshotTotal(snapshot growthSnapshot) int64 {
	for _, dir := range snapshot.Dirs {
		if samePath(dir.Path, snapshot.Root) {
			return dir.SizeBytes
		}
	}
	total := int64(0)
	for _, dir := range snapshot.Dirs {
		if dir.Depth == 1 {
			total += dir.SizeBytes
		}
	}
	return total
}

func findGrowthSnapshot(store growthStore, id string) (growthSnapshot, bool) {
	for _, snapshot := range store.Snapshots {
		if snapshot.ID == id {
			return snapshot, true
		}
	}
	return growthSnapshot{}, false
}

func ensureGrowthDir(stats map[string]*growthDirectoryStats, root string, path string) *growthDirectoryStats {
	path = filepath.Clean(path)
	if stat, ok := stats[path]; ok {
		return stat
	}
	stat := &growthDirectoryStats{depth: growthDepth(root, path)}
	stats[path] = stat
	return stat
}

func addGrowthFile(stats map[string]*growthDirectoryStats, root string, dir string, size int64) {
	dir = filepath.Clean(dir)
	for {
		stat := ensureGrowthDir(stats, root, dir)
		stat.fileCount++
		stat.sizeBytes += size
		if samePath(dir, root) {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func addGrowthFileWithinDepth(stats map[string]*growthDirectoryStats, root string, dir string, size int64) {
	dir = filepath.Clean(dir)
	for {
		if growthDepth(root, dir) <= snapshotDetailMaxDepth {
			stat := ensureGrowthDir(stats, root, dir)
			stat.fileCount++
			stat.sizeBytes += size
		}
		if samePath(dir, root) {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func buildGrowthResult(taskID string, current growthSnapshot, previous *growthSnapshot, store growthStore, maxDepth int, maxResults int, minGrowth int64, failures []ScanFailure, cancelled bool) DiskGrowthResult {
	if failures == nil {
		failures = make([]ScanFailure, 0)
	}
	previousMap := map[string]growthDirectoryRecord{}
	if previous != nil {
		for _, dir := range previous.Dirs {
			previousMap[strings.ToLower(dir.Path)] = dir
		}
	}

	entries := make([]DiskGrowthEntry, 0)
	totalBytes := int64(0)
	totalGrowthBytes := int64(0)
	fileCount := 0

	for _, dir := range current.Dirs {
		if samePath(dir.Path, current.Root) {
			totalBytes = dir.SizeBytes
			fileCount = dir.FileCount
		}
		if dir.Depth > maxDepth {
			continue
		}
		prev := previousMap[strings.ToLower(dir.Path)]
		growthBytes := dir.SizeBytes - prev.SizeBytes
		if previous != nil && growthBytes < minGrowth {
			continue
		}
		if previous == nil && dir.Depth == 0 {
			continue
		}
		if previous != nil {
			totalGrowthBytes += maxInt64(growthBytes, 0)
		}
		entries = append(entries, DiskGrowthEntry{
			Path:              dir.Path,
			Name:              growthName(dir.Path),
			SizeBytes:         dir.SizeBytes,
			PreviousSizeBytes: prev.SizeBytes,
			GrowthBytes:       growthBytes,
			GrowthPercent:     growthPercent(prev.SizeBytes, growthBytes),
			Depth:             dir.Depth,
			FileCount:         dir.FileCount,
			DirCount:          dir.DirCount,
			Trend:             growthTrend(dir.Path, dir.SizeBytes, previous, store),
			Cleanable:         isAllowedGrowthCleanupPath(dir.Path),
			DefaultSelected:   false,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].GrowthBytes != entries[j].GrowthBytes {
			return entries[i].GrowthBytes > entries[j].GrowthBytes
		}
		return entries[i].SizeBytes > entries[j].SizeBytes
	})
	if len(entries) > maxResults {
		entries = entries[:maxResults]
	}

	result := DiskGrowthResult{
		TaskID:           taskID,
		SnapshotID:       current.ID,
		Root:             current.Root,
		ScannedAt:        current.ScannedAt,
		HasBaseline:      previous != nil,
		TotalBytes:       totalBytes,
		TotalGrowthBytes: totalGrowthBytes,
		DirCount:         len(current.Dirs),
		FileCount:        fileCount,
		Entries:          entries,
		Failures:         failures,
		Cancelled:        cancelled,
	}
	if previous != nil {
		result.PreviousSnapshotID = previous.ID
		result.PreviousScannedAt = previous.ScannedAt
	}
	return result
}

func resolveGrowthRoot(root string) (string, error) {
	if root == "" {
		root = os.Getenv("SystemDrive")
		if root == "" {
			root = "C:"
		}
		root += `\`
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	info, err := os.Stat(clean)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", clean)
	}
	return clean, nil
}

func growthStorePath() string {
	if p := os.Getenv("CLEANAPP_GROWTH_STORE"); p != "" {
		return p
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
	}
	if appData == "" {
		appData = os.TempDir()
	}
	return filepath.Join(appData, "BQ Clean", "snapshots", "growth_snapshots.json")
}

func loadGrowthStore() (growthStore, error) {
	path := growthStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return growthStore{Version: 1, Snapshots: make([]growthSnapshot, 0)}, nil
		}
		return growthStore{}, err
	}
	store := growthStore{}
	if err := json.Unmarshal(data, &store); err != nil {
		return growthStore{}, err
	}
	if store.Snapshots == nil {
		store.Snapshots = make([]growthSnapshot, 0)
	}
	for i := range store.Snapshots {
		store.Snapshots[i] = compactGrowthSnapshot(store.Snapshots[i])
	}
	return store, nil
}

func saveGrowthStore(store growthStore) error {
	path := growthStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func latestGrowthSnapshot(store growthStore, root string) *growthSnapshot {
	for i := len(store.Snapshots) - 1; i >= 0; i-- {
		if samePath(store.Snapshots[i].Root, root) {
			snapshot := store.Snapshots[i]
			return &snapshot
		}
	}
	return nil
}

// maxGrowthSnapshotsPerRoot caps how many snapshots are kept per scanned
// root; the newest ones win. Snapshots are appended in chronological order,
// so trailing entries are the most recent.
const maxGrowthSnapshotsPerRoot = 12

func pruneGrowthSnapshots(snapshots []growthSnapshot) []growthSnapshot {
	counts := map[string]int{}
	keep := make([]bool, len(snapshots))
	for i := len(snapshots) - 1; i >= 0; i-- {
		key := strings.ToLower(filepath.Clean(snapshots[i].Root))
		if counts[key] < maxGrowthSnapshotsPerRoot {
			counts[key]++
			keep[i] = true
		}
	}
	pruned := make([]growthSnapshot, 0, len(snapshots))
	for i, snapshot := range snapshots {
		if keep[i] {
			pruned = append(pruned, snapshot)
		}
	}
	return pruned
}

func compactGrowthSnapshot(snapshot growthSnapshot) growthSnapshot {
	if len(snapshot.Dirs) == 0 {
		return snapshot
	}
	dirs := make([]growthDirectoryRecord, 0, len(snapshot.Dirs))
	seen := map[string]bool{}
	for _, dir := range snapshot.Dirs {
		if shouldStoreSnapshotDir(dir, snapshot.Root) {
			key := strings.ToLower(filepath.Clean(dir.Path))
			if seen[key] {
				continue
			}
			seen[key] = true
			dirs = append(dirs, dir)
		}
	}
	snapshot.Dirs = dirs
	return snapshot
}

// shouldStoreSnapshotDir decides whether a directory record is kept in a
// stored snapshot. Root and the first two levels are always kept (so the
// top-level comparison is unchanged); deeper directories are kept when they
// are cleanup targets or large enough to be worth drilling into.
func shouldStoreSnapshotDir(dir growthDirectoryRecord, root string) bool {
	if samePath(dir.Path, root) || dir.Depth <= 1 || isAllowedGrowthCleanupPath(dir.Path) {
		return true
	}
	return dir.Depth <= snapshotDetailMaxDepth && dir.SizeBytes >= snapshotStoreMinBytes
}

func growthTrend(path string, currentSize int64, previous *growthSnapshot, store growthStore) string {
	if previous == nil {
		return "baseline"
	}
	sizes := make([]int64, 0, 4)
	for _, snapshot := range store.Snapshots {
		if !samePath(snapshot.Root, previous.Root) {
			continue
		}
		for _, dir := range snapshot.Dirs {
			if samePath(dir.Path, path) {
				sizes = append(sizes, dir.SizeBytes)
				break
			}
		}
	}
	if len(sizes) == 0 {
		return "new"
	}
	sizes = append(sizes, currentSize)
	if len(sizes) >= 3 && sizes[len(sizes)-3] < sizes[len(sizes)-2] && sizes[len(sizes)-2] < sizes[len(sizes)-1] {
		return "continued_growth"
	}
	if sizes[len(sizes)-1] > sizes[len(sizes)-2] {
		return "increased"
	}
	if sizes[len(sizes)-1] < sizes[len(sizes)-2] {
		return "decreased"
	}
	return "stable"
}

func deleteGrowthTarget(ctx context.Context, path string) (int64, bool, *CleanFailure) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, true, nil
		}
		return 0, false, &CleanFailure{Path: path, Reason: err.Error()}
	}
	reparse, err := winapi.InfoIsReparsePoint(path, info)
	if err != nil {
		return 0, false, &CleanFailure{Path: path, Reason: err.Error()}
	}
	if reparse || info.Mode()&os.ModeSymlink != 0 {
		return 0, true, &CleanFailure{Path: path, Reason: "refusing to delete symlink or reparse point"}
	}

	size := int64(0)
	if info.IsDir() {
		walkErr := filepath.WalkDir(path, func(child string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				return walkErr
			}
			reparse, err := winapi.EntryIsReparsePoint(child, entry)
			if err != nil {
				return err
			}
			if reparse {
				return fmt.Errorf("refusing to delete reparse point %q", child)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return nil
		})
		if walkErr != nil {
			if errors.Is(walkErr, context.Canceled) {
				return 0, false, &CleanFailure{Path: path, Reason: walkErr.Error()}
			}
			return 0, false, &CleanFailure{Path: path, Reason: walkErr.Error()}
		}
	} else {
		size = info.Size()
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return 0, false, &CleanFailure{Path: path, Reason: err.Error()}
		}
		for _, entry := range entries {
			child := filepath.Join(path, entry.Name())
			reparse, err := winapi.EntryIsReparsePoint(child, entry)
			if err != nil {
				return 0, false, &CleanFailure{Path: child, Reason: err.Error()}
			}
			if reparse {
				return 0, true, &CleanFailure{Path: child, Reason: "refusing to delete symlink or reparse point"}
			}
			if err := os.RemoveAll(child); err != nil {
				return 0, false, &CleanFailure{Path: child, Reason: err.Error()}
			}
		}
	} else if err := os.Remove(path); err != nil {
		return 0, false, &CleanFailure{Path: path, Reason: err.Error()}
	}
	return size, false, nil
}

func isAllowedGrowthCleanupPath(path string) bool {
	clean, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	clean = filepath.Clean(clean)
	if isUnsafeGrowthPath(clean) {
		return false
	}

	tempRoots := []string{os.TempDir(), os.Getenv("TEMP"), os.Getenv("TMP"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp")}
	for _, root := range tempRoots {
		if root != "" && pathInsideOrEqual(clean, root) && !samePath(clean, root) {
			return true
		}
	}

	for _, root := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")} {
		if root == "" || !pathInsideOrEqual(clean, root) || samePath(clean, root) {
			continue
		}
		if hasCacheLikeSegment(clean) {
			return true
		}
	}
	return false
}

func isUnsafeGrowthPath(path string) bool {
	volume := filepath.VolumeName(path)
	if volume != "" && samePath(path, volume+`\`) {
		return true
	}
	for _, root := range []string{
		os.Getenv("SystemRoot"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("ProgramData"),
		filepath.Join(filepath.VolumeName(path)+`\`, "$Recycle.Bin"),
		filepath.Join(filepath.VolumeName(path)+`\`, "System Volume Information"),
		filepath.Join(filepath.VolumeName(path)+`\`, "Recovery"),
	} {
		if root != "" && pathInsideOrEqual(path, root) {
			return true
		}
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" && samePath(path, userProfile) {
		return true
	}
	return false
}

func hasCacheLikeSegment(path string) bool {
	for _, part := range strings.Split(strings.ToLower(filepath.Clean(path)), string(os.PathSeparator)) {
		switch part {
		case "cache", "caches", "code cache", "gpucache", "cachedata", "cachestorage", "temp", "tmp", "logs", "log", "crashes", "cachedextensionvsixs":
			return true
		}
		if strings.Contains(part, "cache") || strings.Contains(part, "temp") {
			return true
		}
	}
	return false
}

func growthDepth(root string, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	depth := 1
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part != "" {
			depth++
		}
	}
	return depth - 1
}

func growthName(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(os.PathSeparator) || base == "" {
		return path
	}
	return base
}

func growthPercent(previous int64, growth int64) float64 {
	if previous <= 0 {
		if growth > 0 {
			return 100
		}
		return 0
	}
	return float64(growth) / float64(previous) * 100
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func samePath(a string, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func pathInsideOrEqual(path string, root string) bool {
	cleanPath := strings.ToLower(filepath.Clean(path))
	cleanRoot := strings.ToLower(filepath.Clean(root))
	return cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator))
}
