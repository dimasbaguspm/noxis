import { Check, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useDecideRequest, useToApproveQuery } from "@/hooks/use-access-requests";
import { formatDateTime } from "@/lib/dates";

// AccessInbox is the pending requests the caller may decide (super admin, or the admin of the connection).
export function AccessInbox({ connection }: { connection?: string }) {
  const { data } = useToApproveQuery();
  const decide = useDecideRequest();

  const rows = (data?.requests ?? []).filter((r) => !connection || r.connection === connection);
  if (rows.length === 0) return null;

  return (
    <Card>
      <CardHeader className="gap-1">
        <CardTitle className="text-base">Pending requests</CardTitle>
        <CardDescription>Access asked for, waiting on your decision.</CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="flex flex-col gap-2">
          {rows.map((r) => (
            <li
              key={r.id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-md border p-3"
            >
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2 text-sm">
                  <span className="font-mono text-xs">{r.subject}</span>
                  <span className="text-muted-foreground">wants</span>
                  <Badge variant="secondary">{r.level}</Badge>
                  <span className="text-muted-foreground">on {r.connection}</span>
                </div>
                <div className="truncate text-xs text-muted-foreground">
                  {r.reason || "no reason given"} · requested {formatDateTime(r.requestedAt)}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={decide.isPending}
                  onClick={() => decide.mutate({ id: r.id ?? "", approve: true })}
                >
                  <Check /> Approve
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={decide.isPending}
                  onClick={() => decide.mutate({ id: r.id ?? "", approve: false })}
                >
                  <X /> Reject
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
