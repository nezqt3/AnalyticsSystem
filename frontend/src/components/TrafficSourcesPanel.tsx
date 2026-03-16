import type { TrafficSource } from "../types/api";

type TrafficSourcesPanelProps = {
  items: TrafficSource[];
};

export function TrafficSourcesPanel({ items }: TrafficSourcesPanelProps) {
  const total = items.reduce((sum, item) => sum + item.count, 0) || 1;

  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Источники трафика</div>
          <div className="panel-subtitle">
            Откуда пришли пользователи на выбранную страницу.
          </div>
        </div>
      </div>
      <div className="stack-list">
        {items.length === 0 && (
          <div className="empty-state">Нет данных по источникам.</div>
        )}
        {items.map((item) => (
          <div key={item.source} className="stack-row">
            <div>
              <strong>{item.source}</strong>
              <span>{item.count} событий</span>
            </div>
            <span className="stack-bar-track">
              <span
                className="stack-bar"
                style={{ width: `${(item.count / total) * 100}%` }}
              />
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
