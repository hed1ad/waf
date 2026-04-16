import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ShieldAlert, ShieldCheck, Globe, Zap } from "lucide-react";

interface Props {
  total: number;
  blocked: number;
}

export function StatsCards({ total, blocked }: Props) {
  const passed = total - blocked;
  const blockRate = total > 0 ? ((blocked / total) * 100).toFixed(1) : "0";

  const cards = [
    {
      title: "Total Requests",
      value: total.toLocaleString(),
      icon: Zap,
      color: "text-blue-400",
    },
    {
      title: "Blocked",
      value: blocked.toLocaleString(),
      sub: `${blockRate}% block rate`,
      icon: ShieldAlert,
      color: "text-red-400",
    },
    {
      title: "Passed",
      value: passed.toLocaleString(),
      icon: ShieldCheck,
      color: "text-green-400",
    },
    {
      title: "Block Rate",
      value: `${blockRate}%`,
      icon: Globe,
      color: "text-yellow-400",
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {cards.map((c) => (
        <Card key={c.title} className="bg-card border-border">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {c.title}
            </CardTitle>
            <c.icon className={`h-4 w-4 ${c.color}`} />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{c.value}</p>
            {c.sub && <p className="text-xs text-muted-foreground mt-1">{c.sub}</p>}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
