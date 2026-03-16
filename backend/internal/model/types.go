package model

type Site struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type CollectEvent struct {
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

type RealtimePoint struct {
	Minute string `json:"minute"`
	Count  int    `json:"count"`
}

type RealtimeResponse struct {
	ActiveUsers int             `json:"active_users"`
	Series      []RealtimePoint `json:"series"`
}

type HeatmapPoint struct {
	XBucket int `json:"x_pct"`
	YBucket int `json:"y_pct"`
	Count   int `json:"count"`
}

type TrafficSource struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type EventItem struct {
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

type PageStat struct {
	Path            string `json:"path"`
	Pageviews       int    `json:"pageviews"`
	Clicks          int    `json:"clicks"`
	FormSubmissions int    `json:"form_submissions"`
	UniqueVisitors  int    `json:"unique_visitors"`
	LastSeen        string `json:"last_seen"`
}

type ClickTarget struct {
	Selector string  `json:"selector"`
	Text     string  `json:"text"`
	Href     string  `json:"href"`
	Tag      string  `json:"tag"`
	Count    int     `json:"count"`
	Share    float64 `json:"share"`
}

type ScrollDepthPoint struct {
	Depth int `json:"depth"`
	Count int `json:"count"`
}

type PageAnalytics struct {
	Path              string             `json:"path"`
	Pageviews         int                `json:"pageviews"`
	Clicks            int                `json:"clicks"`
	FormSubmissions   int                `json:"form_submissions"`
	UniqueVisitors    int                `json:"unique_visitors"`
	AvgScrollDepth    float64            `json:"avg_scroll_depth"`
	TopTargets        []ClickTarget      `json:"top_targets"`
	ScrollDepths      []ScrollDepthPoint `json:"scroll_depths"`
	LastInteractionAt string             `json:"last_interaction_at"`
}

type OverviewMetrics struct {
	Pageviews      int    `json:"pageviews"`
	Clicks         int    `json:"clicks"`
	UniqueVisitors int    `json:"unique_visitors"`
	TopPage        string `json:"top_page"`
}

type TimelinePoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type VisitEntry struct {
	CreatedAt string `json:"created_at"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Source    string `json:"source"`
	SessionID string `json:"session_id"`
}

type AuthUser struct {
	Email string `json:"email"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Filters struct {
	SiteID   int64
	Path     string
	From     string
	To       string
	Limit    int
	Bucket   int
	Interval string
}

type StoredEvent struct {
	CreatedAt string
	Meta      string
}
