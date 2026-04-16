"use client";

import { useEffect, useRef, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { format } from "date-fns";

interface LiveEvent {
  id: string;
  ts: number;
  ip: string;
  method: string;
  uri: string;
  status: number;
  action: string;
  country: string;
  rule_count: number;
}

const MAX_EVENTS = 50;
const WS_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:4000";

export function LiveFeed() {
  const [events, setEvents] = useState<LiveEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource(`${WS_URL}/api/stream`);
    esRef.current = es;

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (e) => {
      try {
        const ev: LiveEvent = JSON.parse(e.data);
        setEvents((prev) => [ev, ...prev].slice(0, MAX_EVENTS));
      } catch {
        // malformed event — ignore
      }
    };

    return () => {
      es.close();
      setConnected(false);
    };
  }, []);

  return (
    <Card className="bg-card border-border">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">Live Feed</CardTitle>
        <div className="flex items-center gap-1.5">
          <span className={`h-2 w-2 rounded-full ${connected ? "bg-green-400 animate-pulse" : "bg-zinc-600"}`} />
          <span className="text-xs text-muted-foreground">{connected ? "live" : "disconnected"}</span>
        </div>
      </CardHeader>
      <CardContent className="p-0 max-h-72 overflow-y-auto">
        {events.length === 0 && (
          <p className="text-center text-sm text-muted-foreground py-8">Waiting for events…</p>
        )}
        {events.map((ev) => (
          <div
            key={ev.id}
            className="flex items-center gap-3 px-4 py-2 border-b border-border last:border-0 text-xs"
          >
            <span className="text-muted-foreground w-12 shrink-0">
              {format(new Date(ev.ts), "HH:mm:ss")}
            </span>
            <Badge
              variant={ev.action === "block" ? "destructive" : "outline"}
              className="w-12 justify-center text-xs shrink-0"
            >
              {ev.action === "block" ? "BLOCK" : "PASS"}
            </Badge>
            <span className="text-muted-foreground w-8 shrink-0">{ev.country || "??"}</span>
            <span className="font-mono">{ev.ip}</span>
            <span className="text-muted-foreground shrink-0">{ev.method}</span>
            <span className="truncate text-foreground flex-1">{ev.uri}</span>
            {ev.rule_count > 0 && (
              <span className="text-red-400 shrink-0">{ev.rule_count} rules</span>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
