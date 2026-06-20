package cleaner

import "cleanapp/internal/cleaner/model"

type ScanOptions = model.ScanOptions
type CleanRequest = model.CleanRequest
type CleanCategory = model.CleanCategory
type RiskLevel = model.RiskLevel
type ScanItem = model.ScanItem
type CategorySummary = model.CategorySummary
type ScanFailure = model.ScanFailure
type ScanResult = model.ScanResult
type CleanFailure = model.CleanFailure
type CleanResult = model.CleanResult
type DiskGrowthOptions = model.DiskGrowthOptions
type DiskGrowthEntry = model.DiskGrowthEntry
type DiskGrowthResult = model.DiskGrowthResult
type GrowthCleanRequest = model.GrowthCleanRequest
type DirEntry = model.DirEntry
type DiskSnapshot = model.DiskSnapshot
type SnapshotDiff = model.SnapshotDiff
type SnapshotCompareResult = model.SnapshotCompareResult
type SnapshotPathCompareResult = model.SnapshotPathCompareResult
type SnapshotInfo = model.SnapshotInfo

const (
	CategoryUserTemp      = model.CategoryUserTemp
	CategorySystemTemp    = model.CategorySystemTemp
	CategoryChromeCache   = model.CategoryChromeCache
	CategoryEdgeCache     = model.CategoryEdgeCache
	CategoryVSCodeCache   = model.CategoryVSCodeCache
	CategoryWindowsCache  = model.CategoryWindowsCache
	CategoryDevCache      = model.CategoryDevCache
	CategoryWindowsUpdate = model.CategoryWindowsUpdate
	CategoryWindowsLogs   = model.CategoryWindowsLogs
	CategoryAppCache      = model.CategoryAppCache
	CategoryRecycleBin    = model.CategoryRecycleBin
	RiskLow               = model.RiskLow
	RiskMedium            = model.RiskMedium
)
