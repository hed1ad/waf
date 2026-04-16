// API_URL is used for server-side fetches (SSR/RSC inside Docker → internal hostname).
// NEXT_PUBLIC_API_URL is used client-side (browser → exposed port).
const API_BASE =
  typeof window === "undefined"
    ? (process.env.API_URL ?? "http://api:4000")
    : (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:4000");

export interface StatsPoint {
  minute: string;
  total: number;
  blocked: number;
}

export interface TopIP {
  client_ip: string;
  country_code: string;
  total_hits: number;
  blocked_hits: number;
  last_seen: string;
}

export interface StatsResponse {
  total: number;
  blocked: number;
  timeseries: StatsPoint[];
  top_ips: TopIP[];
}

export interface Event {
  timestamp: string;
  transaction_id: string;
  client_ip: string;
  method: string;
  uri: string;
  host: string;
  status: number;
  action: string;
  anomaly_score: number;
  country_code: string;
  country_name: string;
  rule_ids: number[];
  rule_msgs: string[];
}

export interface EventsResponse {
  data: Event[];
}

async function apiFetch<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = new URL(`${API_BASE}/api${path}`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => v && url.searchParams.set(k, v));
  }
  const res = await fetch(url.toString(), { cache: "no-store" });
  if (!res.ok) throw new Error(`API error ${res.status}`);
  return res.json();
}

export const api = {
  stats: (from?: string, to?: string) =>
    apiFetch<StatsResponse>("/events/stats", {
      from: from ?? new Date(Date.now() - 86400_000).toISOString(),
      to: to ?? new Date().toISOString(),
    }),

  events: (params?: { action?: string; ip?: string; country?: string; limit?: string; offset?: string }) =>
    apiFetch<EventsResponse>("/events", {
      from: new Date(Date.now() - 86400_000).toISOString(),
      to: new Date().toISOString(),
      ...params,
    }),
};
