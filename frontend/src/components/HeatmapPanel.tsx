import type { HeatmapPoint } from "../types/api";

type HeatmapPanelProps = {
  points: HeatmapPoint[];
};

export function HeatmapPanel({ points }: HeatmapPanelProps) {
  const peak = Math.max(...points.map((point) => point.count), 1);

  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Heatmap</div>
          <div className="panel-subtitle">Тепловая карта кликов по экрану.</div>
        </div>
      </div>
      <div className="heatmap-canvas">
        {points.length === 0 && (
          <div className="empty-state">Нет данных для построения карты.</div>
        )}
        {points.map((point) => {
          const intensity = point.count / peak;
          const size = 18 + intensity * 42;
          return (
            <div
              key={`${point.x_pct}-${point.y_pct}`}
              className="heat-dot"
              style={{
                left: `${point.x_pct}%`,
                top: `${point.y_pct}%`,
                width: `${size}px`,
                height: `${size}px`,
                opacity: 0.25 + intensity * 0.7,
              }}
              title={`${point.count} кликов`}
            />
          );
        })}
      </div>
    </section>
  );
}
