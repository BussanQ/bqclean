package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cleanapp/internal/cleaner/model"
	"cleanapp/internal/cleaner/rules"
)

func TestScanSkipsReparsePointSubtree(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "cache")
	linkDir := filepath.Join(root, "linkish")

	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.tmp"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, "skip.tmp"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	scan := New(func(path string) (bool, error) {
		return strings.Contains(path, "linkish"), nil
	})
	items, failures, cancelled := scan.Scan(context.Background(), []rules.Root{{
		Path:            root,
		Category:        model.CategoryUserTemp,
		DefaultSelected: true,
		Risk:            model.RiskLow,
	}})

	if cancelled {
		t.Fatalf("scan should not be cancelled")
	}
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %#v", failures)
	}
	if len(items) != 1 {
		t.Fatalf("expected one scanned file, got %d", len(items))
	}
	if filepath.Base(items[0].Path) != "keep.tmp" {
		t.Fatalf("expected keep.tmp, got %q", items[0].Path)
	}
}
