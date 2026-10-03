import { Fragment, useState } from "react";
import { Check, ChevronDown, ChevronUp, Play, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Change } from "@/lib/api";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applied: "default",
  failed: "destructive",
};

// ChangeTable lists changes with the approval workflow actions.
export function ChangeTable({
  changes,
  onApprove,
  onApply,
  onReject,
  emptyHint,
}: {
  changes: Change[];
  onApprove: (id: string) => Promise<void>;
  onApply: (id: string) => Promise<void>;
  onReject: (id: string) => Promise<void>;
  emptyHint: string;
}) {
  const [expanded, setExpanded] = useState<string | null>(null);

  if (changes.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyHint}</p>;
  }

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[90px]">status</TableHead>
            <TableHead>notes</TableHead>
            <TableHead className="w-[140px]">connection</TableHead>
            <TableHead className="w-[220px]">actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {changes.map((change) => {
            const id = change.id ?? "";
            const open = expanded === id;
            return (
              <Fragment key={id}>
                <TableRow className="align-top">
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>
                      {change.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <button
                      className="flex items-center gap-1 text-left text-sm hover:underline"
                      onClick={() => setExpanded(open ? null : id)}
                    >
                      {open ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                      {change.notes}
                    </button>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">{change.connection}</TableCell>
                  <TableCell className="flex flex-wrap gap-1.5">
                    {(change.status === "validated" || change.status === "unverified") && (
                      <Button size="xs" variant="secondary" onClick={() => void onApprove(id)}>
                        <Check /> Approve
                      </Button>
                    )}
                    {change.status === "approved" && (
                      <Button size="xs" onClick={() => void onApply(id)}>
                        <Play /> Apply
                      </Button>
                    )}
                    <Button size="xs" variant="ghost" onClick={() => void onReject(id)}>
                      <Trash2 /> Discard
                    </Button>
                  </TableCell>
                </TableRow>
                {open && (
                  <TableRow>
                    <TableCell colSpan={4} className="bg-muted/30">
                      <div className="grid gap-2 md:grid-cols-2">
                        <Detail label="query" value={change.query ?? ""} />
                        <Detail label="rollback" value={change.rollback ?? ""} />
                      </div>
                      <div className="mt-2 grid gap-1">
                        {(change.checks ?? []).map((check) => (
                          <p
                            key={`${check.target}:${check.error ?? check.detail ?? ""}`}
                            className={check.ok ? "text-xs text-muted-foreground" : "text-xs text-destructive"}
                          >
                            {check.ok ? "✓" : "✗"} <span className="font-medium">{check.target}</span>
                            {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
                          </p>
                        ))}
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </Fragment>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <pre className="max-h-40 overflow-auto rounded-md border border-border bg-background p-2 font-mono text-xs">
        {value}
      </pre>
    </div>
  );
}
