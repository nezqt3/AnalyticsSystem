import type { TimelinePoint } from "../types/api";

type TimelineChartProps = {
  points: TimelinePoint[];
  title: string;
  subtitle: string;
};

export function TimelineChart({ points, title, subtitle }: TimelineChartProps) {
  const max = Math.max(...points.map((point) => point.count), 1);

  return (
    <section className="panel chart-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">{title}</div>
          <div className="panel-subtitle">{subtitle}</div>
        </div>
      </div>
      {points.length === 0 && (
        <div className="empty-state">Пока нет точек для графика.</div>
      )}
      {points.length > 0 && (
        <div className="timeline-chart">
          {points.map((point) => (
            <div key={point.label} className="timeline-bar-item">
              <span className="timeline-bar-track">
                <span
                  className="timeline-bar-fill"
                  style={{
                    height: `${Math.max((point.count / max) * 100, 6)}%`,
                  }}
                />
              </span>
              <strong>{point.count}</strong>
              <span className="timeline-label">{point.label}</span>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
