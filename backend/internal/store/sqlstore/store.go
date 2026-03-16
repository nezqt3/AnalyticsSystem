package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"analytics-backend/internal/config"
	"analytics-backend/internal/model"
)

type Store struct {
	db       *sql.DB
	isSQLite bool
	cfg      config.Config
}

type clickMeta struct {
	Tag      string `json:"tag"`
	ID       string `json:"id"`
	Class    string `json:"class"`
	Text     string `json:"text"`
	Href     string `json:"href"`
	Selector string `json:"selector"`
	DepthPct int    `json:"depth_pct"`
}

func Open(ctx context.Context, cfg config.Config) (*Store, error) {
	if cfg.DatabaseURL != "" {
		log.Printf("database: using PostgreSQL via DATABASE_URL")
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		if err := db.PingContext(ctx); err != nil {
			return nil, err
		}

		store := &Store{db: db, cfg: cfg}
		if err := store.applySchema(ctx); err != nil {
			return nil, err
		}
		if err := store.EnsureEventPartitions(ctx, 3); err != nil {
			return nil, err
		}
		return store, nil
	}

	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
		return nil, err
	}
	log.Printf("database: using SQLite at %s", cfg.SQLitePath)

	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	store := &Store{db: db, isSQLite: true, cfg: cfg}
	if err := store.applySchema(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) IsSQLite() bool {
	return s.isSQLite
}

func (s *Store) CreateSite(ctx context.Context, input model.Site) (model.Site, error) {
	if s.isSQLite {
		result, err := s.db.ExecContext(ctx, `insert into sites (name, domain) values (?, ?)`, input.Name, input.Domain)
		if err != nil {
			return model.Site{}, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return model.Site{}, err
		}
		input.ID = id
		return input, nil
	}

	if err := s.db.QueryRowContext(ctx, `insert into sites (name, domain) values ($1, $2) returning id`, input.Name, input.Domain).Scan(&input.ID); err != nil {
		return model.Site{}, err
	}
	return input, nil
}

func (s *Store) ListSites(ctx context.Context) ([]model.Site, error) {
	rows, err := s.db.QueryContext(ctx, `select id, name, domain from sites order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []model.Site
	for rows.Next() {
		var site model.Site
		if err := rows.Scan(&site.ID, &site.Name, &site.Domain); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func (s *Store) InsertEvents(ctx context.Context, events []model.CollectEvent, ip string, userAgent string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, event := range events {
		refDomain := normalizeRefDomain(event.Referrer)
		utmSource, utmMedium, utmCampaign := normalizeUTM(event)
		entryURL := strings.TrimSpace(event.EntryURL)

		if s.isSQLite {
			_, err = tx.ExecContext(ctx, `
				insert into events
				(site_id, event_type, path, title, referrer, ref_domain, source, utm_source, utm_medium, utm_campaign, entry_url, meta, screen_w, screen_h, x, y, session_id, user_id, ip, user_agent)
				values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			`, event.SiteID, event.EventType, event.Path, event.Title, event.Referrer, refDomain, event.Source, utmSource, utmMedium, utmCampaign, entryURL, event.Meta, event.ScreenW, event.ScreenH, event.X, event.Y, nullIfEmpty(event.SessionID), nullIfEmpty(event.UserID), ip, userAgent)
		} else {
			_, err = tx.ExecContext(ctx, `
				insert into events
				(site_id, event_type, path, title, referrer, ref_domain, source, utm_source, utm_medium, utm_campaign, entry_url, meta, screen_w, screen_h, x, y, session_id, user_id, ip, user_agent)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			`, event.SiteID, event.EventType, event.Path, event.Title, event.Referrer, refDomain, event.Source, utmSource, utmMedium, utmCampaign, entryURL, event.Meta, event.ScreenW, event.ScreenH, event.X, event.Y, nullIfEmpty(event.SessionID), nullIfEmpty(event.UserID), ip, userAgent)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) FetchRealtime(ctx context.Context, siteID int64) (model.RealtimeResponse, error) {
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
		return model.RealtimeResponse{}, err
	}
	defer rows.Close()

	series := make([]model.RealtimePoint, 0, 8)
	for rows.Next() {
		var point model.RealtimePoint
		if err := rows.Scan(&point.Minute, &point.Count); err != nil {
			return model.RealtimeResponse{}, err
		}
		series = append(series, point)
	}

	var activeUsers int
	var row *sql.Row
	if s.isSQLite {
		row = s.db.QueryRowContext(ctx, `
			select count(distinct session_id)
			from events
			where site_id = ? and created_at >= datetime('now', '-5 minutes') and session_id is not null
		`, siteID)
	} else {
		row = s.db.QueryRowContext(ctx, `
			select count(distinct session_id)
			from events
			where site_id = $1 and created_at >= now() - interval '5 minutes' and session_id is not null
		`, siteID)
	}
	if err := row.Scan(&activeUsers); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.RealtimeResponse{}, err
	}

	return model.RealtimeResponse{ActiveUsers: activeUsers, Series: series}, nil
}

func (s *Store) FetchTrafficSources(ctx context.Context, filters model.Filters) ([]model.TrafficSource, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		if filters.Path != "" {
			rows, err = s.db.QueryContext(ctx, `
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
			`, filters.SiteID, filters.Path, filters.From, filters.To)
		} else {
			rows, err = s.db.QueryContext(ctx, `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = ?
				  and event_type = 'pageview'
				  and created_at >= datetime(?)
				  and created_at < datetime(?, '+1 day')
				group by 1
				order by 2 desc
				limit 20
			`, filters.SiteID, filters.From, filters.To)
		}
	} else {
		if filters.Path != "" {
			rows, err = s.db.QueryContext(ctx, `
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
			`, filters.SiteID, filters.Path, filters.From, filters.To)
		} else {
			rows, err = s.db.QueryContext(ctx, `
				select coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as src, count(*)
				from events
				where site_id = $1
				  and event_type = 'pageview'
				  and created_at >= $2::date
				  and created_at < ($3::date + interval '1 day')
				group by 1
				order by 2 desc
				limit 20
			`, filters.SiteID, filters.From, filters.To)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.TrafficSource, 0, 20)
	for rows.Next() {
		var item model.TrafficSource
		if err := rows.Scan(&item.Source, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FetchHeatmap(ctx context.Context, filters model.Filters) ([]model.HeatmapPoint, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select
				cast(((x * 100.0 / screen_w) / ?) as int) * ? as x_bucket,
				cast(((y * 100.0 / screen_h) / ?) as int) * ? as y_bucket,
				count(*)
			from events
			where site_id = ? and path = ? and event_type = 'click'
			and x is not null and y is not null
			and screen_w > 0 and screen_h > 0
			and (? = '' or created_at >= datetime(?))
			and (? = '' or created_at < datetime(?, '+1 day'))
			group by 1,2
			order by 3 desc
		`, filters.Bucket, filters.Bucket, filters.Bucket, filters.Bucket, filters.SiteID, filters.Path, filters.From, filters.From, filters.To, filters.To)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			select
				floor(((x / screen_w) * 100) / $3)::int * $3 as x_bucket,
				floor(((y / screen_h) * 100) / $3)::int * $3 as y_bucket,
				count(*)
			from events
			where site_id = $1 and path = $2 and event_type = 'click'
			and x is not null and y is not null
			and screen_w > 0 and screen_h > 0
			and ($4 = '' or created_at >= $4::date)
			and ($5 = '' or created_at < ($5::date + interval '1 day'))
			group by 1,2
			order by 3 desc
		`, filters.SiteID, filters.Path, filters.Bucket, filters.From, filters.To)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.HeatmapPoint, 0, 64)
	for rows.Next() {
		var item model.HeatmapPoint
		if err := rows.Scan(&item.XBucket, &item.YBucket, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FetchEvents(ctx context.Context, filters model.Filters, eventType string) ([]model.EventItem, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select created_at, event_type, path, title, coalesce(meta, ''), coalesce(ref_domain, ''), coalesce(utm_source, ''), coalesce(utm_medium, ''), coalesce(utm_campaign, '')
			from events
			where site_id = ?
			  and (? = '' or path = ?)
			  and (? = '' or event_type = ?)
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			order by created_at desc
			limit ?
		`, filters.SiteID, filters.Path, filters.Path, eventType, eventType, filters.From, filters.From, filters.To, filters.To, filters.Limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
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
		`, filters.SiteID, filters.Path, eventType, filters.From, filters.To, filters.Limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.EventItem, 0, filters.Limit)
	for rows.Next() {
		var item model.EventItem
		if err := rows.Scan(&item.CreatedAt, &item.EventType, &item.Path, &item.Title, &item.Meta, &item.RefDomain, &item.UtmSource, &item.UtmMedium, &item.UtmCampaign); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FetchPages(ctx context.Context, filters model.Filters) ([]model.PageStat, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select
				path,
				sum(case when event_type = 'pageview' then 1 else 0 end) as pageviews,
				sum(case when event_type = 'click' then 1 else 0 end) as clicks,
				sum(case when event_type = 'form_submit' then 1 else 0 end) as form_submissions,
				count(distinct case when session_id is not null then session_id end) as unique_visitors,
				max(created_at) as last_seen
			from events
			where site_id = ?
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			group by path
			having path <> ''
			order by pageviews desc, clicks desc, last_seen desc
			limit ?
		`, filters.SiteID, filters.From, filters.From, filters.To, filters.To, filters.Limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			select
				path,
				sum(case when event_type = 'pageview' then 1 else 0 end) as pageviews,
				sum(case when event_type = 'click' then 1 else 0 end) as clicks,
				sum(case when event_type = 'form_submit' then 1 else 0 end) as form_submissions,
				count(distinct session_id) filter (where session_id is not null) as unique_visitors,
				to_char(max(created_at), 'YYYY-MM-DD HH24:MI:SS') as last_seen
			from events
			where site_id = $1
			  and ($2 = '' or created_at >= $2::date)
			  and ($3 = '' or created_at < ($3::date + interval '1 day'))
			group by path
			having path <> ''
			order by pageviews desc, clicks desc, max(created_at) desc
			limit $4
		`, filters.SiteID, filters.From, filters.To, filters.Limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PageStat, 0, filters.Limit)
	for rows.Next() {
		var item model.PageStat
		if err := rows.Scan(&item.Path, &item.Pageviews, &item.Clicks, &item.FormSubmissions, &item.UniqueVisitors, &item.LastSeen); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FetchOverview(ctx context.Context, filters model.Filters) (model.OverviewMetrics, error) {
	var overview model.OverviewMetrics

	if s.isSQLite {
		if err := s.db.QueryRowContext(ctx, `
			select
				coalesce(sum(case when event_type = 'pageview' then 1 else 0 end), 0) as pageviews,
				coalesce(sum(case when event_type = 'click' then 1 else 0 end), 0) as clicks,
				coalesce(count(distinct case when session_id is not null then session_id end), 0) as unique_visitors
			from events
			where site_id = ?
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
		`, filters.SiteID, filters.From, filters.From, filters.To, filters.To).Scan(&overview.Pageviews, &overview.Clicks, &overview.UniqueVisitors); err != nil {
			return model.OverviewMetrics{}, err
		}

		if err := s.db.QueryRowContext(ctx, `
			select coalesce(path, '')
			from events
			where site_id = ? and event_type = 'pageview'
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			group by path
			order by count(*) desc, path asc
			limit 1
		`, filters.SiteID, filters.From, filters.From, filters.To, filters.To).Scan(&overview.TopPage); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return model.OverviewMetrics{}, err
		}
		return overview, nil
	}

	if err := s.db.QueryRowContext(ctx, `
		select
			coalesce(sum(case when event_type = 'pageview' then 1 else 0 end), 0) as pageviews,
			coalesce(sum(case when event_type = 'click' then 1 else 0 end), 0) as clicks,
			coalesce(count(distinct session_id) filter (where session_id is not null), 0) as unique_visitors
		from events
		where site_id = $1
		  and ($2 = '' or created_at >= $2::date)
		  and ($3 = '' or created_at < ($3::date + interval '1 day'))
	`, filters.SiteID, filters.From, filters.To).Scan(&overview.Pageviews, &overview.Clicks, &overview.UniqueVisitors); err != nil {
		return model.OverviewMetrics{}, err
	}

	if err := s.db.QueryRowContext(ctx, `
		select coalesce(path, '')
		from events
		where site_id = $1 and event_type = 'pageview'
		  and ($2 = '' or created_at >= $2::date)
		  and ($3 = '' or created_at < ($3::date + interval '1 day'))
		group by path
		order by count(*) desc, path asc
		limit 1
	`, filters.SiteID, filters.From, filters.To).Scan(&overview.TopPage); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.OverviewMetrics{}, err
	}

	return overview, nil
}

func (s *Store) FetchTimeline(ctx context.Context, filters model.Filters) ([]model.TimelinePoint, error) {
	interval := normalizeInterval(filters.Interval)
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		format := "%Y-%m-%d"
		if interval == "hour" {
			format = "%Y-%m-%d %H:00"
		}
		rows, err = s.db.QueryContext(ctx, `
			select strftime(?, created_at) as label, count(*) as count
			from events
			where site_id = ?
			  and event_type = 'pageview'
			  and (? = '' or path = ?)
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			group by 1
			order by 1 asc
		`, format, filters.SiteID, filters.Path, filters.Path, filters.From, filters.From, filters.To, filters.To)
	} else {
		groupExpr := "YYYY-MM-DD"
		if interval == "hour" {
			groupExpr = "YYYY-MM-DD HH24:00"
		}
		rows, err = s.db.QueryContext(ctx, `
			select to_char(date_trunc($4, created_at), $5) as label, count(*) as count
			from events
			where site_id = $1
			  and event_type = 'pageview'
			  and ($2 = '' or path = $2)
			  and ($3 = '' or created_at >= $3::date)
			  and ($6 = '' or created_at < ($6::date + interval '1 day'))
			group by 1
			order by 1 asc
		`, filters.SiteID, filters.Path, filters.From, interval, groupExpr, filters.To)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]model.TimelinePoint, 0, 32)
	for rows.Next() {
		var point model.TimelinePoint
		if err := rows.Scan(&point.Label, &point.Count); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (s *Store) FetchRecentVisits(ctx context.Context, filters model.Filters) ([]model.VisitEntry, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select
				created_at,
				path,
				coalesce(title, ''),
				coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as source,
				coalesce(session_id, '')
			from events
			where site_id = ?
			  and event_type = 'pageview'
			  and (? = '' or path = ?)
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			order by created_at desc
			limit ?
		`, filters.SiteID, filters.Path, filters.Path, filters.From, filters.From, filters.To, filters.To, filters.Limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			select
				to_char(created_at, 'YYYY-MM-DD HH24:MI:SS'),
				path,
				coalesce(title, ''),
				coalesce(nullif(utm_source, ''), nullif(ref_domain, ''), nullif(source, ''), 'direct') as source,
				coalesce(session_id, '')
			from events
			where site_id = $1
			  and event_type = 'pageview'
			  and ($2 = '' or path = $2)
			  and ($3 = '' or created_at >= $3::date)
			  and ($4 = '' or created_at < ($4::date + interval '1 day'))
			order by created_at desc
			limit $5
		`, filters.SiteID, filters.Path, filters.From, filters.To, filters.Limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visits := make([]model.VisitEntry, 0, filters.Limit)
	for rows.Next() {
		var visit model.VisitEntry
		if err := rows.Scan(&visit.CreatedAt, &visit.Path, &visit.Title, &visit.Source, &visit.SessionID); err != nil {
			return nil, err
		}
		visits = append(visits, visit)
	}
	return visits, rows.Err()
}

func (s *Store) FetchPageAnalytics(ctx context.Context, filters model.Filters) (model.PageAnalytics, error) {
	analytics := model.PageAnalytics{Path: filters.Path}
	if filters.Path == "" {
		return analytics, fmt.Errorf("path is required")
	}

	var row *sql.Row
	if s.isSQLite {
		row = s.db.QueryRowContext(ctx, `
			select
				sum(case when event_type = 'pageview' then 1 else 0 end) as pageviews,
				sum(case when event_type = 'click' then 1 else 0 end) as clicks,
				sum(case when event_type = 'form_submit' then 1 else 0 end) as form_submissions,
				count(distinct case when session_id is not null then session_id end) as unique_visitors,
				max(created_at) as last_seen
			from events
			where site_id = ? and path = ?
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
		`, filters.SiteID, filters.Path, filters.From, filters.From, filters.To, filters.To)
	} else {
		row = s.db.QueryRowContext(ctx, `
			select
				sum(case when event_type = 'pageview' then 1 else 0 end) as pageviews,
				sum(case when event_type = 'click' then 1 else 0 end) as clicks,
				sum(case when event_type = 'form_submit' then 1 else 0 end) as form_submissions,
				count(distinct session_id) filter (where session_id is not null) as unique_visitors,
				to_char(max(created_at), 'YYYY-MM-DD HH24:MI:SS') as last_seen
			from events
			where site_id = $1 and path = $2
			  and ($3 = '' or created_at >= $3::date)
			  and ($4 = '' or created_at < ($4::date + interval '1 day'))
		`, filters.SiteID, filters.Path, filters.From, filters.To)
	}

	if err := row.Scan(&analytics.Pageviews, &analytics.Clicks, &analytics.FormSubmissions, &analytics.UniqueVisitors, &analytics.LastInteractionAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.PageAnalytics{}, err
	}

	storedEvents, err := s.fetchClickAndScrollEvents(ctx, filters)
	if err != nil {
		return model.PageAnalytics{}, err
	}

	targets := map[string]*model.ClickTarget{}
	scrollDepths := map[int]int{}
	var scrollSum int
	var scrollCount int

	for _, event := range storedEvents {
		var meta clickMeta
		if strings.TrimSpace(event.Meta) != "" {
			_ = json.Unmarshal([]byte(event.Meta), &meta)
		}

		if meta.DepthPct > 0 {
			scrollDepths[meta.DepthPct]++
			scrollSum += meta.DepthPct
			scrollCount++
			continue
		}

		selector := strings.TrimSpace(meta.Selector)
		if selector == "" {
			selector = fallbackSelector(meta)
		}
		if selector == "" {
			selector = "unknown"
		}

		target, ok := targets[selector]
		if !ok {
			target = &model.ClickTarget{
				Selector: selector,
				Text:     strings.TrimSpace(meta.Text),
				Href:     strings.TrimSpace(meta.Href),
				Tag:      strings.TrimSpace(meta.Tag),
			}
			targets[selector] = target
		}
		target.Count++
	}

	if scrollCount > 0 {
		analytics.AvgScrollDepth = float64(scrollSum) / float64(scrollCount)
	}

	analytics.TopTargets = make([]model.ClickTarget, 0, len(targets))
	for _, target := range targets {
		if analytics.Clicks > 0 {
			target.Share = float64(target.Count) / float64(analytics.Clicks)
		}
		analytics.TopTargets = append(analytics.TopTargets, *target)
	}
	sort.Slice(analytics.TopTargets, func(i, j int) bool {
		if analytics.TopTargets[i].Count == analytics.TopTargets[j].Count {
			return analytics.TopTargets[i].Selector < analytics.TopTargets[j].Selector
		}
		return analytics.TopTargets[i].Count > analytics.TopTargets[j].Count
	})
	if len(analytics.TopTargets) > 8 {
		analytics.TopTargets = analytics.TopTargets[:8]
	}

	depthKeys := make([]int, 0, len(scrollDepths))
	for depth := range scrollDepths {
		depthKeys = append(depthKeys, depth)
	}
	sort.Ints(depthKeys)
	analytics.ScrollDepths = make([]model.ScrollDepthPoint, 0, len(depthKeys))
	for _, depth := range depthKeys {
		analytics.ScrollDepths = append(analytics.ScrollDepths, model.ScrollDepthPoint{Depth: depth, Count: scrollDepths[depth]})
	}

	return analytics, nil
}

func (s *Store) fetchClickAndScrollEvents(ctx context.Context, filters model.Filters) ([]model.StoredEvent, error) {
	var rows *sql.Rows
	var err error

	if s.isSQLite {
		rows, err = s.db.QueryContext(ctx, `
			select created_at, coalesce(meta, '')
			from events
			where site_id = ? and path = ?
			  and event_type in ('click', 'scroll')
			  and (? = '' or created_at >= datetime(?))
			  and (? = '' or created_at < datetime(?, '+1 day'))
			order by created_at desc
			limit ?
		`, filters.SiteID, filters.Path, filters.From, filters.From, filters.To, filters.To, max(filters.Limit, 5000))
	} else {
		rows, err = s.db.QueryContext(ctx, `
			select created_at, coalesce(meta, '')
			from events
			where site_id = $1 and path = $2
			  and event_type in ('click', 'scroll')
			  and ($3 = '' or created_at >= $3::date)
			  and ($4 = '' or created_at < ($4::date + interval '1 day'))
			order by created_at desc
			limit $5
		`, filters.SiteID, filters.Path, filters.From, filters.To, max(filters.Limit, 5000))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.StoredEvent, 0, max(filters.Limit, 5000))
	for rows.Next() {
		var item model.StoredEvent
		if err := rows.Scan(&item.CreatedAt, &item.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) EnsureEventPartitions(ctx context.Context, monthsAhead int) error {
	if s.isSQLite {
		return nil
	}
	if monthsAhead < 1 {
		monthsAhead = 1
	}

	now := time.Now().UTC()
	for i := 0; i < monthsAhead; i++ {
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, i, 0)
		start := monthStart.Format("2006-01-02")
		end := monthStart.AddDate(0, 1, 0).Format("2006-01-02")
		partitionName := fmt.Sprintf("events_%04d_%02d", monthStart.Year(), int(monthStart.Month()))

		query := fmt.Sprintf(`
			create table if not exists %s
			partition of events
			for values from ('%s') to ('%s')
		`, partitionName, start, end)
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RebuildDailyAggregates(ctx context.Context, days int) error {
	if days < 1 {
		days = 1
	}
	utcNow := time.Now().UTC()
	for offset := 0; offset < days; offset++ {
		day := utcNow.AddDate(0, 0, -offset).Format("2006-01-02")
		if s.isSQLite {
			if _, err := s.db.ExecContext(ctx, `delete from event_daily where day = ?`, day); err != nil {
				return err
			}
			if _, err := s.db.ExecContext(ctx, `
				insert into event_daily (day, site_id, event_type, path, count)
				select date(created_at), site_id, event_type, coalesce(path, ''), count(*)
				from events
				where created_at >= datetime(?) and created_at < datetime(?, '+1 day')
				group by 1,2,3,4
			`, day, day); err != nil {
				return err
			}
			continue
		}

		if _, err := s.db.ExecContext(ctx, `delete from event_daily where day = $1`, day); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `
			insert into event_daily (day, site_id, event_type, path, count)
			select date_trunc('day', created_at)::date, site_id, event_type, coalesce(path, ''), count(*)
			from events
			where created_at >= $1::date and created_at < ($1::date + interval '1 day')
			group by 1,2,3,4
		`, day); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ApplyRetention(ctx context.Context) error {
	if s.isSQLite {
		if _, err := s.db.ExecContext(ctx, `delete from events where created_at < datetime('now', ? || ' days')`, fmt.Sprintf("-%d", s.cfg.RawRetentionDays)); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `delete from event_daily where day < date('now', ? || ' months')`, fmt.Sprintf("-%d", s.cfg.AggRetentionMonths)); err != nil {
			return err
		}
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `delete from events where created_at < now() - ($1 * interval '1 day')`, s.cfg.RawRetentionDays); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `delete from event_daily where day < (current_date - ($1 * interval '1 month'))`, s.cfg.AggRetentionMonths); err != nil {
		return err
	}
	return nil
}

func (s *Store) applySchema(ctx context.Context) error {
	schemaName := "schema.pg.sql"
	if s.isSQLite {
		schemaName = "schema.sqlite.sql"
	}

	content, err := os.ReadFile(resolveSchemaPath(schemaName))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, string(content))
	return err
}

func resolveSchemaPath(name string) string {
	candidates := []string{
		filepath.Join("db", name),
		filepath.Join("..", "..", "db", name),
		filepath.Join("..", "..", "..", "db", name),
		filepath.Join("backend", "db", name),
		filepath.Join("/app", "db", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("db", name)
}

func normalizeRefDomain(raw string) string {
	reference := strings.TrimSpace(raw)
	if reference == "" {
		return ""
	}
	parsed, err := url.Parse(reference)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Host)
}

func normalizeUTM(event model.CollectEvent) (string, string, string) {
	return strings.ToLower(strings.TrimSpace(event.UtmSource)), strings.ToLower(strings.TrimSpace(event.UtmMedium)), strings.ToLower(strings.TrimSpace(event.UtmCampaign))
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func fallbackSelector(meta clickMeta) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(meta.Tag) != "" {
		parts = append(parts, strings.ToLower(meta.Tag))
	}
	if strings.TrimSpace(meta.ID) != "" {
		parts = append(parts, "#"+meta.ID)
	}
	if strings.TrimSpace(meta.Class) != "" {
		classNames := strings.Fields(meta.Class)
		if len(classNames) > 0 {
			parts = append(parts, "."+strings.Join(classNames, "."))
		}
	}
	return strings.Join(parts, "")
}

func ParseInt64(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseInt(value, 10, 64)
}

func DecodeEvents(body []byte) ([]model.CollectEvent, error) {
	var batch []model.CollectEvent
	if err := json.Unmarshal(body, &batch); err == nil {
		return batch, nil
	}

	var single model.CollectEvent
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []model.CollectEvent{single}, nil
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizeInterval(value string) string {
	if strings.TrimSpace(value) == "hour" {
		return "hour"
	}
	return "day"
}
