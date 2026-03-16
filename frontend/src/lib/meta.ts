export type EventMeta = {
  tag?: string;
  id?: string;
  class?: string;
  text?: string;
  href?: string;
  selector?: string;
  depth_pct?: number;
  location?: string;
};

export function parseEventMeta(meta: string): EventMeta {
  if (!meta) return {};
  try {
    return JSON.parse(meta) as EventMeta;
  } catch {
    return {};
  }
}
