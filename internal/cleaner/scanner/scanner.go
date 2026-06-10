package scanner

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"cleanapp/internal/cleaner/model"
	"cleanapp/internal/cleaner/rules"
)

type ReparseChecker func(path string, entry fs.DirEntry) (bool, error)

type Scanner struct {
	IsReparsePoint ReparseChecker
}

func New(isReparsePoint ReparseChecker) Scanner {
	return Scanner{IsReparsePoint: isReparsePoint}
}

func (s Scanner) Scan(ctx context.Context, roots []rules.Root) ([]model.ScanItem, []model.ScanFailure, bool) {
	items := make([]model.ScanItem, 0, 512)
	failures := make([]model.ScanFailure, 0)
	cancelled := false

	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			cancelled = errors.Is(err, context.Canceled)
			break
		}

		err := filepath.WalkDir(root.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				cancelled = errors.Is(err, context.Canceled)
				return err
			}
			if walkErr != nil {
				failures = append(failures, model.ScanFailure{Path: path, Reason: walkErr.Error()})
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			reparse, err := s.isReparsePoint(path, entry)
			if err != nil {
				failures = append(failures, model.ScanFailure{Path: path, Reason: err.Error()})
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if reparse {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				failures = append(failures, model.ScanFailure{Path: path, Reason: err.Error()})
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() {
				return nil
			}

			items = append(items, model.ScanItem{
				Path:            path,
				SizeBytes:       info.Size(),
				ModifiedAt:      info.ModTime().Format(time.RFC3339),
				Category:        root.Category,
				Risk:            root.Risk,
				DefaultSelected: root.DefaultSelected,
				IsDirectory:     false,
			})
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			failures = append(failures, model.ScanFailure{Path: root.Path, Reason: err.Error()})
		}
	}

	return items, failures, cancelled
}

func (s Scanner) isReparsePoint(path string, entry fs.DirEntry) (bool, error) {
	if s.IsReparsePoint == nil {
		return false, nil
	}
	return s.IsReparsePoint(path, entry)
}
