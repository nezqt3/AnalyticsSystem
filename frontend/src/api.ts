export const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080";

export type RealtimeResp = {
  active_users: number;
  series: { minute: string; count: number }[];
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

export async function getRealtime(siteId: number): Promise<RealtimeResp> {
  const res = await fetch(`${API_BASE}/api/realtime?site_id=${siteId}`);
  if (!res.ok) throw new Error("realtime fetch failed");
  return res.json();
}

export async function getTrafficSources(siteId: number, path: string, from: string, to: string): Promise<TrafficSource[]> {
  const qs = new URLSearchParams({
    site_id: String(siteId),
    path,
    from,
    to
  });
  const res = await fetch(`${API_BASE}/api/traffic-sources?${qs.toString()}`);
  if (!res.ok) throw new Error("traffic fetch failed");
  return res.json();
}

export async function getHeatmap(siteId: number, path: string, bucket: number): Promise<HeatmapPoint[]> {
  const qs = new URLSearchParams({
    site_id: String(siteId),
    path,
    bucket: String(bucket)
  });
  const res = await fetch(`${API_BASE}/api/heatmap?${qs.toString()}`);
  if (!res.ok) throw new Error("heatmap fetch failed");
  return res.json();
}

export async function getEvents(siteId: number, path: string, from: string, to: string, limit: number): Promise<EventItem[]> {
  const qs = new URLSearchParams({
    site_id: String(siteId),
    path,
    from,
    to,
    limit: String(limit)
  });
  const res = await fetch(`${API_BASE}/api/events?${qs.toString()}`);
  if (!res.ok) throw new Error("events fetch failed");
  return res.json();
}
