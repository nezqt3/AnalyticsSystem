import { useEffect, useState } from "react";

import { API_BASE, getRealtime } from "../api";
import type { RealtimeResponse } from "../types/api";

type RealtimeState = {
  realtime: RealtimeResponse | null;
  error: string;
};

export function useRealtime(siteId: number): RealtimeState {
  const [realtime, setRealtime] = useState<RealtimeResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    let socket: WebSocket | null = null;
    let pollTimer: number | null = null;

    const startPolling = () => {
      const poll = async () => {
        try {
          const response = await getRealtime(siteId);
          if (!active) return;
          setRealtime(response);
          setError("");
        } catch (pollError) {
          if (!active) return;
          setError(
            pollError instanceof Error
              ? pollError.message
              : "Realtime недоступен",
          );
        }
      };

      void poll();
      pollTimer = window.setInterval(poll, 5000);
    };

    try {
      socket = new WebSocket(
        `${API_BASE.replace(/^http/, "ws")}/ws/realtime?site_id=${siteId}`,
      );
      socket.onmessage = (event) => {
        if (!active) return;
        try {
          setRealtime(JSON.parse(event.data) as RealtimeResponse);
          setError("");
        } catch {
          setError("Некорректный realtime-пакет");
        }
      };
      socket.onerror = () => {
        if (active && pollTimer === null) startPolling();
      };
      socket.onclose = () => {
        if (active && pollTimer === null) startPolling();
      };
    } catch {
      startPolling();
    }

    return () => {
      active = false;
      if (pollTimer !== null) {
        window.clearInterval(pollTimer);
      }
      socket?.close();
    };
  }, [siteId]);

  return { realtime, error };
}
