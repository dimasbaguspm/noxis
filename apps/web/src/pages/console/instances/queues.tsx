import { useOutletContext } from "react-router-dom";
import { ChangePage } from "@/pages/console/changes";
import { RequestsQueue } from "@/pages/console/requests";
import type { InstanceContextValue } from "@/pages/console/instances/layout";
import type { ChangeKind } from "@/hooks/use-changes";

// The instance layout already loaded the connection, so a review tab reads the driver and the name from it instead of
// fetching the instance a second time.
function useScope() {
  const { instance } = useOutletContext<InstanceContextValue>();
  return { driver: instance.driver ?? "", connection: instance.name ?? "" };
}

// InstanceChangeQueue is one instance's DML or DDL queue: the same table, filters and paging as the driver's queue.
export function InstanceChangeQueue({ kind }: { kind: ChangeKind }) {
  const scope = useScope();
  if (!scope.driver) return null;
  return <ChangePage kind={kind} scope={scope} />;
}

// InstanceRequestsQueue is one document store's request queue, scoped the same way.
export function InstanceRequestsQueue() {
  const scope = useScope();
  if (!scope.driver) return null;
  return <RequestsQueue scope={scope} />;
}
