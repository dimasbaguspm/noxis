import { Card, CardContent } from "@/components/ui/card";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { ChangeTable } from "@/components/changes/change-table";
import { useChanges, type ChangeKind } from "@/hooks/use-data";

// ChangePage is the shared DML/DDL queue: submit through a modal, review on
// the detail page.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { changes, submit } = useChanges(kind);

  const label = kind.toUpperCase();

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <span className="flex-1" />
        <ChangeSubmitDialog kind={kind} onSubmit={(input) => submit.run(input)} />
      </div>

      <Card>
        <CardContent className="pt-4">
          <ChangeTable changes={changes} emptyHint={`no ${label} changes yet`} />
        </CardContent>
      </Card>
    </div>
  );
}
