import type { AuthUser, RealtimeResponse } from "../types/api";

type HeaderProps = {
  siteId: number;
  realtime: RealtimeResponse | null;
  user: AuthUser;
  onLogout: () => Promise<void>;
};

export function Header({ siteId, realtime, user, onLogout }: HeaderProps) {
  return (
    <header className="hero">
      <div>
        <div className="eyebrow">Analytics System</div>
        <h1>Панель поведения пользователей по страницам</h1>
        <p>
          Видно, какие страницы читают, в какой момент на них заходят, какие
          точки входа самые сильные и где пользователь взаимодействует с
          интерфейсом.
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
        <div className="hero-stat hero-stat-user">
          <span>Админ</span>
          <strong>{user.email}</strong>
          <button
            className="ghost-button"
            onClick={() => void onLogout()}
            type="button"
          >
            Выйти
          </button>
        </div>
      </div>
    </header>
  );
}
