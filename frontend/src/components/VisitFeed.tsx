import { formatDateTime } from "../lib/date";
import type { VisitEntry } from "../types/api";

type VisitFeedProps = {
  visits: VisitEntry[];
};

export function VisitFeed({ visits }: VisitFeedProps) {
  return (
    <section className="panel events-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Когда и на какие страницы заходили</div>
          <div className="panel-subtitle">
            Последние входы пользователей по страницам.
          </div>
        </div>
      </div>
      {visits.length === 0 && (
        <div className="empty-state">Нет последних входов.</div>
      )}
      {visits.length > 0 && (
        <div className="visit-feed">
          {visits.map((visit, index) => (
            <div
              key={`${visit.created_at}-${visit.path}-${index}`}
              className="visit-item"
            >
              <div>
                <strong>{visit.path}</strong>
                <p>{visit.title || visit.source}</p>
              </div>
              <div className="visit-meta">
                <span>{formatDateTime(visit.created_at)}</span>
                <span>{visit.source}</span>
                <span className="mono">
                  {visit.session_id || "session unknown"}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
