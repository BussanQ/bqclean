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
