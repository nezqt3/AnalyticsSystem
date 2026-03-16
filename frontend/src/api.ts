import { API_BASE_URL } from "./env";
import type {
  AuthUser,
  EventItem,
  HeatmapPoint,
  OverviewMetrics,
  PageAnalytics,
  PageStat,
  RealtimeResponse,
  TimelinePoint,
  TrafficSource,
  VisitEntry,
} from "./types/api";

export const API_BASE = API_BASE_URL;

type QueryValue = string | number | undefined;

type RequestOptions = {
  method?: string;
  query?: Record<string, QueryValue>;
  body?: unknown;
};

function buildURL(
  path: string,
  query: Record<string, QueryValue> = {},
): string {
  const search = new URLSearchParams();

  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === "") return;
    search.set(key, String(value));
  });

  const suffix = search.toString();
  return `${API_BASE}${path}${suffix ? `?${suffix}` : ""}`;
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const response = await fetch(buildURL(path, options.query), {
    method: options.method ?? "GET",
    headers: options.body ? { "Content-Type": "application/json" } : undefined,
    body: options.body ? JSON.stringify(options.body) : undefined,
    credentials: "include",
  });

  if (!response.ok) {
    const details = (await response.text()).trim();
    throw new Error(details || `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export function getAuthMe(): Promise<AuthUser> {
  return request<AuthUser>("/api/auth/me");
}

export function login(email: string, password: string): Promise<AuthUser> {
  return request<AuthUser>("/api/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function logout(): Promise<void> {
  return request<void>("/api/auth/logout", { method: "POST" });
}

export function getRealtime(siteId: number): Promise<RealtimeResponse> {
  return request<RealtimeResponse>("/api/realtime", {
    query: { site_id: siteId },
  });
}

export function getPages(
  siteId: number,
  from: string,
  to: string,
  limit = 200,
): Promise<PageStat[]> {
  return request<PageStat[]>("/api/pages", {
    query: { site_id: siteId, from, to, limit },
  });
}

export function getOverview(
  siteId: number,
  from: string,
  to: string,
): Promise<OverviewMetrics> {
  return request<OverviewMetrics>("/api/overview", {
    query: { site_id: siteId, from, to },
  });
}

export function getTimeline(
  siteId: number,
  path: string,
  from: string,
  to: string,
  interval: string,
): Promise<TimelinePoint[]> {
  return request<TimelinePoint[]>("/api/timeline", {
    query: { site_id: siteId, path, from, to, interval },
  });
}

export function getRecentVisits(
  siteId: number,
  path: string,
  from: string,
  to: string,
  limit: number,
): Promise<VisitEntry[]> {
  return request<VisitEntry[]>("/api/visits", {
    query: { site_id: siteId, path, from, to, limit },
  });
}

export function getTrafficSources(
  siteId: number,
  path: string,
  from: string,
  to: string,
): Promise<TrafficSource[]> {
  return request<TrafficSource[]>("/api/traffic-sources", {
    query: { site_id: siteId, path, from, to },
  });
}

export function getHeatmap(
  siteId: number,
  path: string,
  from: string,
  to: string,
  bucket: number,
): Promise<HeatmapPoint[]> {
  return request<HeatmapPoint[]>("/api/heatmap", {
    query: { site_id: siteId, path, from, to, bucket },
  });
}

export function getEvents(
  siteId: number,
  path: string,
  from: string,
  to: string,
  limit: number,
): Promise<EventItem[]> {
  return request<EventItem[]>("/api/events", {
    query: { site_id: siteId, path, from, to, limit },
  });
}

export function getPageAnalytics(
  siteId: number,
  path: string,
  from: string,
  to: string,
): Promise<PageAnalytics> {
  return request<PageAnalytics>("/api/page-analytics", {
    query: { site_id: siteId, path, from, to },
  });
}
