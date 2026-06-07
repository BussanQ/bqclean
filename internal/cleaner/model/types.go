package model

type ScanOptions struct {
	Categories []CleanCategory `json:"categories"`
}

type CleanRequest struct {
	TaskID  string   `json:"taskId"`
	ItemIDs []string `json:"itemIds"`
}

type CleanCategory string

const (
	CategoryUserTemp    CleanCategory = "user_temp"
	CategorySystemTemp  CleanCategory = "system_temp"
	CategoryChromeCache CleanCategory = "chrome_cache"
	CategoryEdgeCache   CleanCategory = "edge_cache"
	CategoryVSCodeCache CleanCategory = "vscode_cache"
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
	ModifiedAt      string        `json:"modifiedAt"`
	Category        CleanCategory `json:"category"`
	Risk            RiskLevel     `json:"risk"`
	DefaultSelected bool          `json:"defaultSelected"`
	IsDirectory     bool          `json:"isDirectory"`
	IsVirtual       bool          `json:"isVirtual"`
}

type CategorySummary struct {
	Category  CleanCategory `json:"category"`
	ItemCount int           `json:"itemCount"`
	SizeBytes int64         `json:"sizeBytes"`
}

type ScanFailure struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type ScanResult struct {
	TaskID     string            `json:"taskId"`
	Items      []ScanItem        `json:"items"`
	Summaries  []CategorySummary `json:"summaries"`
	TotalCount int               `json:"totalCount"`
	TotalBytes int64             `json:"totalBytes"`
	Failures   []ScanFailure     `json:"failures"`
	Cancelled  bool              `json:"cancelled"`
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
	Cancelled    bool           `json:"cancelled"`
}

type DiskGrowthOptions struct {
	Root           string `json:"root"`
	MaxDepth       int    `json:"maxDepth"`
	MaxResults     int    `json:"maxResults"`
	MinGrowthBytes int64  `json:"minGrowthBytes"`
}

type DiskGrowthEntry struct {
	Path              string  `json:"path"`
	Name              string  `json:"name"`
	SizeBytes         int64   `json:"sizeBytes"`
	PreviousSizeBytes int64   `json:"previousSizeBytes"`
	GrowthBytes       int64   `json:"growthBytes"`
	GrowthPercent     float64 `json:"growthPercent"`
	Depth             int     `json:"depth"`
	FileCount         int     `json:"fileCount"`
	DirCount          int     `json:"dirCount"`
	Trend             string  `json:"trend"`
	Cleanable         bool    `json:"cleanable"`
	DefaultSelected   bool    `json:"defaultSelected"`
}

type DiskGrowthResult struct {
	TaskID             string            `json:"taskId"`
	SnapshotID         string            `json:"snapshotId"`
	Root               string            `json:"root"`
	ScannedAt          string            `json:"scannedAt"`
	PreviousSnapshotID string            `json:"previousSnapshotId"`
	PreviousScannedAt  string            `json:"previousScannedAt"`
	HasBaseline        bool              `json:"hasBaseline"`
	TotalBytes         int64             `json:"totalBytes"`
	TotalGrowthBytes   int64             `json:"totalGrowthBytes"`
	DirCount           int               `json:"dirCount"`
	FileCount          int               `json:"fileCount"`
	Entries            []DiskGrowthEntry `json:"entries"`
	Failures           []ScanFailure     `json:"failures"`
	Cancelled          bool              `json:"cancelled"`
}

type GrowthCleanRequest struct {
	SnapshotID string   `json:"snapshotId"`
	Paths      []string `json:"paths"`
}

type DirEntry struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	FileCount int    `json:"fileCount"`
}

type DiskSnapshot struct {
	ID         string     `json:"id"`
	CreatedAt  string     `json:"createdAt"`
	Label      string     `json:"label"`
	Drive      string     `json:"drive"`
	Entries    []DirEntry `json:"entries"`
	TotalBytes int64      `json:"totalBytes"`
}

type SnapshotDiff struct {
	Path         string  `json:"path"`
	OldSize      int64   `json:"oldSize"`
	NewSize      int64   `json:"newSize"`
	DeltaBytes   int64   `json:"deltaBytes"`
	DeltaPercent float64 `json:"deltaPercent"`
	Cleanable    bool    `json:"cleanable"`
}

type SnapshotCompareResult struct {
	OldSnapshotID string         `json:"oldSnapshotId"`
	NewSnapshotID string         `json:"newSnapshotId"`
	OldLabel      string         `json:"oldLabel"`
	NewLabel      string         `json:"newLabel"`
	OldTotalBytes int64          `json:"oldTotalBytes"`
	NewTotalBytes int64          `json:"newTotalBytes"`
	Diffs         []SnapshotDiff `json:"diffs"`
}

type SnapshotPathCompareResult struct {
	OldSnapshotID string         `json:"oldSnapshotId"`
	NewSnapshotID string         `json:"newSnapshotId"`
	Path          string         `json:"path"`
	Diffs         []SnapshotDiff `json:"diffs"`
}

type SnapshotInfo struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	Label      string `json:"label"`
	TotalBytes int64  `json:"totalBytes"`
	EntryCount int    `json:"entryCount"`
}
