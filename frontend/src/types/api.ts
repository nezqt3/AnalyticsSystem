export type RealtimePoint = {
  minute: string;
  count: number;
};

export type RealtimeResponse = {
  active_users: number;
  series: RealtimePoint[];
};

export type TrafficSource = {
  source: string;
  count: number;
};

export type HeatmapPoint = {
  x_pct: number;
  y_pct: number;
  count: number;
};

export type EventItem = {
  created_at: string;
  event_type: string;
  path: string;
  title: string;
  meta: string;
  ref_domain: string;
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
};

export type PageStat = {
  path: string;
  pageviews: number;
  clicks: number;
  form_submissions: number;
  unique_visitors: number;
  last_seen: string;
};

export type ClickTarget = {
  selector: string;
  text: string;
  href: string;
  tag: string;
  count: number;
  share: number;
};

export type ScrollDepthPoint = {
  depth: number;
  count: number;
};

export type PageAnalytics = {
  path: string;
  pageviews: number;
  clicks: number;
  form_submissions: number;
  unique_visitors: number;
  avg_scroll_depth: number;
  top_targets: ClickTarget[];
  scroll_depths: ScrollDepthPoint[];
  last_interaction_at: string;
};
