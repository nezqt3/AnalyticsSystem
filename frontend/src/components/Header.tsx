import type { RealtimeResponse } from "../types/api";

type HeaderProps = {
  siteId: number;
  realtime: RealtimeResponse | null;
};

export function Header({ siteId, realtime }: HeaderProps) {
  return (
    <header className="hero">
      <div>
        <div className="eyebrow">Analytics System</div>
        <h1>Панель поведения пользователей по страницам</h1>
        <p>
          Видно, какие страницы читают, куда кликают, как глубоко скроллят и
          какие элементы реально притягивают внимание.
        </p>
      </div>
      <div className="hero-stats">
        <div className="hero-stat">
          <span>Site ID</span>
          <strong>{siteId}</strong>
        </div>
        <div className="hero-stat">
          <span>Активные сейчас</span>
          <strong>{realtime?.active_users ?? "-"}</strong>
        </div>
      </div>
    </header>
  );
}
