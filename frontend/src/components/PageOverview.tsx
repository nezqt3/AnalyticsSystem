import { formatDateTime, formatPercent } from "../lib/date";
import type { PageAnalytics } from "../types/api";

type PageOverviewProps = {
  analytics: PageAnalytics | null;
};

export function PageOverview({ analytics }: PageOverviewProps) {
  return (
    <section className="panel overview-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Сводка по странице</div>
          <div className="panel-subtitle">
            Ключевые сигналы по выбранному URL.
          </div>
        </div>
      </div>
      {!analytics && (
        <div className="empty-state">
          Выберите страницу, чтобы увидеть детализацию.
        </div>
      )}
      {analytics && (
        <>
          <div className="stats-grid">
            <article className="metric-card">
              <span>Просмотры</span>
              <strong>{analytics.pageviews}</strong>
            </article>
            <article className="metric-card">
              <span>Клики</span>
              <strong>{analytics.clicks}</strong>
            </article>
            <article className="metric-card">
              <span>Формы</span>
              <strong>{analytics.form_submissions}</strong>
            </article>
            <article className="metric-card">
              <span>Средний скролл</span>
              <strong>{Math.round(analytics.avg_scroll_depth)}%</strong>
            </article>
          </div>
          <div className="overview-meta">
            <span>Посетители: {analytics.unique_visitors}</span>
            <span>
              Последняя активность:{" "}
              {formatDateTime(analytics.last_interaction_at)}
            </span>
          </div>
          <div className="target-list">
            <div className="section-caption">Топ элементов по кликам</div>
            {analytics.top_targets.length === 0 && (
              <div className="empty-state compact">Нет данных по кликам.</div>
            )}
            {analytics.top_targets.map((target) => (
              <div key={target.selector} className="target-item">
                <div>
                  <strong>{target.selector}</strong>
                  <p>
                    {target.text ||
                      target.href ||
                      target.tag ||
                      "Без текстового описания"}
                  </p>
                </div>
                <div className="target-values">
                  <span>{target.count} кликов</span>
                  <span>{formatPercent(target.share)}</span>
                </div>
              </div>
            ))}
          </div>
          <div className="scroll-depths">
            <div className="section-caption">Глубина скролла</div>
            {analytics.scroll_depths.length === 0 && (
              <div className="empty-state compact">
                События скролла ещё не накопились.
              </div>
            )}
            {analytics.scroll_depths.map((point) => (
              <div key={point.depth} className="scroll-row">
                <span>{point.depth}%</span>
                <span className="series-bar-track">
                  <span
                    className="series-bar soft"
                    style={{ width: `${Math.min(point.depth, 100)}%` }}
                  />
                </span>
                <strong>{point.count}</strong>
              </div>
            ))}
          </div>
        </>
      )}
    </section>
  );
}
