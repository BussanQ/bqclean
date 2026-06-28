package rules

import (
	"os"
	"path/filepath"
	"testing"

	"cleanapp/internal/cleaner/model"
)

func TestInAllowedRootNormalizesPaths(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "cache")
	target := filepath.Join(root, "child", "..", "file.tmp")
	outside := filepath.Join(temp, "cache-other", "file.tmp")

	if !InAllowedRoot(target, []Root{{Path: root}}) {
		t.Fatalf("expected %q to be inside %q", target, root)
	}
	if InAllowedRoot(outside, []Root{{Path: root}}) {
		t.Fatalf("expected sibling path %q to be rejected", outside)
	}
}

func TestDefaultChromeRulesOnlyIncludeCacheRoots(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("LOCALAPPDATA", temp)

	profile := filepath.Join(temp, "Google", "Chrome", "User Data", "Default")
	for _, dir := range []string{"Cache", "Code Cache", "GPUCache", filepath.Join("Service Worker", "CacheStorage"), "Cookies"} {
		if err := os.MkdirAll(filepath.Join(profile, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryChromeCache})
	if len(ruleSet.Roots) != 4 {
		t.Fatalf("expected 4 chrome cache roots, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}

	cookies := filepath.Join(profile, "Cookies")
	for _, root := range ruleSet.Roots {
		if filepath.Clean(root.Path) == filepath.Clean(cookies) {
			t.Fatalf("cookies directory must not be included in cleanup roots")
		}
	}
}

func TestDefaultWindowsCacheRulesIncludeExistingCachesOnly(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("LOCALAPPDATA", temp)

	// Create three of the four candidate roots; the missing one must be skipped.
	for _, rel := range []string{
		filepath.Join("Microsoft", "Windows", "Explorer"),
		filepath.Join("Microsoft", "Windows", "INetCache"),
		"CrashDumps",
	} {
		if err := os.MkdirAll(filepath.Join(temp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryWindowsCache})
	if len(ruleSet.Roots) != 3 {
		t.Fatalf("expected 3 windows cache roots, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	for _, root := range ruleSet.Roots {
		if root.Category != model.CategoryWindowsCache {
			t.Fatalf("expected category %q, got %q", model.CategoryWindowsCache, root.Category)
		}
		if root.Risk != model.RiskLow || !root.DefaultSelected {
			t.Fatalf("expected low-risk default-selected root, got %#v", root)
		}
	}
}

func TestDefaultDevCacheRulesAreLowRiskNotDefaultSelected(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("LOCALAPPDATA", temp)

	if err := os.MkdirAll(filepath.Join(temp, "npm-cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "go-build"), 0o755); err != nil {
		t.Fatal(err)
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryDevCache})
	if len(ruleSet.Roots) != 2 {
		t.Fatalf("expected 2 dev cache roots, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	for _, root := range ruleSet.Roots {
		if root.Category != model.CategoryDevCache {
			t.Fatalf("expected category %q, got %q", model.CategoryDevCache, root.Category)
		}
		if root.Risk != model.RiskLow || root.DefaultSelected {
			t.Fatalf("expected low-risk non-default-selected root, got %#v", root)
		}
	}
}

func TestDefaultWindowsUpdateRulesAreMediumRiskUnderSystemRoot(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("SystemRoot", temp)

	expected := filepath.Join(temp, "SoftwareDistribution", "Download")
	if err := os.MkdirAll(expected, 0o755); err != nil {
		t.Fatal(err)
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryWindowsUpdate})
	if len(ruleSet.Roots) != 1 {
		t.Fatalf("expected 1 windows update root, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	root := ruleSet.Roots[0]
	if filepath.Clean(root.Path) != filepath.Clean(expected) {
		t.Fatalf("expected windows update root %q, got %q", expected, root.Path)
	}
	if root.Category != model.CategoryWindowsUpdate {
		t.Fatalf("expected category %q, got %q", model.CategoryWindowsUpdate, root.Category)
	}
	if root.Risk != model.RiskMedium || root.DefaultSelected {
		t.Fatalf("expected medium-risk non-default-selected root, got %#v", root)
	}
}

func TestDefaultWindowsLogsRuleFiltersRotatedETLOnly(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("SystemRoot", temp)

	wmiDir := filepath.Join(temp, "System32", "LogFiles", "WMI")
	if err := os.MkdirAll(wmiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryWindowsLogs})
	if len(ruleSet.Roots) != 1 {
		t.Fatalf("expected 1 windows logs root, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	root := ruleSet.Roots[0]
	if filepath.Clean(root.Path) != filepath.Clean(wmiDir) {
		t.Fatalf("expected windows logs root %q, got %q", wmiDir, root.Path)
	}
	if root.Category != model.CategoryWindowsLogs {
		t.Fatalf("expected category %q, got %q", model.CategoryWindowsLogs, root.Category)
	}
	if root.Risk != model.RiskMedium || !root.DefaultSelected {
		t.Fatalf("expected medium-risk default-selected root, got %#v", root)
	}
	if root.Filter == nil {
		t.Fatal("expected windows logs root to carry a filter")
	}
	if root.SkipDir == nil {
		t.Fatal("expected windows logs root to skip locked subdirs")
	}
	if !root.SkipDir("RtBackup") || !root.SkipDir("rtbackup") {
		t.Fatal("expected RtBackup subdir to be skipped (case-insensitive)")
	}
	if root.SkipDir("SomethingElse") {
		t.Fatal("expected unrelated subdir not to be skipped")
	}

	keep := []string{"Diagtrack-Listener.etl.001", "Diagtrack-Listener.etl.0001", "LwtNetLog.etl.bak"}
	for _, name := range keep {
		if !root.Filter(name, 0) {
			t.Fatalf("expected rotated segment %q to be kept", name)
		}
	}
	drop := []string{"Diagtrack-Listener.etl", "RadioMgr.etl", "Diagtrack-Listener.etl.", "notes.txt"}
	for _, name := range drop {
		if root.Filter(name, 0) {
			t.Fatalf("expected %q to be skipped", name)
		}
	}
}

func TestDefaultWindowsLogsRuleSkippedWhenDirMissing(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("SystemRoot", temp)

	ruleSet := Default([]model.CleanCategory{model.CategoryWindowsLogs})
	if len(ruleSet.Roots) != 0 {
		t.Fatalf("expected no roots when WMI dir is absent, got %#v", ruleSet.Roots)
	}
}

func TestDefaultAppCacheRuleResolvesUnderProgramData(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("ProgramData", temp)

	expected := filepath.Join(temp, "Thunder Network", "XLLiveUD", "Download")
	if err := os.MkdirAll(expected, 0o755); err != nil {
		t.Fatal(err)
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryAppCache})
	if len(ruleSet.Roots) != 1 {
		t.Fatalf("expected 1 app cache root, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	root := ruleSet.Roots[0]
	if filepath.Clean(root.Path) != filepath.Clean(expected) {
		t.Fatalf("expected app cache root %q, got %q", expected, root.Path)
	}
	if root.Category != model.CategoryAppCache {
		t.Fatalf("expected category %q, got %q", model.CategoryAppCache, root.Category)
	}
	if root.Risk != model.RiskLow || !root.DefaultSelected {
		t.Fatalf("expected low-risk default-selected root, got %#v", root)
	}
}

func TestDefaultAppCacheRuleSkippedWhenDirMissing(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("ProgramData", temp)

	ruleSet := Default([]model.CleanCategory{model.CategoryAppCache})
	if len(ruleSet.Roots) != 0 {
		t.Fatalf("expected no roots when app cache dir is absent, got %#v", ruleSet.Roots)
	}
}

func TestDefaultVSCodeRulesIncludeCachedExtensionVSIXs(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("APPDATA", temp)

	expected := filepath.Join(temp, "Code", "CachedExtensionVSIXs")
	ruleSet := Default([]model.CleanCategory{model.CategoryVSCodeCache})

	if len(ruleSet.Roots) != 1 {
		t.Fatalf("expected 1 vscode cache root, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
	if filepath.Clean(ruleSet.Roots[0].Path) != filepath.Clean(expected) {
		t.Fatalf("expected vscode cache root %q, got %q", expected, ruleSet.Roots[0].Path)
	}
	if ruleSet.Roots[0].Category != model.CategoryVSCodeCache {
		t.Fatalf("expected category %q, got %q", model.CategoryVSCodeCache, ruleSet.Roots[0].Category)
	}
}

func TestDefaultEdgeIndexedDBRuleFiltersLargeBlobsOnly(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("LOCALAPPDATA", temp)

	userData := filepath.Join(temp, "Microsoft", "Edge", "User Data")
	for _, profile := range []string{"Default", "Profile 1"} {
		if err := os.MkdirAll(filepath.Join(userData, profile, "IndexedDB"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryEdgeIndexedDB})
	if len(ruleSet.Roots) != 2 {
		t.Fatalf("expected 2 edge indexeddb roots, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}

	for _, root := range ruleSet.Roots {
		if root.Category != model.CategoryEdgeIndexedDB {
			t.Fatalf("expected category %q, got %q", model.CategoryEdgeIndexedDB, root.Category)
		}
		if root.Risk != model.RiskMedium || root.DefaultSelected {
			t.Fatalf("expected medium-risk non-default-selected root, got %#v", root)
		}
		if root.Filter == nil {
			t.Fatal("expected edge indexeddb root to carry a size filter")
		}
		if !root.Filter("0000123.blob", 50<<20) || !root.Filter("anything", 64<<20) {
			t.Fatal("expected files at or above 50 MiB to be kept regardless of name")
		}
		if root.Filter("0000123.blob", (50<<20)-1) {
			t.Fatal("expected files under 50 MiB to be skipped")
		}
	}
}

func TestDefaultEdgeIndexedDBRuleSkippedWhenDirMissing(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("LOCALAPPDATA", temp)

	// A profile directory without an IndexedDB store must not surface a root.
	if err := os.MkdirAll(filepath.Join(temp, "Microsoft", "Edge", "User Data", "Default"), 0o755); err != nil {
		t.Fatal(err)
	}

	ruleSet := Default([]model.CleanCategory{model.CategoryEdgeIndexedDB})
	if len(ruleSet.Roots) != 0 {
		t.Fatalf("expected no edge indexeddb roots, got %d: %#v", len(ruleSet.Roots), ruleSet.Roots)
	}
}
