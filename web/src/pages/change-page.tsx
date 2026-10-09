import { useEffect, useState } from "react";
import { title } from "radash";
import { Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useApp } from "@/app/app-provider";
import { ChangeWizard } from "@/components/changes/change-wizard";
import { BatchChangeDialog } from "@/components/changes/batch-change-dialog";
import { ChangeTable } from "@/components/changes/change-table";
import type { Change } from "@/lib/api";
import { classify } from "@/lib/statements";
import { useChanges, type ChangeKind } from "@/hooks/use-data";

// ChangePage is the shared DML/DDL queue: raise through the wizard, read the
// kind mix at a glance, and keep immediate work apart from scheduled work.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { can, registry } = useApp();
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const { changes, all, submit, reload } = useChanges(kind, query || undefined);
  const label = kind.toUpperCase();

  // debounce the name filter so typing issues one request, not one per key
  useEffect(() => {
    const handle = setTimeout(() => setQuery(filter.trim()), 200);
    return () => clearTimeout(handle);
  }, [filter]);
  // raising a change needs a write grant on some SQL instance; the wizard
  // still lets the server refuse a specific connection
  const mayRaise = registry.sql.connections.some((c) => can(c.name, "write"));

  const immediate = changes.filter((c) => !c.scheduledAt);
  const scheduled = changes.filter((c) => Boolean(c.scheduledAt));
  const kindOf = (c: Change) => c.kind || classify(c.query ?? "");
  const dml = all.filter((c) => kindOf(c) === "dml").length;
  const ddl = all.filter((c) => kindOf(c) === "ddl").length;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Stat label={title("immediate")} value={immediate.length} />
        <Stat label={title("scheduled")} value={scheduled.length} />
        <Stat label="DML" value={dml} />
        <Stat label="DDL" value={ddl} />
        <div className="relative min-w-56">
          <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-7"
            placeholder="filter by instance"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        <span className="flex-1" />
        {mayRaise && (
          <>
            <BatchChangeDialog onSubmitted={() => void reload()} />
            <ChangeWizard kind={kind} onSubmit={(input) => submit.run(input)} />
          </>
        )}
      </div>

      <Group
        heading={title("immediate")}
        hint="runs when applied"
        changes={immediate}
        emptyHint={`no immediate ${label} changes`}
      />
      <Group
        heading={title("scheduled")}
        hint="runs when its slot passes"
        changes={scheduled}
        emptyHint={`no scheduled ${label} changes`}
      />
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <Badge variant="outline" className="gap-1.5">
      {label}
      <span className="font-semibold tabular-nums">{value}</span>
    </Badge>
  );
}

function Group({
  heading,
  hint,
  changes,
  emptyHint,
}: {
  heading: string;
  hint: string;
  changes: Change[];
  emptyHint: string;
}) {
  return (
    <Card>
      <CardContent className="pt-4">
        <div className="mb-3 flex flex-wrap items-baseline gap-2">
          <h2 className="text-sm font-medium">{heading}</h2>
          <span className="text-xs text-muted-foreground">{hint}</span>
          <span className="flex-1" />
          <span className="text-xs text-muted-foreground tabular-nums">{changes.length}</span>
        </div>
        <ChangeTable changes={changes} emptyHint={emptyHint} />
      </CardContent>
    </Card>
  );
}
