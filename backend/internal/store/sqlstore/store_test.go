package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"analytics-backend/internal/config"
	"analytics-backend/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()
	store, err := Open(ctx, config.Config{
		SQLitePath:         filepath.Join(t.TempDir(), "analytics-test.db"),
		RawRetentionDays:   30,
		AggRetentionMonths: 12,
		HeatmapBucketPct:   5,
		EventsLimit:        200,
	})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close test store: %v", err)
		}
	})

	return store
}

func TestDecodeEventsSupportsBatchAndSingle(t *testing.T) {
	t.Run("single event", func(t *testing.T) {
		events, err := DecodeEvents([]byte(`{"site_id":1,"event_type":"pageview","path":"/"}`))
		if err != nil {
			t.Fatalf("decode single event: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
	})

	t.Run("batch events", func(t *testing.T) {
		events, err := DecodeEvents([]byte(`[{"site_id":1,"event_type":"pageview","path":"/"},{"site_id":1,"event_type":"click","path":"/pricing"}]`))
		if err != nil {
			t.Fatalf("decode batch: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}
	})
}

func TestStoreInsertAndAggregateAnalytics(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	site, err := store.CreateSite(ctx, model.Site{Name: "Main site", Domain: "example.com"})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	events := []model.CollectEvent{
		{
			SiteID:    site.ID,
			EventType: "pageview",
			Path:      "/pricing",
			Title:     "Pricing",
			SessionID: "session-1",
			UtmSource: "google",
			EntryURL:  "https://example.com/pricing",
			Meta:      `{"location":"https://example.com/pricing"}`,
		},
		{
			SiteID:    site.ID,
			EventType: "click",
			Path:      "/pricing",
			Title:     "Pricing",
			SessionID: "session-1",
			ScreenW:   1440,
			ScreenH:   900,
			X:         320,
			Y:         240,
			Meta:      `{"selector":"button.primary","text":"Start trial","tag":"button"}`,
		},
		{
			SiteID:    site.ID,
			EventType: "scroll",
			Path:      "/pricing",
			Title:     "Pricing",
			SessionID: "session-1",
			Meta:      `{"depth_pct":80}`,
		},
		{
			SiteID:    site.ID,
			EventType: "form_submit",
			Path:      "/pricing",
			Title:     "Pricing",
			SessionID: "session-1",
			Meta:      `{"selector":"form#lead"}`,
		},
		{
			SiteID:    site.ID,
			EventType: "pageview",
			Path:      "/blog",
			Title:     "Blog",
			SessionID: "session-2",
			Referrer:  "https://news.ycombinator.com/item?id=1",
		},
	}

	if err := store.InsertEvents(ctx, events, "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	pages, err := store.FetchPages(ctx, model.Filters{SiteID: site.ID, From: "2020-01-01", To: "2099-01-01", Limit: 20})
	if err != nil {
		t.Fatalf("fetch pages: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].Path != "/pricing" {
		t.Fatalf("expected /pricing to lead, got %s", pages[0].Path)
	}

	traffic, err := store.FetchTrafficSources(ctx, model.Filters{SiteID: site.ID, Path: "/pricing", From: "2020-01-01", To: "2099-01-01"})
	if err != nil {
		t.Fatalf("fetch traffic sources: %v", err)
	}
	if len(traffic) == 0 || traffic[0].Source != "google" {
		t.Fatalf("expected google traffic source, got %#v", traffic)
	}

	heatmap, err := store.FetchHeatmap(ctx, model.Filters{SiteID: site.ID, Path: "/pricing", From: "2020-01-01", To: "2099-01-01", Bucket: 5})
	if err != nil {
		t.Fatalf("fetch heatmap: %v", err)
	}
	if len(heatmap) != 1 {
		t.Fatalf("expected 1 heatmap point, got %d", len(heatmap))
	}

	analytics, err := store.FetchPageAnalytics(ctx, model.Filters{SiteID: site.ID, Path: "/pricing", From: "2020-01-01", To: "2099-01-01", Limit: 100})
	if err != nil {
		t.Fatalf("fetch page analytics: %v", err)
	}
	if analytics.Pageviews != 1 || analytics.Clicks != 1 || analytics.FormSubmissions != 1 {
		t.Fatalf("unexpected counters: %#v", analytics)
	}
	if analytics.UniqueVisitors != 1 {
		t.Fatalf("expected 1 unique visitor, got %d", analytics.UniqueVisitors)
	}
	if analytics.AvgScrollDepth != 80 {
		t.Fatalf("expected avg scroll depth 80, got %v", analytics.AvgScrollDepth)
	}
	if len(analytics.TopTargets) != 1 || analytics.TopTargets[0].Selector != "button.primary" {
		t.Fatalf("unexpected top targets: %#v", analytics.TopTargets)
	}
	if len(analytics.ScrollDepths) != 1 || analytics.ScrollDepths[0].Depth != 80 {
		t.Fatalf("unexpected scroll depths: %#v", analytics.ScrollDepths)
	}
}
