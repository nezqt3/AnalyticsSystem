import { useCallback, useEffect, useState } from "react";

import { getAuthMe, login, logout } from "../api";
import type { AuthUser } from "../types/api";

type AuthState = {
  user: AuthUser | null;
  loading: boolean;
  error: string;
  loginAsAdmin: (email: string, password: string) => Promise<void>;
  logoutAsAdmin: () => Promise<void>;
};

function humanizeAuthError(message: string): string {
  if (message.includes("invalid credentials")) {
    return "Неверный email или пароль";
  }
  if (message.includes("auth is not configured")) {
    return "Авторизация не настроена: проверь `ADMIN_EMAIL`, `ADMIN_PASSWORD` и `SESSION_SECRET` в `.env`.";
  }
  if (message.includes("unauthorized")) {
    return "Сессия администратора не найдена. Войдите снова.";
  }
  return message || "Не удалось войти";
}

export function useAuth(): AuthState {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    async function bootstrap() {
      try {
        const currentUser = await getAuthMe();
        if (!active) return;
        setUser(currentUser);
      } catch {
        if (!active) return;
        setUser(null);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void bootstrap();
    return () => {
      active = false;
    };
  }, []);

  const loginAsAdmin = useCallback(async (email: string, password: string) => {
    setError("");
    try {
      const nextUser = await login(email, password);
      setUser(nextUser);
    } catch (reason) {
      const message =
        reason instanceof Error ? reason.message : "Не удалось войти";
      setError(humanizeAuthError(message));
      throw reason;
    }
  }, []);

  const logoutAsAdmin = useCallback(async () => {
    await logout();
    setUser(null);
  }, []);

  return { user, loading, error, loginAsAdmin, logoutAsAdmin };
}
