import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { ChangeTable } from "@/components/changes/change-table";
import { useApp } from "@/app/app-provider";
import { useChanges, type ChangeKind } from "@/hooks/use-data";

// ChangePage is the shared DML/DDL workbench: submit for validation, approve
// (self-approval is allowed), then apply.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { active } = useApp();
  const { changes, submit, approve, apply, reject } = useChanges(kind);

  const label = kind.toUpperCase();
  const description =
    kind === "ddl"
      ? "schema changes — validated against their rollback before you approve"
      : "data changes — validated against their rollback before you approve";

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3 text-base">
            {label} changes
            <span className="flex-1" />
            <ChangeSubmitDialog kind={kind} onSubmit={(input) => submit.run(input)} />
          </CardTitle>
          <CardDescription>
            {description} · target: {active ?? "no connection"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ChangeTable
            changes={changes}
            onApprove={(id) => approve.run(id).then(() => undefined)}
            onApply={(id) => apply.run(id).then(() => undefined)}
            onReject={(id) => reject.run(id).then(() => undefined)}
            emptyHint={`no ${label} changes yet`}
          />
        </CardContent>
      </Card>
    </div>
  );
}
