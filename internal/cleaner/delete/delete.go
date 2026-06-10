package delete

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"cleanapp/internal/cleaner/model"
	"cleanapp/internal/cleaner/rules"
)

type ReparseChecker func(path string, info os.FileInfo) (bool, error)

type Deleter struct {
	IsReparsePoint ReparseChecker
}

func New(isReparsePoint ReparseChecker) Deleter {
	return Deleter{IsReparsePoint: isReparsePoint}
}

func (d Deleter) Delete(ctx context.Context, item model.ScanItem, roots []rules.Root) (deleted bool, skipped bool, failure *model.CleanFailure) {
	if err := ctx.Err(); err != nil {
		return false, false, &model.CleanFailure{Path: item.Path, Reason: err.Error()}
	}
	if item.IsVirtual {
		return false, true, nil
	}
	if !rules.InAllowedRoot(item.Path, roots) {
		return false, true, &model.CleanFailure{Path: item.Path, Reason: "path is outside allowed cleanup roots"}
	}

	info, err := os.Lstat(item.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, true, nil
		}
		return false, false, &model.CleanFailure{Path: item.Path, Reason: err.Error()}
	}

	reparse, err := d.isReparsePoint(item.Path, info)
	if err != nil {
		return false, false, &model.CleanFailure{Path: item.Path, Reason: err.Error()}
	}
	if reparse || info.Mode()&os.ModeSymlink != 0 {
		return false, true, &model.CleanFailure{Path: item.Path, Reason: "refusing to delete symlink or reparse point"}
	}

	if info.IsDir() {
		if err := os.Remove(item.Path); err != nil {
			return false, false, &model.CleanFailure{Path: item.Path, Reason: err.Error()}
		}
		return true, false, nil
	}

	if err := os.Remove(item.Path); err != nil {
		return false, false, &model.CleanFailure{Path: item.Path, Reason: err.Error()}
	}
	removeEmptyParents(filepath.Dir(item.Path), roots)
	return true, false, nil
}

func (d Deleter) isReparsePoint(path string, info os.FileInfo) (bool, error) {
	if d.IsReparsePoint == nil {
		return false, nil
	}
	return d.IsReparsePoint(path, info)
}

func removeEmptyParents(dir string, roots []rules.Root) {
	for rules.InAllowedRoot(dir, roots) {
		if isAllowedRoot(dir, roots) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func isAllowedRoot(path string, roots []rules.Root) bool {
	for _, root := range roots {
		if filepath.Clean(path) == filepath.Clean(root.Path) {
			return true
		}
	}
	return false
}
