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

func (a *App) CancelTask(taskID string) error {
	return a.service.CancelTask(taskID)
}
