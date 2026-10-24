import { InstanceAccess } from "@/components/access/instance-access";
import { useInstanceContext } from "./layout";

export default function InstanceAccessTab() {
  const { instance } = useInstanceContext();
  return <InstanceAccess instance={instance} />;
}
