package cleaner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	cleandelete "cleanapp/internal/cleaner/delete"
	"cleanapp/internal/cleaner/rules"
	"cleanapp/internal/cleaner/scanner"
	"cleanapp/internal/cleaner/winapi"
)

// maxScanSessions bounds how many completed scan sessions are kept for
// later Clean calls; older sessions are evicted oldest-first.
const maxScanSessions = 3

type Service struct {
	mu           sync.Mutex
	sessions     map[string]scanSession
	sessionOrder []string
	cancels      map[string]context.CancelFunc
}

type scanSession struct {
	roots []rules.Root
	items map[string]ScanItem
}

func NewService() *Service {
	return &Service{
		sessions: make(map[string]scanSession),
		cancels:  make(map[string]context.CancelFunc),
	}
}

func (s *Service) Scan(ctx context.Context, options ScanOptions) (ScanResult, error) {
	ctx = fallbackContext(ctx)
	taskID := newTaskID()
	taskCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.cancels[taskID] = cancel
	s.mu.Unlock()
	defer s.removeCancel(taskID)

	ruleSet := rules.Default(options.Categories)
	scan := scanner.New(winapi.EntryIsReparsePoint)
	items, failures, cancelled := scan.Scan(taskCtx, ruleSet.Roots)

	if rules.IncludesCategory(options.Categories, CategoryRecycleBin) {
		recycleItems, recycleFailures := scanRecycleBins(recycleBinRoots(), winapi.QueryRecycleBin)
		failures = append(failures, recycleFailures...)
		items = append(items, recycleItems...)
	}

	for i := range items {
		items[i].ID = makeItemID(taskID, i, items[i])
	}

	result := buildScanResult(taskID, items, failures, cancelled)
	sessionItems := make(map[string]ScanItem, len(items))
	for _, item := range items {
		sessionItems[item.ID] = item
	}

	s.mu.Lock()
	s.sessions[taskID] = scanSession{roots: ruleSet.Roots, items: sessionItems}
	s.sessionOrder = append(s.sessionOrder, taskID)
	for len(s.sessionOrder) > maxScanSessions {
		oldID := s.sessionOrder[0]
		s.sessionOrder = s.sessionOrder[1:]
		delete(s.sessions, oldID)
	}
	s.mu.Unlock()

	return result, nil
}

func (s *Service) Clean(ctx context.Context, request CleanRequest) (CleanResult, error) {
	ctx = fallbackContext(ctx)
	if request.TaskID == "" {
		return CleanResult{}, errors.New("taskId is required")
	}

	s.mu.Lock()
	session, ok := s.sessions[request.TaskID]
	s.mu.Unlock()
	if !ok {
		return CleanResult{}, fmt.Errorf("scan task %q was not found", request.TaskID)
	}

	taskCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancels[request.TaskID] = cancel
	s.mu.Unlock()
	defer s.removeCancel(request.TaskID)

	deleter := cleandelete.New(winapi.InfoIsReparsePoint)
	result := CleanResult{Failures: make([]CleanFailure, 0)}
	seen := map[string]bool{}

	for _, itemID := range request.ItemIDs {
		if err := taskCtx.Err(); err != nil {
			result.Cancelled = true
			break
		}
		if seen[itemID] {
			continue
		}
		seen[itemID] = true

		item, ok := session.items[itemID]
		if !ok {
			result.SkippedCount++
			result.Failures = append(result.Failures, CleanFailure{Path: itemID, Reason: "item is not part of this scan task"})
			continue
		}

		if item.IsVirtual && item.Category == CategoryRecycleBin {
			if err := winapi.EmptyRecycleBin(filepath.VolumeName(item.Path) + `\`); err != nil {
				result.Failures = append(result.Failures, CleanFailure{Path: item.Path, Reason: err.Error()})
				continue
			}
			result.DeletedCount++
			result.DeletedBytes += item.SizeBytes
			continue
		}

		deleted, skipped, failure := deleter.Delete(taskCtx, item, session.roots)
		if deleted {
			result.DeletedCount++
			result.DeletedBytes += item.SizeBytes
		}
		if skipped {
			result.SkippedCount++
		}
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
		}
	}

	return result, nil
}

func (s *Service) CancelTask(taskID string) error {
	s.mu.Lock()
	cancel, ok := s.cancels[taskID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("task %q is not running", taskID)
	}
	cancel()
	return nil
}

func (s *Service) removeCancel(taskID string) {
	s.mu.Lock()
	delete(s.cancels, taskID)
	s.mu.Unlock()
}

func fallbackContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// recycleBinRoots returns the drive roots whose recycle bins should be
// scanned, falling back to the system drive when enumeration fails.
func recycleBinRoots() []string {
	roots, err := winapi.FixedDriveRoots()
	if err == nil && len(roots) > 0 {
		return roots
	}
	systemDrive := os.Getenv("SystemDrive")
	if systemDrive == "" {
		systemDrive = "C:"
	}
	return []string{systemDrive + `\`}
}

func scanRecycleBins(roots []string, query func(root string) (int64, int64, error)) ([]ScanItem, []ScanFailure) {
	items := make([]ScanItem, 0, len(roots))
	failures := make([]ScanFailure, 0)
	for _, root := range roots {
		size, count, err := query(root)
		if err != nil {
			failures = append(failures, ScanFailure{Path: "Recycle Bin (" + root + ")", Reason: err.Error()})
			continue
		}
		if size <= 0 && count <= 0 {
			continue
		}
		items = append(items, ScanItem{
			Path:            root,
			SizeBytes:       size,
			ModifiedAt:      time.Now().Format(time.RFC3339),
			Category:        CategoryRecycleBin,
			Risk:            RiskMedium,
			DefaultSelected: false,
			IsVirtual:       true,
		})
	}
	return items, failures
}

func buildScanResult(taskID string, items []ScanItem, failures []ScanFailure, cancelled bool) ScanResult {
	if items == nil {
		items = make([]ScanItem, 0)
	}
	if failures == nil {
		failures = make([]ScanFailure, 0)
	}

	summaryMap := make(map[CleanCategory]CategorySummary)
	totalBytes := int64(0)

	for _, item := range items {
		totalBytes += item.SizeBytes
		summary := summaryMap[item.Category]
		summary.Category = item.Category
		summary.ItemCount++
		summary.SizeBytes += item.SizeBytes
		summaryMap[item.Category] = summary
	}

	summaries := make([]CategorySummary, 0, len(summaryMap))
	for _, summary := range summaryMap {
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Category < summaries[j].Category
	})

	return ScanResult{
		TaskID:     taskID,
		Items:      items,
		Summaries:  summaries,
		TotalCount: len(items),
		TotalBytes: totalBytes,
		Failures:   failures,
		Cancelled:  cancelled,
	}
}

func makeItemID(taskID string, index int, item ScanItem) string {
	return fmt.Sprintf("%s-%04d-%x", taskID, index, len(item.Path))
}

func newTaskID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
