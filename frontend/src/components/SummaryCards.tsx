import type { OverviewMetrics } from "../types/api";

type SummaryCardsProps = {
  overview: OverviewMetrics | null;
};

export function SummaryCards({ overview }: SummaryCardsProps) {
  const items = [
    { label: "Просмотры", value: overview?.pageviews ?? 0 },
    { label: "Клики", value: overview?.clicks ?? 0 },
    { label: "Посетители", value: overview?.unique_visitors ?? 0 },
    { label: "Топ-страница", value: overview?.top_page || "-" },
  ];

  return (
    <section className="summary-grid">
      {items.map((item) => (
        <article key={item.label} className="metric-card summary-card">
          <span>{item.label}</span>
          <strong>{item.value}</strong>
        </article>
      ))}
    </section>
  );
}
