import { useEffect, useMemo, useState } from "react";

import {
  getEvents,
  getHeatmap,
  getOverview,
  getPageAnalytics,
  getPages,
  getRecentVisits,
  getTimeline,
  getTrafficSources,
} from "../api";
import type {
  EventItem,
  HeatmapPoint,
  OverviewMetrics,
  PageAnalytics,
  PageStat,
  TimelinePoint,
  TrafficSource,
  VisitEntry,
} from "../types/api";

type UseDashboardDataArgs = {
  siteId: number;
  selectedPath: string;
  from: string;
  to: string;
  bucket: number;
  enabled: boolean;
};

type DashboardState = {
  pages: PageStat[];
  overview: OverviewMetrics | null;
  timeline: TimelinePoint[];
  visits: VisitEntry[];
  trafficSources: TrafficSource[];
  heatmap: HeatmapPoint[];
  events: EventItem[];
  pageAnalytics: PageAnalytics | null;
  loading: boolean;
  error: string;
};

const EMPTY_STATE: DashboardState = {
  pages: [],
  overview: null,
  timeline: [],
  visits: [],
  trafficSources: [],
  heatmap: [],
  events: [],
  pageAnalytics: null,
  loading: false,
  error: "",
};

function resolveInterval(from: string, to: string): string {
  const fromDate = new Date(from);
  const toDate = new Date(to);
  const days = Math.ceil(
    (toDate.getTime() - fromDate.getTime()) / (24 * 60 * 60 * 1000),
  );
  return days <= 2 ? "hour" : "day";
}

export function useDashboardData({
  siteId,
  selectedPath,
  from,
  to,
  bucket,
  enabled,
}: UseDashboardDataArgs): DashboardState {
  const [state, setState] = useState<DashboardState>(EMPTY_STATE);
  const interval = useMemo(() => resolveInterval(from, to), [from, to]);

  useEffect(() => {
    if (!enabled) {
      setState(EMPTY_STATE);
      return;
    }

    let active = true;

    async function load() {
      setState((current) => ({ ...current, loading: true, error: "" }));

      try {
        const pages = await getPages(siteId, from, to);
        const nextPath = selectedPath || pages[0]?.path || "";

        const [
          overview,
          timeline,
          visits,
          trafficSources,
          heatmap,
          events,
          pageAnalytics,
        ] = await Promise.all([
          getOverview(siteId, from, to),
          getTimeline(siteId, nextPath, from, to, interval),
          getRecentVisits(siteId, nextPath, from, to, 25),
          nextPath
            ? getTrafficSources(siteId, nextPath, from, to)
            : Promise.resolve([]),
          nextPath
            ? getHeatmap(siteId, nextPath, from, to, bucket)
            : Promise.resolve([]),
          nextPath
            ? getEvents(siteId, nextPath, from, to, 150)
            : Promise.resolve([]),
          nextPath
            ? getPageAnalytics(siteId, nextPath, from, to)
            : Promise.resolve(null),
        ]);

        if (!active) return;

        setState({
          pages,
          overview,
          timeline,
          visits,
          trafficSources,
          heatmap,
          events,
          pageAnalytics,
          loading: false,
          error: "",
        });
      } catch (error) {
        if (!active) return;
        setState((current) => ({
          ...current,
          loading: false,
          error:
            error instanceof Error
              ? error.message
              : "Не удалось загрузить данные",
        }));
      }
    }

    void load();
    const timer = window.setInterval(load, 10000);

    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [siteId, selectedPath, from, to, bucket, enabled, interval]);

  return state;
}
