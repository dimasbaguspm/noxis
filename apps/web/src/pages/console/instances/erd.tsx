import { InstanceErd } from "@/components/instance/erd-tab";
import { useInstanceContext } from "./layout";

export default function InstanceErdTab() {
  const { instance } = useInstanceContext();
  return <InstanceErd driver={instance.driver} connection={instance.name} instanceId={instance.id} />;
}
