package main

import (
	"context"

	"cleanapp/internal/cleaner"
)

type App struct {
	ctx     context.Context
	service *cleaner.Service
}

func NewApp() *App {
	return &App{service: cleaner.NewService()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Scan(options cleaner.ScanOptions) (cleaner.ScanResult, error) {
	return a.service.Scan(a.ctx, options)
}

func (a *App) Clean(request cleaner.CleanRequest) (cleaner.CleanResult, error) {
	return a.service.Clean(a.ctx, request)
}

func (a *App) AnalyzeDiskGrowth(options cleaner.DiskGrowthOptions) (cleaner.DiskGrowthResult, error) {
	return a.service.AnalyzeDiskGrowth(a.ctx, options)
}

func (a *App) CleanGrowthPaths(request cleaner.GrowthCleanRequest) (cleaner.CleanResult, error) {
	return a.service.CleanGrowthPaths(a.ctx, request)
}

func (a *App) TakeSnapshot(drive string, label string) (cleaner.DiskSnapshot, error) {
	return a.service.TakeSnapshot(a.ctx, drive, label)
}

func (a *App) ListSnapshots() ([]cleaner.SnapshotInfo, error) {
	return a.service.ListSnapshots()
}

func (a *App) CompareSnapshots(oldID string, newID string) (cleaner.SnapshotCompareResult, error) {
	return a.service.CompareSnapshots(oldID, newID)
}

func (a *App) CompareSnapshotPath(oldID string, newID string, path string) (cleaner.SnapshotPathCompareResult, error) {
	return a.service.CompareSnapshotPath(oldID, newID, path)
}

func (a *App) DeleteSnapshot(id string) error {
	return a.service.DeleteSnapshot(id)
}

func (a *App) OpenInExplorer(path string) error {
	return cleaner.OpenInExplorer(path)
}

func (a *App) CancelTask(taskID string) error {
	return a.service.CancelTask(taskID)
}
