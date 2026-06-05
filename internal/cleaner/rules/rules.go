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

	return RuleSet{Roots: normalizeExistingRoots(roots)}
}

func categorySet(categories []model.CleanCategory) map[model.CleanCategory]bool {
	all := []model.CleanCategory{
		model.CategoryUserTemp,
		model.CategorySystemTemp,
		model.CategoryChromeCache,
		model.CategoryEdgeCache,
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
