import { formatDateTime } from "../lib/date";
import { parseEventMeta } from "../lib/meta";
import type { EventItem } from "../types/api";

type EventsTableProps = {
  events: EventItem[];
};

export function EventsTable({ events }: EventsTableProps) {
  return (
    <section className="panel events-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Последние события</div>
          <div className="panel-subtitle">
            Сырые события по выбранной странице.
          </div>
        </div>
      </div>
      {events.length === 0 && (
        <div className="empty-state">События ещё не пришли.</div>
      )}
      {events.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Время</th>
                <th>Тип</th>
                <th>Элемент</th>
                <th>Источник</th>
                <th>Meta</th>
              </tr>
            </thead>
            <tbody>
              {events.map((event, index) => {
                const meta = parseEventMeta(event.meta);
                return (
                  <tr key={`${event.created_at}-${index}`}>
                    <td>{formatDateTime(event.created_at)}</td>
                    <td>{event.event_type}</td>
                    <td>{meta.selector || meta.text || event.title || "-"}</td>
                    <td>{event.utm_source || event.ref_domain || "direct"}</td>
                    <td className="mono">{event.meta || "-"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
