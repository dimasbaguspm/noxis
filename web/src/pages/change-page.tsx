import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { ChangeTable } from "@/components/changes/change-table";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { useChanges, type ChangeKind } from "@/hooks/use-data";

// ChangePage is the shared DML/DDL queue: submit through a modal, review on
// the detail page.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const { changes, submit } = useChanges(kind);

  const label = kind.toUpperCase();
  const description =
    kind === "ddl"
      ? "schema changes — validated against their rollback before review"
      : "data changes — validated against their rollback before review";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-lg font-semibold">{label}</h2>
        <ConnectionSelect value={connection} onChange={pick} />
        <span className="flex-1" />
        <ChangeSubmitDialog
          kind={kind}
          tables={tables}
          onSubmit={async (input) => (connection ? submit.run({ ...input, connection }) : undefined)}
        />
      </div>

      <Card>
        <CardHeader>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <ChangeTable changes={changes} emptyHint={`no ${label} changes yet`} />
        </CardContent>
      </Card>
    </div>
  );
}
