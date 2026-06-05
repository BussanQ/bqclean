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
