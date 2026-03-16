import { useEffect, useMemo, useState } from "react";

import { EventsTable } from "./components/EventsTable";
import { Filters } from "./components/Filters";
import { Header } from "./components/Header";
import { HeatmapPanel } from "./components/HeatmapPanel";
import { PageList } from "./components/PageList";
import { PageOverview } from "./components/PageOverview";
import { RealtimePanel } from "./components/RealtimePanel";
import { TrafficSourcesPanel } from "./components/TrafficSourcesPanel";
import { useDashboardData } from "./hooks/useDashboardData";
import { useRealtime } from "./hooks/useRealtime";
import { getDefaultDateRange } from "./lib/date";
import { DEFAULT_SITE_ID } from "./env";

const DEFAULT_BUCKET = 5;

export default function App() {
  const defaultRange = useMemo(() => getDefaultDateRange(), []);
  const [siteId] = useState(DEFAULT_SITE_ID);
  const [selectedPath, setSelectedPath] = useState("");
  const [from, setFrom] = useState(defaultRange.from);
  const [to, setTo] = useState(defaultRange.to);
  const [bucket, setBucket] = useState(DEFAULT_BUCKET);

  const { realtime, error: realtimeError } = useRealtime(siteId);
  const {
    pages,
    trafficSources,
    heatmap,
    events,
    pageAnalytics,
    loading,
    error,
  } = useDashboardData({
    siteId,
    selectedPath,
    from,
    to,
    bucket,
  });

  useEffect(() => {
    if (!selectedPath && pages[0]?.path) {
      setSelectedPath(pages[0].path);
      return;
    }

    if (
      selectedPath &&
      pages.length > 0 &&
      !pages.some((page) => page.path === selectedPath)
    ) {
      setSelectedPath(pages[0].path);
    }
  }, [pages, selectedPath]);

  const activePath = selectedPath || pages[0]?.path || "";

  return (
    <div className="app-shell">
      <div className="background-orb background-orb-left" />
      <div className="background-orb background-orb-right" />
      <main className="page">
        <Header siteId={siteId} realtime={realtime} />
        <Filters
          from={from}
          to={to}
          bucket={bucket}
          onFromChange={setFrom}
          onToChange={setTo}
          onBucketChange={setBucket}
        />

        {(error || realtimeError) && (
          <div className="alert">{error || realtimeError}</div>
        )}

        <div className="layout-grid">
          <PageList
            pages={pages}
            selectedPath={activePath}
            onSelect={setSelectedPath}
          />
          <div className="content-grid">
            <PageOverview analytics={pageAnalytics} />
            <RealtimePanel realtime={realtime} />
            <TrafficSourcesPanel items={trafficSources} />
            <HeatmapPanel points={heatmap} />
          </div>
        </div>

        <EventsTable events={events} />
        {loading && <div className="loading-indicator">Обновляем данные…</div>}
      </main>
    </div>
  );
}
