package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"analytics-backend/internal/config"
	"analytics-backend/internal/model"
	"analytics-backend/internal/store/sqlstore"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	store, err := sqlstore.Open(context.Background(), config.Config{
		SQLitePath:         filepath.Join(t.TempDir(), "analytics-handler.db"),
		RawRetentionDays:   30,
		AggRetentionMonths: 12,
		HeatmapBucketPct:   5,
		EventsLimit:        200,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return NewHandler(config.Config{HeatmapBucketPct: 5, EventsLimit: 200}, store)
}

func createSite(t *testing.T, handler *Handler) model.Site {
	t.Helper()

	body, _ := json.Marshal(model.Site{Name: "Main", Domain: "example.com"})
	request := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("create site failed with status %d: %s", recorder.Code, recorder.Body.String())
	}

	var site model.Site
	if err := json.Unmarshal(recorder.Body.Bytes(), &site); err != nil {
		t.Fatalf("decode site: %v", err)
	}
	return site
}

func TestHandleCollectRejectsInvalidEvent(t *testing.T) {
	handler := newTestHandler(t)

	request := httptest.NewRequest(http.MethodPost, "/collect", bytes.NewBufferString(`{"site_id":0,"event_type":"","path":""}`))
	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestHandleCollectAndPageAnalyticsFlow(t *testing.T) {
	handler := newTestHandler(t)
	site := createSite(t, handler)

	payload := []model.CollectEvent{
		{
			SiteID:    site.ID,
			EventType: "pageview",
			Path:      "/features",
			Title:     "Features",
			SessionID: "session-a",
			UtmSource: "newsletter",
			Meta:      `{"location":"https://example.com/features"}`,
		},
		{
			SiteID:    site.ID,
			EventType: "click",
			Path:      "/features",
			Title:     "Features",
			SessionID: "session-a",
			ScreenW:   1280,
			ScreenH:   800,
			X:         200,
			Y:         160,
			Meta:      `{"selector":"a.cta","text":"Подробнее","href":"/contact"}`,
		},
	}

	body, _ := json.Marshal(payload)
	collectRequest := httptest.NewRequest(http.MethodPost, "/collect", bytes.NewReader(body))
	collectRequest.Header.Set("Content-Type", "application/json")
	collectRecorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(collectRecorder, collectRequest)
	if collectRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", collectRecorder.Code, collectRecorder.Body.String())
	}

	analyticsRequest := httptest.NewRequest(http.MethodGet, "/api/page-analytics?site_id=1&path=/features&from=2020-01-01&to=2099-01-01", nil)
	analyticsRecorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(analyticsRecorder, analyticsRequest)
	if analyticsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", analyticsRecorder.Code, analyticsRecorder.Body.String())
	}

	var analytics model.PageAnalytics
	if err := json.Unmarshal(analyticsRecorder.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("decode analytics: %v", err)
	}

	if analytics.Path != "/features" || analytics.Pageviews != 1 || analytics.Clicks != 1 {
		t.Fatalf("unexpected analytics payload: %#v", analytics)
	}
}
