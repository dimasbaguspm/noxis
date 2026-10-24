import { InstancePermissions } from "@/components/access/instance-permissions";
import { useInstanceContext } from "./layout";

export default function InstanceAccessTab() {
  const { instance } = useInstanceContext();
  return <InstancePermissions instance={instance} />;
}
