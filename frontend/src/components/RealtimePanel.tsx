import type { RealtimeResponse } from "../types/api";

type RealtimePanelProps = {
  realtime: RealtimeResponse | null;
};

export function RealtimePanel({ realtime }: RealtimePanelProps) {
  const peak = Math.max(
    ...(realtime?.series.map((point) => point.count) ?? [1]),
  );

  return (
    <section className="panel realtime-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Realtime</div>
          <div className="panel-subtitle">Последние 5 минут активности.</div>
        </div>
      </div>
      <div className="realtime-content">
        <div className="metric-card accent">
          <span>Активные пользователи</span>
          <strong>{realtime?.active_users ?? 0}</strong>
        </div>
        <div className="series-list">
          {(realtime?.series ?? []).map((point) => (
            <div key={point.minute} className="series-row">
              <span>{point.minute}</span>
              <span className="series-bar-track">
                <span
                  className="series-bar"
                  style={{ width: `${(point.count / peak) * 100}%` }}
                />
              </span>
              <strong>{point.count}</strong>
            </div>
          ))}
          {(realtime?.series ?? []).length === 0 && (
            <div className="empty-state">Нет realtime-данных.</div>
          )}
        </div>
      </div>
    </section>
  );
}
