import { StatsCards } from "@/components/stats-cards";
import { TrafficChart } from "@/components/traffic-chart";
import { TopIPsTable } from "@/components/top-ips-table";
import { LiveFeed } from "@/components/live-feed";
import { api } from "@/lib/api";

export const revalidate = 30;

export default async function OverviewPage() {
  let stats: { total: number; blocked: number; timeseries: import("@/lib/api").StatsPoint[]; top_ips: import("@/lib/api").TopIP[] } = { total: 0, blocked: 0, timeseries: [], top_ips: [] };
  try {
    stats = await api.stats();
  } catch {
    // API not available yet — render empty state
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Overview</h1>
        <p className="text-sm text-muted-foreground">Last 24 hours</p>
      </div>

      <StatsCards total={stats.total} blocked={stats.blocked} />

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <TrafficChart data={stats.timeseries} />
        </div>
        <TopIPsTable ips={stats.top_ips} />
      </div>

      <LiveFeed />
    </div>
  );
}
