import { useEffect, useMemo, useState } from "react";

import {
  getEvents,
  getHeatmap,
  getPageAnalytics,
  getPages,
  getTrafficSources,
} from "../api";
import type {
  EventItem,
  HeatmapPoint,
  PageAnalytics,
  PageStat,
  TrafficSource,
} from "../types/api";

type UseDashboardDataArgs = {
  siteId: number;
  selectedPath: string;
  from: string;
  to: string;
  bucket: number;
};

type DashboardState = {
  pages: PageStat[];
  trafficSources: TrafficSource[];
  heatmap: HeatmapPoint[];
  events: EventItem[];
  pageAnalytics: PageAnalytics | null;
  loading: boolean;
  error: string;
};

const EMPTY_STATE: DashboardState = {
  pages: [],
  trafficSources: [],
  heatmap: [],
  events: [],
  pageAnalytics: null,
  loading: true,
  error: "",
};

export function useDashboardData({
  siteId,
  selectedPath,
  from,
  to,
  bucket,
}: UseDashboardDataArgs): DashboardState {
  const [state, setState] = useState<DashboardState>(EMPTY_STATE);

  const shouldLoadPageDetails = useMemo(
    () => selectedPath.trim().length > 0,
    [selectedPath],
  );

  useEffect(() => {
    let active = true;

    async function load() {
      setState((current) => ({ ...current, loading: true, error: "" }));

      try {
        const pages = await getPages(siteId, from, to);
        const nextPath = shouldLoadPageDetails
          ? selectedPath
          : (pages[0]?.path ?? "");

        const [trafficSources, heatmap, events, pageAnalytics] = nextPath
          ? await Promise.all([
              getTrafficSources(siteId, nextPath, from, to),
              getHeatmap(siteId, nextPath, from, to, bucket),
              getEvents(siteId, nextPath, from, to, 150),
              getPageAnalytics(siteId, nextPath, from, to),
            ])
          : [[], [], [], null];

        if (!active) return;

        setState({
          pages,
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
  }, [siteId, selectedPath, from, to, bucket, shouldLoadPageDetails]);

  return state;
}
