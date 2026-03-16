type FiltersProps = {
  from: string;
  to: string;
  bucket: number;
  onFromChange: (value: string) => void;
  onToChange: (value: string) => void;
  onBucketChange: (value: number) => void;
};

export function Filters({
  from,
  to,
  bucket,
  onFromChange,
  onToChange,
  onBucketChange,
}: FiltersProps) {
  return (
    <section className="panel controls-panel">
      <div>
        <div className="panel-title">Фильтры</div>
        <div className="panel-subtitle">
          Управляем периодом и плотностью тепловой карты.
        </div>
      </div>
      <div className="filters-grid">
        <label>
          <span>С</span>
          <input
            type="date"
            value={from}
            onChange={(event) => onFromChange(event.target.value)}
          />
        </label>
        <label>
          <span>По</span>
          <input
            type="date"
            value={to}
            onChange={(event) => onToChange(event.target.value)}
          />
        </label>
        <label>
          <span>Heatmap bucket</span>
          <input
            type="number"
            min={1}
            max={25}
            value={bucket}
            onChange={(event) =>
              onBucketChange(Number(event.target.value) || 5)
            }
          />
        </label>
      </div>
    </section>
  );
}
