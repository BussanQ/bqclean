# Windows Cache Cleaner - Product Technical Plan

## 1. Product Goal

Build a Windows desktop cleaner application based on Go and Wails. The first version focuses on safe cleanup of common cache and temporary files on the C drive. It should scan known safe locations, show users what can be cleaned, and delete only after explicit user confirmation.

The application should feel similar to a lightweight "file cleaner master" tool, but the first version must prioritize safety and transparency over aggressive cleanup.

## 2. First-Version Scope

### In Scope

- Windows desktop GUI application.
- C drive cleanup only.
- Rule-based scanning of known cache and temporary directories.
- Category-level and item-level cleanup result display.
- User-controlled selection before cleanup.
- Safe deletion with path validation before every delete operation.
- Clear reporting of skipped files, failed deletions, and reclaimed space.

### Out of Scope

- Full-disk recursive smart cleanup.
- Registry cleanup.
- Driver cleanup.
- Duplicate file detection.
- Background resident service.
- Scheduled automatic cleanup.
- Browser history, cookie, password, session, or local storage cleanup.
- Automatic privilege elevation.

## 3. Technical Stack

- Language: Go.
- Desktop shell: Wails.
- Runtime UI container: WebView2.
- Frontend: Wails-supported web frontend, preferably TypeScript.
- Target platform: Windows, amd64.

Go owns the scanning, classification, safety validation, and deletion logic. The frontend owns presentation, user selection, confirmation dialogs, progress display, and result views.

Wails CLI may be installed during implementation. The current development environment already has Go, Node, and npm available.

## 4. User Flow

1. User opens the application.
2. Home page shows a scan button and the currently supported cleanup categories.
3. User starts a scan.
4. App scans only configured safe locations and reports progress.
5. Result page groups files by cleanup category.
6. Categories and individual items can be selected or unselected.
7. User clicks clean.
8. App shows a confirmation dialog with selected file count and total size.
9. App validates selected paths again and deletes eligible files.
10. Completion page shows reclaimed size, deleted count, skipped items, and failed items.

## 5. Default Scan Rules

The first version must use explicit allow-list rules. It must not traverse the whole C drive looking for files by extension or age.

### Supported Categories

| Category | Example Paths | Default Selected |
| --- | --- | --- |
| User temporary files | `%LOCALAPPDATA%\Temp` | Yes |
| System temporary files | `C:\Windows\Temp` | Yes |
| Chrome cache | `%LOCALAPPDATA%\Google\Chrome\User Data\*\Cache`, `Code Cache`, `GPUCache`, `Service Worker\CacheStorage` | Yes |
| Edge cache | `%LOCALAPPDATA%\Microsoft\Edge\User Data\*\Cache`, `Code Cache`, `GPUCache`, `Service Worker\CacheStorage` | Yes |
| Recycle Bin | Windows Recycle Bin API | No or confirm separately |

Browser profile scanning may include multiple profile directories such as `Default`, `Profile 1`, and `Profile 2`, but it must only target cache subdirectories.

### Explicit Exclusions

The scanner must exclude:

- `C:\Windows` except `C:\Windows\Temp`.
- `C:\Program Files`.
- `C:\Program Files (x86)`.
- User `Documents`, `Desktop`, and `Downloads`.
- Browser `Cookies`, `History`, `Login Data`, `Local Storage`, `Session Storage`, and other user data files.
- Junctions, symlinks, and reparse points.
- Directories that cannot be accessed without elevated privileges.

Permission failures must be recorded as scan failures and must not stop the whole scan.

## 6. Safety Model

Safety is the core product requirement.

- Scanning is read-only and should only collect metadata.
- Cleanup only deletes user-selected scan items.
- Every delete operation must re-check that the target path is still inside an allowed cleanup root.
- The cleaner must not follow symlinks, junctions, or other reparse points.
- Files that changed, disappeared, or became inaccessible after scan should be skipped or reported as failed.
- Permission errors must not trigger automatic privilege elevation.
- The app must continue after individual file failures.
- Deletion should prefer file-by-file cleanup so partial success can be reported.

For directories, delete contents first and remove the directory only if it is empty and still inside an allowed root.

## 7. Go Service API

Expose the cleanup service to Wails with a small, stable interface.

```go
type CleanerService struct {}

func (s *CleanerService) Scan(ctx context.Context, options ScanOptions) (ScanResult, error)
func (s *CleanerService) Clean(ctx context.Context, request CleanRequest) (CleanResult, error)
func (s *CleanerService) CancelTask(taskID string) error
```

### Data Types

```go
type ScanOptions struct {
    Categories []CleanCategory `json:"categories"`
}

type CleanRequest struct {
    TaskID string   `json:"taskId"`
    ItemIDs []string `json:"itemIds"`
}

type CleanCategory string

const (
    CategoryUserTemp    CleanCategory = "user_temp"
    CategorySystemTemp  CleanCategory = "system_temp"
    CategoryChromeCache CleanCategory = "chrome_cache"
    CategoryEdgeCache   CleanCategory = "edge_cache"
    CategoryRecycleBin  CleanCategory = "recycle_bin"
)

type RiskLevel string

const (
    RiskLow    RiskLevel = "low"
    RiskMedium RiskLevel = "medium"
)

type ScanItem struct {
    ID              string        `json:"id"`
    Path            string        `json:"path"`
    SizeBytes       int64         `json:"sizeBytes"`
    ModifiedAt      time.Time     `json:"modifiedAt"`
    Category        CleanCategory `json:"category"`
    Risk            RiskLevel     `json:"risk"`
    DefaultSelected bool          `json:"defaultSelected"`
    IsDirectory     bool          `json:"isDirectory"`
}

type CategorySummary struct {
    Category  CleanCategory `json:"category"`
    ItemCount int           `json:"itemCount"`
    SizeBytes int64         `json:"sizeBytes"`
}

type ScanFailure struct {
    Path    string `json:"path"`
    Reason  string `json:"reason"`
}

type ScanResult struct {
    TaskID     string            `json:"taskId"`
    Items      []ScanItem        `json:"items"`
    Summaries  []CategorySummary `json:"summaries"`
    TotalCount int               `json:"totalCount"`
    TotalBytes int64             `json:"totalBytes"`
    Failures   []ScanFailure     `json:"failures"`
}

type CleanFailure struct {
    Path   string `json:"path"`
    Reason string `json:"reason"`
}

type CleanResult struct {
    DeletedCount int            `json:"deletedCount"`
    DeletedBytes int64          `json:"deletedBytes"`
    SkippedCount int            `json:"skippedCount"`
    Failures     []CleanFailure `json:"failures"`
}
```

## 8. Internal Module Design

Recommended Go package layout:

```text
cmd/cleanapp/
internal/cleaner/
internal/cleaner/rules/
internal/cleaner/scanner/
internal/cleaner/delete/
internal/cleaner/winapi/
frontend/
```

Responsibilities:

- `rules`: build allowed cleanup roots and exclusion rules.
- `scanner`: walk allowed roots, collect metadata, avoid reparse points.
- `delete`: validate and delete selected items.
- `winapi`: Windows-specific Recycle Bin integration.
- `cleaner`: service orchestration, task state, cancellation, result aggregation.
- `frontend`: Wails UI.

Task state may be kept in memory for the first version. Persisting historical scan results is not required.

## 9. UI Requirements

The GUI should be practical and quiet, not marketing-oriented.

Required views:

- Home view with scan action and supported cleanup categories.
- Scanning view with progress by category.
- Results view with category summaries, total reclaimable size, and selectable items.
- Confirmation dialog before deletion.
- Cleanup progress view.
- Completion view with reclaimed space and failure details.

Required interactions:

- Select or unselect a category.
- Expand a category to inspect items.
- Select or unselect individual items.
- Cancel a running scan or cleanup when technically possible.
- Retry scan after completion or failure.

## 10. Error Handling

- Access denied: record and continue.
- File in use: report deletion failure and continue.
- Path disappeared after scan: count as skipped.
- Path moved outside allowed root: skip and report as safety validation failure.
- Recycle Bin API failure: report category-level failure.
- Scan cancellation: return partial result only if it is clearly marked as cancelled.

Errors shown in the UI should be understandable to normal users while preserving detailed technical reasons for debugging.

## 11. Testing Plan

### Unit Tests

- Include safe cleanup roots correctly.
- Exclude protected system and user data paths.
- Normalize paths before safety validation.
- Reject paths outside allowed roots.
- Reject symlink, junction, and reparse-point traversal.
- Preserve scan progress when one directory fails.

### Integration Tests

- Create temporary directory trees that simulate cache directories.
- Verify scan counts, size totals, categories, and default selections.
- Delete selected test files and verify results.
- Simulate missing files between scan and cleanup.
- Simulate file-in-use or permission failures where practical.

### UI Acceptance Tests

- Home page starts a scan.
- Results page groups files by category.
- Category selection updates selected size.
- Cleanup cannot start without confirmation.
- Completion view shows deleted size, deleted count, skipped count, and failures.

## 12. Implementation Milestones

1. Initialize Wails project and Go module.
2. Implement cleanup rule definitions and path safety validation.
3. Implement scanner for user temp and system temp.
4. Add browser cache rules for Chrome and Edge.
5. Add cleanup deletion engine.
6. Expose Go service methods to Wails.
7. Build the main GUI flow.
8. Add Recycle Bin support through Windows API.
9. Add unit and integration tests.
10. Polish UI states, errors, and packaging.

## 13. Acceptance Criteria

The first version is acceptable when:

- The app runs as a Windows desktop GUI.
- It scans only allowed cleanup locations.
- It never deletes without explicit user confirmation.
- It validates paths again before deletion.
- It does not follow symlinks or junctions.
- It reports skipped and failed files.
- It can clean test cache files and show reclaimed size correctly.
- Unit and integration tests cover the safety model.

