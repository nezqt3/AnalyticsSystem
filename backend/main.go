package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

//go:embed assets/tracker.js
var trackerJS []byte

type server struct {
	db       *sql.DB
	isSQLite bool
}

type site struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type collectEvent struct {
	SiteID      int64   `json:"site_id"`
	EventType   string  `json:"event_type"`
	Path        string  `json:"path"`
	Title       string  `json:"title"`
	Referrer    string  `json:"referrer"`
	Source      string  `json:"source"`
	UtmSource   string  `json:"utm_source"`
	UtmMedium   string  `json:"utm_medium"`
	UtmCampaign string  `json:"utm_campaign"`
	EntryURL    string  `json:"entry_url"`
	Meta        string  `json:"meta"`
	ScreenW     int     `json:"screen_w"`
	ScreenH     int     `json:"screen_h"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	SessionID   string  `json:"session_id"`
	UserID      string  `json:"user_id"`
}

type realtimePoint struct {
	Minute string `json:"minute"`
	Count  int    `json:"count"`
}

type realtimeResp struct {
	ActiveUsers int             `json:"active_users"`
	Series      []realtimePoint `json:"series"`
}

type heatmapPoint struct {
	XBucket int `json:"x_pct"`
	YBucket int `json:"y_pct"`
	Count   int `json:"count"`
}

type trafficSource struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type eventItem struct {
	CreatedAt   string `json:"created_at"`
	EventType   string `json:"event_type"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Meta        string `json:"meta"`
	RefDomain   string `json:"ref_domain"`
	UtmSource   string `json:"utm_source"`
	UtmMedium   string `json:"utm_medium"`
	UtmCampaign string `json:"utm_campaign"`
}

func main() {
	ctx := context.Background()

	db, isSQLite, err := openDatabase()
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	if err := applySchema(ctx, db, isSQLite); err != nil {
		log.Fatalf("apply schema: %v", err)
	}

	s := &server{db: db, isSQLite: isSQLite}
	if !isSQLite {
		if err := ensureEventPartitions(ctx, db, 3); err != nil {
			log.Fatalf("ensure partitions: %v", err)
		}
	}
	startMaintenance(ctx, s)

	mux := http.NewServeMux()
	mux.HandleFunc("/collect", s.handleCollect)
	mux.HandleFunc("/tracker.js", s.handleTracker)
	mux.HandleFunc("/api/sites", s.handleSites)
	mux.HandleFunc("/api/realtime", s.handleRealtime)
	mux.HandleFunc("/ws/realtime", s.handleRealtimeWS)
	mux.HandleFunc("/api/heatmap", s.handleHeatmap)
	mux.HandleFunc("/api/traffic-sources", s.handleTrafficSources)
	mux.HandleFunc("/api/events", s.handleEvents)

	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func openDatabase() (*sql.DB, bool, error) {
	if os.Getenv("DATABASE_URL") != "" {
		db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
		if err != nil {
			return nil, false, err
		}
		if err := db.Ping(); err != nil {
			return nil, false, err
		}
		return db, false, nil
	}

	path := os.Getenv("SQLITE_PATH")
	if path == "" {
		path = "./data/analytics.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, true, err
	}
	// SQLite DSN
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, true, err
	}
	if err := db.Ping(); err != nil {
		return nil, true, err
	}
	return db, true, nil
}

func applySchema(ctx context.Context, db *sql.DB, isSQLite bool) error {
	var path string
	if isSQLite {
		path = resolveSchemaPath("schema.sqlite.sql")
	} else {
		path = resolveSchemaPath("schema.pg.sql")
	}
	if path == "" {
		return fmt.Errorf("schema file not found")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, string(content))
	return err
}

func resolveSchemaPath(name string) string {
	candidates := []string{
		filepath.Join("db", name),
		filepath.Join("backend", "db", name),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func startMaintenance(ctx context.Context, s *server) {
	rawRetentionDays := envInt("RAW_RETENTION_DAYS", 30)
	aggRetentionMonths := envInt("AGG_RETENTION_MONTHS", 12)

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			if !s.isSQLite {
				if err := ensureEventPartitions(ctx, s.db, 3); err != nil {
					log.Printf("maintenance: ensure partitions: %v", err)
				}
			}
			if err := rebuildDailyAgg(ctx, s.db, s.isSQLite, 2); err != nil {
				log.Printf("maintenance: rebuild daily agg: %v", err)
			}
			if err := applyRetention(ctx, s.db, s.isSQLite, rawRetentionDays, aggRetentionMonths); err != nil {
				log.Printf("maintenance: retention: %v", err)
			}

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
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

func (s *server) handleTracker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(trackerJS)
}

func (s *server) handleSites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		var in site
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Domain) == "" {
			http.Error(w, "name and domain required", http.StatusBadRequest)
			return
		}
		row := s.db.QueryRowContext(ctx, `insert into sites (name, domain) values ($1, $2) returning id`, in.Name, in.Domain)
		if s.isSQLite {
			row = s.db.QueryRowContext(ctx, `insert into sites (name, domain) values (?, ?)`, in.Name, in.Domain)
		}
		if err := row.Scan(&in.ID); err != nil {
			if s.isSQLite {
				idRow := s.db.QueryRowContext(ctx, `select last_insert_rowid()`)
				if err := idRow.Scan(&in.ID); err != nil {
					http.Error(w, "db error", http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, in)
	case http.MethodGet:
		rows, err := s.db.QueryContext(ctx, `select id, name, domain from sites order by id`)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var out []site
		for rows.Next() {
			var s site
			if err := rows.Scan(&s.ID, &s.Name, &s.Domain); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			out = append(out, s)
		}
		writeJSON(w, out)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *server) handleCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	events, err := decodeEvents(body)
	if err != nil || len(events) == 0 {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, ev := range events {
		if ev.SiteID == 0 || ev.EventType == "" || ev.Path == "" {
			http.Error(w, "site_id, event_type, path required", http.StatusBadRequest)
			return
		}
		refDomain := normalizeRefDomain(ev.Referrer)
		utmSource, utmMedium, utmCampaign := normalizeUTM(ev)
		entryURL := normalizeEntryURL(ev)

		if s.isSQLite {
			_, err = tx.ExecContext(r.Context(), `
				insert into events
				(site_id, event_type, path, title, referrer, ref_domain, source, utm_source, utm_medium, utm_campaign, entry_url, meta, screen_w, screen_h, x, y, session_id, user_id, ip, user_agent)
				values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			`, ev.SiteID, ev.EventType, ev.Path, ev.Title, ev.Referrer, refDomain, ev.Source, utmSource, utmMedium, utmCampaign, entryURL, ev.Meta, ev.ScreenW, ev.ScreenH, ev.X, ev.Y, nullIfEmpty(ev.SessionID), nullIfEmpty(ev.UserID), ip, ua)
		} else {
			_, err = tx.ExecContext(r.Context(), `
				insert into events
				(site_id, event_type, path, title, referrer, ref_domain, source, utm_source, utm_medium, utm_campaign, entry_url, meta, screen_w, screen_h, x, y, session_id, user_id, ip, user_agent)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			`, ev.SiteID, ev.EventType, ev.Path, ev.Title, ev.Referrer, refDomain, ev.Source, utmSource, utmMedium, utmCampaign, entryURL, ev.Meta, ev.ScreenW, ev.ScreenH, ev.X, ev.Y, nullIfEmpty(ev.SessionID), nullIfEmpty(ev.UserID), ip, ua)
		}
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleRealtime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}

	resp, err := s.fetchRealtime(r.Context(), siteID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp)
}

func (s *server) handleRealtimeWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		resp, err := s.fetchRealtime(r.Context(), siteID)
		if err != nil {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "db error"))
			return
		}
		if err := conn.WriteJSON(resp); err != nil {
			return
		}

		select {
		case <-ticker.C:
			continue
		case <-r.Context().Done():
			return
		}
	}
}

func (s *server) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	bucket := envInt("HEATMAP_BUCKET_PCT", 5)
	if v := r.URL.Query().Get("bucket"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 25 {
			bucket = n
		}
	}

	var rows *sql.Rows
	if s.isSQLite {
		rows, err = s.db.QueryContext(r.Context(), `
			select
				cast(((x * 100.0 / screen_w) / ?) as int) * ? as x_bucket,
				cast(((y * 100.0 / screen_h) / ?) as int) * ? as y_bucket,
				count(*)
			from events
			where site_id = ? and path = ? and event_type = 'click'
			and x is not null and y is not null
			and screen_w > 0 and screen_h > 0
			group by 1,2
			order by 3 desc
		`, bucket, bucket, bucket, bucket, siteID, path)
	} else {
		rows, err = s.db.QueryContext(r.Context(), `
			select
				floor(((x / screen_w) * 100) / $3)::int * $3 as x_bucket,
				floor(((y / screen_h) * 100) / $3)::int * $3 as y_bucket,
				count(*)
			from events
			where site_id = $1 and path = $2 and event_type = 'click'
			and x is not null and y is not null
			and screen_w > 0 and screen_h > 0
			group by 1,2
			order by 3 desc
		`, siteID, path, bucket)
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []heatmapPoint
	for rows.Next() {
		var p heatmapPoint
		if err := rows.Scan(&p.XBucket, &p.YBucket, &p.Count); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		out = append(out, p)
	}
	writeJSON(w, out)
}

func (s *server) handleTrafficSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}
	path := r.URL.Query().Get("path")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	var rows *sql.Rows
	if s.isSQLite {
		if path != "" {
			rows, err = s.db.QueryContext(r.Context(), `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = ?
				  and event_type = 'pageview'
				  and path = ?
				  and created_at >= datetime(?)
				  and created_at < datetime(?, '+1 day')
				group by 1
				order by 2 desc
				limit 20
			`, siteID, path, from, to)
		} else {
			rows, err = s.db.QueryContext(r.Context(), `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = ?
				  and event_type = 'pageview'
				  and created_at >= datetime(?)
				  and created_at < datetime(?, '+1 day')
				group by 1
				order by 2 desc
				limit 20
			`, siteID, from, to)
		}
	} else {
		if path != "" {
			rows, err = s.db.QueryContext(r.Context(), `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = $1
				  and event_type = 'pageview'
				  and path = $2
				  and created_at >= $3::date
				  and created_at < ($4::date + interval '1 day')
				group by 1
				order by 2 desc
				limit 20
			`, siteID, path, from, to)
		} else {
			rows, err = s.db.QueryContext(r.Context(), `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = $1
				  and event_type = 'pageview'
				  and created_at >= $2::date
				  and created_at < ($3::date + interval '1 day')
				group by 1
				order by 2 desc
				limit 20
			`, siteID, from, to)
		}
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []trafficSource
	for rows.Next() {
		var t trafficSource
		if err := rows.Scan(&t.Source, &t.Count); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		out = append(out, t)
	}
	writeJSON(w, out)
}

func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	siteID, err := parseInt64(r.URL.Query().Get("site_id"))
	if err != nil || siteID == 0 {
		http.Error(w, "site_id required", http.StatusBadRequest)
		return
	}
	limit := envInt("EVENTS_LIMIT", 200)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	path := r.URL.Query().Get("path")
	eventType := r.URL.Query().Get("event_type")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	var rows *sql.Rows
	if s.isSQLite {
		rows, err = s.db.QueryContext(r.Context(), `
			select created_at, event_type, path, title, coalesce(meta, ''), coalesce(ref_domain, ''), coalesce(utm_source, ''), coalesce(utm_medium, ''), coalesce(utm_campaign, '')
			from events
			where site_id = ?
			  and (? = '' or path = ?)
			  and (? = '' or event_type = ?)
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			order by created_at desc
			limit ?
		`, siteID, path, path, eventType, eventType, from, from, to, to, limit)
	} else {
		rows, err = s.db.QueryContext(r.Context(), `
			select to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at,
			       event_type,
			       path,
			       title,
			       coalesce(meta, ''),
			       coalesce(ref_domain, ''),
			       coalesce(utm_source, ''),
			       coalesce(utm_medium, ''),
			       coalesce(utm_campaign, '')
			from events
			where site_id = $1
			  and ($2 = '' or path = $2)
			  and ($3 = '' or event_type = $3)
			  and ($4 = '' or created_at >= $4::date)
			  and ($5 = '' or created_at < ($5::date + interval '1 day'))
			order by created_at desc
			limit $6
		`, siteID, path, eventType, from, to, limit)
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []eventItem
	for rows.Next() {
		var e eventItem
		if err := rows.Scan(&e.CreatedAt, &e.EventType, &e.Path, &e.Title, &e.Meta, &e.RefDomain, &e.UtmSource, &e.UtmMedium, &e.UtmCampaign); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		out = append(out, e)
	}
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func decodeEvents(body []byte) ([]collectEvent, error) {
	var batch []collectEvent
	if err := json.Unmarshal(body, &batch); err == nil {
		return batch, nil
	}
	var single collectEvent
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []collectEvent{single}, nil
}

func (s *server) fetchRealtime(ctx context.Context, siteID int64) (realtimeResp, error) {
	var rows *sql.Rows
	var err error
	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select strftime('%H:%M', created_at) as minute, count(*)
			from events
			where site_id = ? and created_at >= datetime('now', '-5 minutes')
			group by 1
			order by 1
		`, siteID)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			select to_char(date_trunc('minute', created_at), 'HH24:MI') as minute, count(*)
			from events
			where site_id = $1 and created_at >= now() - interval '5 minutes'
			group by 1
			order by 1
		`, siteID)
	}
	if err != nil {
		return realtimeResp{}, err
	}
	defer rows.Close()

	var series []realtimePoint
	for rows.Next() {
		var p realtimePoint
		if err := rows.Scan(&p.Minute, &p.Count); err != nil {
			return realtimeResp{}, err
		}
		series = append(series, p)
	}

	var activeUsers int
	if s.isSQLite {
		row := s.db.QueryRowContext(ctx, `
			select count(distinct session_id)
			from events
			where site_id = ? and created_at >= datetime('now', '-5 minutes') and session_id is not null
		`, siteID)
		if err := row.Scan(&activeUsers); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return realtimeResp{}, err
		}
	} else {
		row := s.db.QueryRowContext(ctx, `
			select count(distinct session_id)
			from events
			where site_id = $1 and created_at >= now() - interval '5 minutes' and session_id is not null
		`, siteID)
		if err := row.Scan(&activeUsers); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return realtimeResp{}, err
		}
	}

	return realtimeResp{ActiveUsers: activeUsers, Series: series}, nil
}

func parseInt64(v string) (int64, error) {
	if v == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseInt(v, 10, 64)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func ensureEventPartitions(ctx context.Context, db *sql.DB, monthsAhead int) error {
	if monthsAhead < 1 {
		monthsAhead = 1
	}
	now := time.Now().UTC()
	for i := 0; i < monthsAhead; i++ {
		m := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, i, 0)
		start := m.Format("2006-01-02")
		end := m.AddDate(0, 1, 0).Format("2006-01-02")
		partName := fmt.Sprintf("events_%04d_%02d", m.Year(), int(m.Month()))

		_, err := db.ExecContext(ctx, fmt.Sprintf(`
			create table if not exists %s
			partition of events
			for values from ('%s') to ('%s')
		`, partName, start, end))
		if err != nil {
			return err
		}
	}
	return nil
}

func rebuildDailyAgg(ctx context.Context, db *sql.DB, isSQLite bool, days int) error {
	if days < 1 {
		days = 1
	}
	now := time.Now().UTC()
	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		if isSQLite {
			if _, err := db.ExecContext(ctx, `delete from event_daily where day = ?`, day); err != nil {
				return err
			}
			_, err := db.ExecContext(ctx, `
				insert into event_daily (day, site_id, event_type, path, count)
				select date(created_at) as day,
				       site_id,
				       event_type,
				       coalesce(path, '') as path,
				       count(*)
				from events
				where created_at >= datetime(?)
				  and created_at < datetime(?, '+1 day')
				group by 1,2,3,4
			`, day, day)
			if err != nil {
				return err
			}
		} else {
			if _, err := db.ExecContext(ctx, `delete from event_daily where day = $1`, day); err != nil {
				return err
			}
			_, err := db.ExecContext(ctx, `
				insert into event_daily (day, site_id, event_type, path, count)
				select date_trunc('day', created_at)::date as day,
				       site_id,
				       event_type,
				       coalesce(path, '') as path,
				       count(*)
				from events
				where created_at >= $1::date
				  and created_at < ($1::date + interval '1 day')
				group by 1,2,3,4
			`, day)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func applyRetention(ctx context.Context, db *sql.DB, isSQLite bool, rawDays int, aggMonths int) error {
	if rawDays < 1 {
		rawDays = 30
	}
	if aggMonths < 1 {
		aggMonths = 12
	}
	if isSQLite {
		if _, err := db.ExecContext(ctx, `delete from events where created_at < datetime('now', ? || ' days')`, fmt.Sprintf("-%d", rawDays)); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `delete from event_daily where day < date('now', ? || ' months')`, fmt.Sprintf("-%d", aggMonths)); err != nil {
			return err
		}
		return nil
	}
	if _, err := db.ExecContext(ctx, `delete from events where created_at < now() - ($1 * interval '1 day')`, rawDays); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `delete from event_daily where day < (current_date - ($1 * interval '1 month'))`, aggMonths); err != nil {
		return err
	}
	return nil
}

func normalizeRefDomain(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Host)
}

func normalizeUTM(ev collectEvent) (string, string, string) {
	us := strings.TrimSpace(ev.UtmSource)
	um := strings.TrimSpace(ev.UtmMedium)
	uc := strings.TrimSpace(ev.UtmCampaign)
	return strings.ToLower(us), strings.ToLower(um), strings.ToLower(uc)
}

func normalizeEntryURL(ev collectEvent) string {
	if strings.TrimSpace(ev.EntryURL) == "" {
		return ""
	}
	return ev.EntryURL
}
