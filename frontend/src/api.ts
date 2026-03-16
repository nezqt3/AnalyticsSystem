import type {
  EventItem,
  HeatmapPoint,
  PageAnalytics,
  PageStat,
  RealtimeResponse,
  TrafficSource,
} from "./types/api";
import { API_BASE_URL } from "./env";

export const API_BASE = API_BASE_URL;

type QueryValue = string | number | undefined;

function buildURL(path: string, query: Record<string, QueryValue>): string {
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
  query: Record<string, QueryValue>,
): Promise<T> {
  const response = await fetch(buildURL(path, query));
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

export function getRealtime(siteId: number): Promise<RealtimeResponse> {
  return request<RealtimeResponse>("/api/realtime", { site_id: siteId });
}

export function getPages(
  siteId: number,
  from: string,
  to: string,
  limit = 200,
): Promise<PageStat[]> {
  return request<PageStat[]>("/api/pages", {
    site_id: siteId,
    from,
    to,
    limit,
  });
}

export function getTrafficSources(
  siteId: number,
  path: string,
  from: string,
  to: string,
): Promise<TrafficSource[]> {
  return request<TrafficSource[]>("/api/traffic-sources", {
    site_id: siteId,
    path,
    from,
    to,
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
    site_id: siteId,
    path,
    from,
    to,
    bucket,
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
    site_id: siteId,
    path,
    from,
    to,
    limit,
  });
}

export function getPageAnalytics(
  siteId: number,
  path: string,
  from: string,
  to: string,
): Promise<PageAnalytics> {
  return request<PageAnalytics>("/api/page-analytics", {
    site_id: siteId,
    path,
    from,
    to,
  });
}
