package rules

import (
	"os"
	"path/filepath"
	"strings"

	"cleanapp/internal/cleaner/model"
)

type Root struct {
	Path            string
	Category        model.CleanCategory
	DefaultSelected bool
	Risk            model.RiskLevel
	// Filter, when non-nil, restricts which files under Path are surfaced as
	// cleanup items. It receives the file's base name and size in bytes and
	// returns true to keep it. Roots without a filter include every regular
	// file in the subtree.
	Filter func(name string, sizeBytes int64) bool
	// SkipDir, when non-nil, prunes subdirectories during the walk. It receives
	// a directory's base name and returns true to skip descending into it,
	// avoiding scan-failure noise from system folders that are always locked.
	SkipDir func(name string) bool
}

type RuleSet struct {
	Roots []Root
}

func Default(categories []model.CleanCategory) RuleSet {
	selected := categorySet(categories)
	roots := make([]Root, 0, 16)

	if selected[model.CategoryUserTemp] {
		if p := os.Getenv("LOCALAPPDATA"); p != "" {
			roots = append(roots, Root{
				Path:            filepath.Join(p, "Temp"),
				Category:        model.CategoryUserTemp,
				DefaultSelected: true,
				Risk:            model.RiskLow,
			})
		}
	}

	if selected[model.CategorySystemTemp] {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		roots = append(roots, Root{
			Path:            filepath.Join(systemRoot, "Temp"),
			Category:        model.CategorySystemTemp,
			DefaultSelected: true,
			Risk:            model.RiskMedium,
		})
	}

	if selected[model.CategoryChromeCache] {
		roots = append(roots, browserRoots(filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data"), model.CategoryChromeCache)...)
	}

	if selected[model.CategoryEdgeCache] {
		roots = append(roots, browserRoots(filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data"), model.CategoryEdgeCache)...)
	}

	if selected[model.CategoryEdgeIndexedDB] {
		roots = append(roots, edgeIndexedDBRoots(filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data"))...)
	}

	if selected[model.CategoryVSCodeCache] {
		if p := os.Getenv("APPDATA"); p != "" {
			roots = append(roots, Root{
				Path:            filepath.Join(p, "Code", "CachedExtensionVSIXs"),
				Category:        model.CategoryVSCodeCache,
				DefaultSelected: true,
				Risk:            model.RiskLow,
			})
		}
	}

	if selected[model.CategoryWindowsCache] {
		for _, parts := range [][]string{
			{"Microsoft", "Windows", "Explorer"},
			{"Microsoft", "Windows", "INetCache"},
			{"CrashDumps"},
			{"Microsoft", "Windows", "WER"},
		} {
			if root, ok := envRoot("LOCALAPPDATA", model.CategoryWindowsCache, model.RiskLow, true, parts...); ok {
				roots = append(roots, root)
			}
		}
	}

	if selected[model.CategoryDevCache] {
		for _, parts := range [][]string{
			{"npm-cache"},
			{"pip", "Cache"},
			{"go-build"},
			{"Yarn", "Cache"},
			{"NuGet", "v3-cache"},
			{"NuGet", "Cache"},
		} {
			if root, ok := envRoot("LOCALAPPDATA", model.CategoryDevCache, model.RiskLow, false, parts...); ok {
				roots = append(roots, root)
			}
		}
	}

	if selected[model.CategoryWindowsUpdate] {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		for _, parts := range [][]string{
			{"SoftwareDistribution", "Download"},
			{"SoftwareDistribution", "DeliveryOptimization"},
			{"Logs"},
			{"Prefetch"},
		} {
			if root, ok := pathRoot(systemRoot, model.CategoryWindowsUpdate, model.RiskMedium, false, parts...); ok {
				roots = append(roots, root)
			}
		}
	}

	if selected[model.CategoryWindowsLogs] {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		// The DiagTrack / WMI AutoLogger trace directory. The live ".etl"
		// session files are kept open by the tracing service (and the RtBackup
		// boot-trace logs are locked too), so isRotatedETL surfaces only the
		// rotated ".etl.<suffix>" segments, which are safe to delete.
		if root, ok := pathRoot(systemRoot, model.CategoryWindowsLogs, model.RiskMedium, true, "System32", "LogFiles", "WMI"); ok {
			root.Filter = isRotatedETL
			root.SkipDir = isLockedWMISubdir
			roots = append(roots, root)
		}
	}

	if selected[model.CategoryAppCache] {
		// Third-party application caches that regenerate on demand. Resolved
		// under %ProgramData% and skipped when the app is not installed.
		programData := os.Getenv("ProgramData")
		for _, parts := range [][]string{
			{"Thunder Network", "XLLiveUD", "Download"},
		} {
			if root, ok := pathRoot(programData, model.CategoryAppCache, model.RiskLow, true, parts...); ok {
				roots = append(roots, root)
			}
		}
	}

	return RuleSet{Roots: normalizeExistingRoots(roots)}
}

// isRotatedETL reports whether name is a rotated/archived ETW trace segment
// (e.g. "Diagtrack-Listener.etl.001") rather than a live ".etl" session log.
// Live logs are held open by the tracing service and cannot be deleted, so only
// the rotated segments are eligible for cleanup.
func isRotatedETL(name string, _ int64) bool {
	lower := strings.ToLower(name)
	const marker = ".etl."
	idx := strings.Index(lower, marker)
	return idx >= 0 && idx+len(marker) < len(lower)
}

// isLockedWMISubdir reports whether name is a WMI trace subdirectory that the OS
// keeps locked and that never holds rotated segments we can clean. RtBackup
// stores real-time boot-trace logs and always denies access, so descending into
// it only produces "access denied" scan noise.
func isLockedWMISubdir(name string) bool {
	return strings.EqualFold(name, "RtBackup")
}

// pathRoot builds a Root by joining base with parts. It returns ok=false when
// base is empty or the resulting path does not exist on disk, so optional
// caches the user does not have never surface as "path not found" scan failures.
func pathRoot(base string, category model.CleanCategory, risk model.RiskLevel, defaultSelected bool, parts ...string) (Root, bool) {
	if base == "" {
		return Root{}, false
	}
	full := filepath.Join(append([]string{base}, parts...)...)
	if _, err := os.Stat(full); err != nil {
		return Root{}, false
	}
	return Root{
		Path:            full,
		Category:        category,
		DefaultSelected: defaultSelected,
		Risk:            risk,
	}, true
}

// envRoot resolves base from the named environment variable, then delegates to
// pathRoot.
func envRoot(env string, category model.CleanCategory, risk model.RiskLevel, defaultSelected bool, parts ...string) (Root, bool) {
	return pathRoot(os.Getenv(env), category, risk, defaultSelected, parts...)
}

func categorySet(categories []model.CleanCategory) map[model.CleanCategory]bool {
	all := []model.CleanCategory{
		model.CategoryUserTemp,
		model.CategorySystemTemp,
		model.CategoryChromeCache,
		model.CategoryEdgeCache,
		model.CategoryEdgeIndexedDB,
		model.CategoryVSCodeCache,
		model.CategoryWindowsCache,
		model.CategoryDevCache,
		model.CategoryWindowsUpdate,
		model.CategoryWindowsLogs,
		model.CategoryAppCache,
		model.CategoryRecycleBin,
	}
	set := make(map[model.CleanCategory]bool, len(all))
	if len(categories) == 0 {
		for _, category := range all {
			set[category] = true
		}
		return set
	}
	for _, category := range categories {
		set[category] = true
	}
	return set
}

func browserRoots(userData string, category model.CleanCategory) []Root {
	if userData == "" {
		return nil
	}

	entries, err := os.ReadDir(userData)
	if err != nil {
		return nil
	}

	roots := make([]Root, 0, len(entries)*4)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name != "Default" && !strings.HasPrefix(name, "Profile ") {
			continue
		}
		profile := filepath.Join(userData, name)
		for _, rel := range []string{
			"Cache",
			"Code Cache",
			"GPUCache",
			filepath.Join("Service Worker", "CacheStorage"),
		} {
			roots = append(roots, Root{
				Path:            filepath.Join(profile, rel),
				Category:        category,
				DefaultSelected: true,
				Risk:            model.RiskLow,
			})
		}
	}
	return roots
}

// largeBlobThreshold is the minimum size for an IndexedDB blob to surface as a
// cleanup item. Browsers store large site data (cached media, offline assets)
// as blob files under IndexedDB; only the big ones are worth reclaiming.
const largeBlobThreshold = 50 << 20 // 50 MiB

// edgeIndexedDBRoots returns one root per browser profile pointing at its
// IndexedDB store, surfacing only blob files at or above largeBlobThreshold.
// IndexedDB holds real site data rather than disposable cache, so the roots are
// medium-risk and not selected by default.
func edgeIndexedDBRoots(userData string) []Root {
	if userData == "" {
		return nil
	}

	entries, err := os.ReadDir(userData)
	if err != nil {
		return nil
	}

	roots := make([]Root, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name != "Default" && !strings.HasPrefix(name, "Profile ") {
			continue
		}
		indexedDB := filepath.Join(userData, name, "IndexedDB")
		if _, err := os.Stat(indexedDB); err != nil {
			continue
		}
		roots = append(roots, Root{
			Path:            indexedDB,
			Category:        model.CategoryEdgeIndexedDB,
			DefaultSelected: false,
			Risk:            model.RiskMedium,
			Filter:          isLargeBlob,
		})
	}
	return roots
}

// isLargeBlob keeps only files at or above largeBlobThreshold, regardless of
// name, since IndexedDB blob files are typically stored without an extension.
func isLargeBlob(_ string, sizeBytes int64) bool {
	return sizeBytes >= largeBlobThreshold
}

func normalizeExistingRoots(roots []Root) []Root {
	normalized := make([]Root, 0, len(roots))
	seen := map[string]bool{}
	for _, root := range roots {
		if root.Path == "" {
			continue
		}
		abs, err := filepath.Abs(root.Path)
		if err != nil {
			continue
		}
		clean := filepath.Clean(abs)
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		root.Path = clean
		normalized = append(normalized, root)
	}
	return normalized
}

func InAllowedRoot(path string, roots []Root) bool {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	cleanPath = strings.ToLower(filepath.Clean(cleanPath))

	for _, root := range roots {
		cleanRoot, err := filepath.Abs(root.Path)
		if err != nil {
			continue
		}
		cleanRoot = strings.ToLower(filepath.Clean(cleanRoot))
		if cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func IncludesCategory(categories []model.CleanCategory, target model.CleanCategory) bool {
	return categorySet(categories)[target]
}
