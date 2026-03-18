package app

import (
	"context"
	"log"
	"time"

	"analytics-backend/internal/config"
	"analytics-backend/internal/httpapi"
	"analytics-backend/internal/store/sqlstore"
)

type App struct {
	Config config.Config
	Store  *sqlstore.Store
}

func New(ctx context.Context) (*App, error) {
	cfg := config.Load()
	store, err := sqlstore.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := store.EnsureDefaultSite(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return &App{Config: cfg, Store: store}, nil
}

func (a *App) Close() error {
	return a.Store.Close()
}

func (a *App) Router() *httpapi.Handler {
	return httpapi.NewHandler(a.Config, a.Store)
}

func (a *App) StartMaintenance(ctx context.Context) {
	go func() {
		runMaintenance := func() {
			if err := a.Store.EnsureEventPartitions(ctx, 3); err != nil {
				log.Printf("maintenance: ensure partitions: %v", err)
			}
			if err := a.Store.RebuildDailyAggregates(ctx, 2); err != nil {
				log.Printf("maintenance: rebuild daily aggregates: %v", err)
			}
			if err := a.Store.ApplyRetention(ctx); err != nil {
				log.Printf("maintenance: retention: %v", err)
			}
		}

		runMaintenance()
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runMaintenance()
			}
		}
	}()
}
