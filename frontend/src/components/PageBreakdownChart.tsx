import type { PageStat } from "../types/api";

type PageBreakdownChartProps = {
  pages: PageStat[];
};

export function PageBreakdownChart({ pages }: PageBreakdownChartProps) {
  const topPages = pages.slice(0, 6);
  const max = Math.max(...topPages.map((page) => page.pageviews), 1);

  return (
    <section className="panel chart-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Входы по страницам</div>
          <div className="panel-subtitle">
            Какие страницы чаще всего открывают пользователи.
          </div>
        </div>
      </div>
      {topPages.length === 0 && (
        <div className="empty-state">Нет данных по страницам.</div>
      )}
      <div className="stack-list">
        {topPages.map((page) => (
          <div key={page.path} className="stack-row">
            <div>
              <strong>{page.path}</strong>
              <span>{page.pageviews} входов</span>
            </div>
            <span className="stack-bar-track">
              <span
                className="stack-bar"
                style={{ width: `${(page.pageviews / max) * 100}%` }}
              />
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
