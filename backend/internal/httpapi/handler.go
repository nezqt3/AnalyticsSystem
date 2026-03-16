package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"analytics-backend/internal/auth"
	"analytics-backend/internal/config"
	"analytics-backend/internal/model"
	"analytics-backend/internal/store/sqlstore"
	"analytics-backend/internal/tracker"
)

type Handler struct {
	cfg         config.Config
	store       Store
	authManager *auth.Manager
}

type Store interface {
	ListSites(ctx context.Context) ([]model.Site, error)
	CreateSite(ctx context.Context, input model.Site) (model.Site, error)
	InsertEvents(ctx context.Context, events []model.CollectEvent, ip string, userAgent string) error
	FetchRealtime(ctx context.Context, siteID int64) (model.RealtimeResponse, error)
	FetchHeatmap(ctx context.Context, filters model.Filters) ([]model.HeatmapPoint, error)
	FetchTrafficSources(ctx context.Context, filters model.Filters) ([]model.TrafficSource, error)
	FetchEvents(ctx context.Context, filters model.Filters, eventType string) ([]model.EventItem, error)
	FetchPages(ctx context.Context, filters model.Filters) ([]model.PageStat, error)
	FetchOverview(ctx context.Context, filters model.Filters) (model.OverviewMetrics, error)
	FetchTimeline(ctx context.Context, filters model.Filters) ([]model.TimelinePoint, error)
	FetchRecentVisits(ctx context.Context, filters model.Filters) ([]model.VisitEntry, error)
	FetchPageAnalytics(ctx context.Context, filters model.Filters) (model.PageAnalytics, error)
}

func NewHandler(cfg config.Config, store Store) *Handler {
	return &Handler{
		cfg:         cfg,
		store:       store,
		authManager: auth.NewManager(cfg.AdminEmail, cfg.AdminPassword, cfg.SessionSecret),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/collect", h.handleCollect)
	mux.HandleFunc("/tracker.js", h.handleTracker)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/auth/me", h.handleAuthMe)

	protected := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{path: "/api/sites", handler: h.handleSites},
		{path: "/api/realtime", handler: h.handleRealtime},
		{path: "/ws/realtime", handler: h.handleRealtimeWS},
		{path: "/api/heatmap", handler: h.handleHeatmap},
		{path: "/api/traffic-sources", handler: h.handleTrafficSources},
		{path: "/api/events", handler: h.handleEvents},
		{path: "/api/pages", handler: h.handlePages},
		{path: "/api/page-analytics", handler: h.handlePageAnalytics},
		{path: "/api/overview", handler: h.handleOverview},
		{path: "/api/timeline", handler: h.handleTimeline},
		{path: "/api/visits", handler: h.handleVisits},
	}

	for _, route := range protected {
		mux.Handle(route.path, h.requireAuth(route.handler))
	}

	return withCORS(mux)
}

func (h *Handler) handleTracker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(tracker.Script)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.authManager.Enabled() {
		http.Error(w, "auth is not configured", http.StatusServiceUnavailable)
		return
	}

	var input model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !h.authManager.ValidateCredentials(input.Email, input.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.authManager.CreateToken()
	if err != nil {
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	writeJSON(w, model.AuthUser{Email: strings.TrimSpace(input.Email)})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user, err := h.currentUser(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, model.AuthUser{Email: user.Email})
}

func (h *Handler) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sites, err := h.store.ListSites(r.Context())
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, sites)
	case http.MethodPost:
		var input model.Site
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Domain) == "" {
			http.Error(w, "name and domain required", http.StatusBadRequest)
			return
		}
		created, err := h.store.CreateSite(r.Context(), input)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	events, err := sqlstore.DecodeEvents(body)
	if err != nil || len(events) == 0 {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		if event.SiteID == 0 || strings.TrimSpace(event.EventType) == "" || strings.TrimSpace(event.Path) == "" {
			http.Error(w, "site_id, event_type, path required", http.StatusBadRequest)
			return
		}
	}

	if err := h.store.InsertEvents(r.Context(), events, clientIP(r), r.UserAgent()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRealtime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseSiteID(r)
	if err != nil {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}

	response, err := h.store.FetchRealtime(r.Context(), siteID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, response)
}

func (h *Handler) handleRealtimeWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, err := h.currentUser(r); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	siteID, err := parseSiteID(r)
	if err != nil {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	for {
		response, err := h.store.FetchRealtime(ctx, siteID)
		if err != nil {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "db error"))
			return
		}
		if err := conn.WriteJSON(response); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *Handler) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filters.Bucket = h.cfg.HeatmapBucketPct
	if value := strings.TrimSpace(r.URL.Query().Get("bucket")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 1 && parsed <= 25 {
			filters.Bucket = parsed
		}
	}

	items, err := h.store.FetchHeatmap(r.Context(), filters)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, items)
}

func (h *Handler) handleTrafficSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)

	items, err := h.store.FetchTrafficSources(r.Context(), filters)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, items)
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filters.Limit = h.cfg.EventsLimit
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 1000 {
			filters.Limit = parsed
		}
	}

	items, err := h.store.FetchEvents(r.Context(), filters, strings.TrimSpace(r.URL.Query().Get("event_type")))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, items)
}

func (h *Handler) handlePages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)
	filters.Limit = 200
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 500 {
			filters.Limit = parsed
		}
	}

	items, err := h.store.FetchPages(r.Context(), filters)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, items)
}

func (h *Handler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)

	overview, err := h.store.FetchOverview(r.Context(), filters)
	if err != nil {
		log.Printf("overview query failed: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, overview)
}

func (h *Handler) handleTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)
	filters.Interval = strings.TrimSpace(r.URL.Query().Get("interval"))

	points, err := h.store.FetchTimeline(r.Context(), filters)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, points)
}

func (h *Handler) handleVisits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)
	filters.Limit = 50
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 200 {
			filters.Limit = parsed
		}
	}

	visits, err := h.store.FetchRecentVisits(r.Context(), filters)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, visits)
}

func (h *Handler) handlePageAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filters, err := h.baseFilters(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyDefaultDateRange(&filters)
	filters.Limit = 5000

	analytics, err := h.store.FetchPageAnalytics(r.Context(), filters)
	if err != nil {
		if err.Error() == "path is required" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, analytics)
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.currentUser(r); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) currentUser(r *http.Request) (auth.User, error) {
	if !h.authManager.Enabled() {
		return auth.User{}, fmt.Errorf("auth is not configured")
	}
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		return auth.User{}, err
	}
	return h.authManager.ParseToken(cookie.Value)
}

func (h *Handler) baseFilters(r *http.Request, requirePath bool) (model.Filters, error) {
	siteID, err := parseSiteID(r)
	if err != nil {
		return model.Filters{}, fmt.Errorf("site_id required")
	}
	filters := model.Filters{
		SiteID: siteID,
		Path:   strings.TrimSpace(r.URL.Query().Get("path")),
		From:   strings.TrimSpace(r.URL.Query().Get("from")),
		To:     strings.TrimSpace(r.URL.Query().Get("to")),
	}
	if requirePath && filters.Path == "" {
		return model.Filters{}, fmt.Errorf("path required")
	}
	return filters, nil
}

func parseSiteID(r *http.Request) (int64, error) {
	siteID, err := sqlstore.ParseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		return 0, fmt.Errorf("invalid site_id")
	}
	return siteID, nil
}

func applyDefaultDateRange(filters *model.Filters) {
	if filters.From == "" {
		filters.From = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if filters.To == "" {
		filters.To = time.Now().Format("2006-01-02")
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
