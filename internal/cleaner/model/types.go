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
