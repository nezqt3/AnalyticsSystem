import { formatDateTime } from "../lib/date";
import type { PageStat } from "../types/api";

type PageListProps = {
  pages: PageStat[];
  selectedPath: string;
  onSelect: (path: string) => void;
};

export function PageList({ pages, selectedPath, onSelect }: PageListProps) {
  return (
    <section className="panel page-list-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Страницы</div>
          <div className="panel-subtitle">
            Рейтинг страниц по просмотрам и кликам.
          </div>
        </div>
        <span className="pill">{pages.length}</span>
      </div>
      <div className="page-list">
        {pages.length === 0 && (
          <div className="empty-state">
            Пока нет страниц в выбранном диапазоне.
          </div>
        )}
        {pages.map((page) => {
          const active = page.path === selectedPath;
          return (
            <button
              key={page.path}
              className={active ? "page-item active" : "page-item"}
              onClick={() => onSelect(page.path)}
              type="button"
            >
              <div className="page-item-top">
                <strong>{page.path}</strong>
                <span>{page.pageviews} pv</span>
              </div>
              <div className="page-item-meta">
                <span>{page.clicks} кликов</span>
                <span>{page.unique_visitors} посетителей</span>
                <span>{formatDateTime(page.last_seen)}</span>
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}
