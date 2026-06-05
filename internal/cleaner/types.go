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

const (
	CategoryUserTemp    = model.CategoryUserTemp
	CategorySystemTemp  = model.CategorySystemTemp
	CategoryChromeCache = model.CategoryChromeCache
	CategoryEdgeCache   = model.CategoryEdgeCache
	CategoryRecycleBin  = model.CategoryRecycleBin
	RiskLow             = model.RiskLow
	RiskMedium          = model.RiskMedium
)
