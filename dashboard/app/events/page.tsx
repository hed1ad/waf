import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import { format } from "date-fns";

export const revalidate = 10;

export default async function EventsPage() {
  let events: Awaited<ReturnType<typeof api.events>>["data"] = [];
  try {
    const res = await api.events({ limit: "100" });
    events = res.data ?? [];
  } catch {
    // API not available
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Events</h1>
        <p className="text-sm text-muted-foreground">Last 24 hours — most recent first</p>
      </div>

      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="border-border hover:bg-transparent">
              <TableHead className="text-xs text-muted-foreground">Time</TableHead>
              <TableHead className="text-xs text-muted-foreground">Action</TableHead>
              <TableHead className="text-xs text-muted-foreground">IP</TableHead>
              <TableHead className="text-xs text-muted-foreground">Country</TableHead>
              <TableHead className="text-xs text-muted-foreground">Method</TableHead>
              <TableHead className="text-xs text-muted-foreground">URI</TableHead>
              <TableHead className="text-xs text-muted-foreground">Status</TableHead>
              <TableHead className="text-xs text-muted-foreground">Rules hit</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.length === 0 && (
              <TableRow>
                <TableCell colSpan={8} className="text-center text-muted-foreground py-12 text-sm">
                  No events yet. Send some traffic through the WAF proxy on :8080.
                </TableCell>
              </TableRow>
            )}
            {events.map((ev) => (
              <TableRow key={ev.transaction_id} className="border-border text-xs">
                <TableCell className="text-muted-foreground whitespace-nowrap">
                  {format(new Date(ev.timestamp), "HH:mm:ss")}
                </TableCell>
                <TableCell>
                  <Badge variant={ev.action === "block" ? "destructive" : "outline"}>
                    {ev.action.toUpperCase()}
                  </Badge>
                </TableCell>
                <TableCell className="font-mono">{ev.client_ip}</TableCell>
                <TableCell>{ev.country_code || "—"}</TableCell>
                <TableCell>{ev.method}</TableCell>
                <TableCell className="max-w-xs truncate">{ev.uri}</TableCell>
                <TableCell>{ev.status}</TableCell>
                <TableCell>
                  {ev.rule_ids?.length > 0 ? (
                    <span className="text-red-400">{ev.rule_ids.length}</span>
                  ) : "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
