import { useEffect, useState } from "react";
import { API_BASE, getEvents, getHeatmap, getRealtime, getTrafficSources, EventItem, HeatmapPoint, RealtimeResp, TrafficSource } from "./api";

const DEFAULT_SITE_ID = Number(import.meta.env.VITE_SITE_ID || 1);
const DEFAULT_PATH = "/";
const today = new Date();
const toISO = (d: Date) => d.toISOString().slice(0, 10);
const DEFAULT_TO = toISO(today);
const DEFAULT_FROM = toISO(new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000));
const DEFAULT_BUCKET = 5;

export default function App() {
  const [siteId] = useState(DEFAULT_SITE_ID);
  const [path, setPath] = useState(DEFAULT_PATH);
  const [from, setFrom] = useState(DEFAULT_FROM);
  const [to, setTo] = useState(DEFAULT_TO);
  const [bucket, setBucket] = useState(DEFAULT_BUCKET);
  const [realtime, setRealtime] = useState<RealtimeResp | null>(null);
  const [traffic, setTraffic] = useState<TrafficSource[]>([]);
  const [heatmap, setHeatmap] = useState<HeatmapPoint[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [error, setError] = useState<string>("");
  const safeTraffic = Array.isArray(traffic) ? traffic : [];
  const safeHeatmap = Array.isArray(heatmap) ? heatmap : [];
  const safeEvents = Array.isArray(events) ? events : [];
  const maxHeat = safeHeatmap.reduce((m, v) => (v.count > m ? v.count : m), 1);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const [ts, hm, ev] = await Promise.all([
          getTrafficSources(siteId, path, from, to),
          getHeatmap(siteId, path, bucket),
          getEvents(siteId, path, from, to, 200)
        ]);
        if (!alive) return;
        setTraffic(Array.isArray(ts) ? ts : []);
        setHeatmap(Array.isArray(hm) ? hm : []);
        setEvents(Array.isArray(ev) ? ev : []);
      } catch (e) {
        if (!alive) return;
        setError((e as Error).message);
      }
    };

    load();
    const t = setInterval(load, 10000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, [siteId, path, from, to, bucket]);

  useEffect(() => {
    let alive = true;
    let pollTimer: number | null = null;
    let ws: WebSocket | null = null;

    const startPoll = () => {
      const poll = async () => {
        try {
          const rt = await getRealtime(siteId);
          if (alive) setRealtime(rt);
        } catch (e) {
          if (alive) setError((e as Error).message);
        }
      };
      poll();
      pollTimer = window.setInterval(poll, 5000);
    };

    try {
      const wsBase = API_BASE.replace(/^http/, "ws");
      ws = new WebSocket(`${wsBase}/ws/realtime?site_id=${siteId}`);
      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data) as RealtimeResp;
          if (alive) setRealtime(data);
        } catch {
          // ignore bad frames
        }
      };
      ws.onclose = () => {
        if (alive && pollTimer === null) startPoll();
      };
      ws.onerror = () => {
        if (alive && pollTimer === null) startPoll();
      };
    } catch {
      startPoll();
    }

    return () => {
      alive = false;
      if (pollTimer !== null) window.clearInterval(pollTimer);
      if (ws) ws.close();
    };
  }, [siteId]);

  return (
    <div className="page">
      <header className="header">
        <h1>Analytics</h1>
        <div className="meta">Site ID: {siteId}</div>
      </header>

      {error && <div className="error">{error}</div>}

      <section className="card">
        <h2>Фильтры</h2>
        <div className="filters">
          <label>
            Path
            <input value={path} onChange={(e) => setPath(e.target.value)} placeholder="/" />
          </label>
          <label>
            From
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
          </label>
          <label>
            To
            <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
          </label>
          <label>
            Heatmap bucket (%)
            <input
              type="number"
              min={1}
              max={25}
              value={bucket}
              onChange={(e) => setBucket(Number(e.target.value) || DEFAULT_BUCKET)}
            />
          </label>
        </div>
      </section>

      <section className="card">
        <h2>Realtime</h2>
        <div className="realtime-grid">
          <div className="metric">
            <div className="label">Active users (5 min)</div>
            <div className="value">{realtime ? realtime.active_users : "—"}</div>
          </div>
          <div className="chart">
            <div className="label">Events per minute</div>
            <ul className="series">
              {(realtime?.series || []).map((p) => (
                <li key={p.minute}>
                  <span>{p.minute}</span>
                  <span className="bar" style={{ width: Math.min(p.count * 10, 200) }} />
                  <span>{p.count}</span>
                </li>
              ))}
              {(realtime?.series || []).length === 0 && <li>Нет данных</li>}
            </ul>
          </div>
        </div>
      </section>

      <section className="card">
        <h2>Источники трафика</h2>
        <ul className="list">
          {safeTraffic.map((t) => (
            <li key={t.source}>
              <span>{t.source}</span>
              <span>{t.count}</span>
            </li>
          ))}
          {safeTraffic.length === 0 && <li>Нет данных</li>}
        </ul>
      </section>

      <section className="card">
        <h2>Heatmap (топ кликов)</h2>
        <div className="heatmap">
          <div className="heatmap-canvas">
            {safeHeatmap.length === 0 && <div className="heatmap-empty">Нет данных</div>}
            {safeHeatmap.map((p, i) => {
              const size = 8 + Math.round((p.count / maxHeat) * 24);
              const opacity = 0.25 + (p.count / maxHeat) * 0.65;
              return (
                <div
                  key={i}
                  className="heat-dot"
                  style={{
                    left: `${p.x_pct}%`,
                    top: `${p.y_pct}%`,
                    width: `${size}px`,
                    height: `${size}px`,
                    opacity
                  }}
                  title={`${p.count} кликов`}
                />
              );
            })}
          </div>
          <div className="heatmap-legend">
            <span>Меньше</span>
            <div className="heatmap-gradient" />
            <span>Больше</span>
          </div>
        </div>
      </section>

      <section className="card">
        <h2>События (последние)</h2>
        <div className="events">
          {safeEvents.length === 0 && <div className="heatmap-empty">Нет данных</div>}
          {safeEvents.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th>Время</th>
                  <th>Тип</th>
                  <th>Путь</th>
                  <th>Заголовок</th>
                  <th>Meta</th>
                  <th>Источник</th>
                </tr>
              </thead>
              <tbody>
                {safeEvents.map((e, i) => (
                  <tr key={i}>
                    <td>{e.created_at}</td>
                    <td>{e.event_type}</td>
                    <td>{e.path}</td>
                    <td>{e.title}</td>
                    <td className="mono">{e.meta}</td>
                    <td>{e.utm_source || e.ref_domain || "direct"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </div>
  );
}
