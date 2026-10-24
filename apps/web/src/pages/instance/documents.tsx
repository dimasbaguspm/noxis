import { MongoExplorer } from "@/components/instance/mongo-explorer";
import { useInstanceContext } from "./layout";

export default function InstanceDocumentsTab() {
  const { instance } = useInstanceContext();
  return <MongoExplorer connection={instance.name} />;
}
