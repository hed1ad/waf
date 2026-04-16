import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TopIP } from "@/lib/api";

interface Props {
  ips: TopIP[];
}

export function TopIPsTable({ ips }: Props) {
  return (
    <Card className="bg-card border-border">
      <CardHeader>
        <CardTitle className="text-sm font-medium">Top Attacking IPs</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow className="border-border hover:bg-transparent">
              <TableHead className="text-xs text-muted-foreground">IP</TableHead>
              <TableHead className="text-xs text-muted-foreground">Country</TableHead>
              <TableHead className="text-xs text-muted-foreground text-right">Hits</TableHead>
              <TableHead className="text-xs text-muted-foreground text-right">Blocked</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {ips.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground text-sm py-8">
                  No data yet
                </TableCell>
              </TableRow>
            )}
            {ips.map((ip) => (
              <TableRow key={ip.client_ip} className="border-border">
                <TableCell className="font-mono text-xs">{ip.client_ip}</TableCell>
                <TableCell>
                  <Badge variant="outline" className="text-xs">
                    {ip.country_code || "??"}
                  </Badge>
                </TableCell>
                <TableCell className="text-right text-sm">{ip.total_hits.toLocaleString()}</TableCell>
                <TableCell className="text-right">
                  <span className="text-red-400 text-sm font-medium">
                    {ip.blocked_hits.toLocaleString()}
                  </span>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
