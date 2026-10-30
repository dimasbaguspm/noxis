import { TablesTab } from "@/components/instance/tables-tab";
import { useInstanceContext } from "./layout";

export default function InstanceTablesTab() {
  const { instance, tables, schemaPending, selectedSchema, updateSchema } = useInstanceContext();
  return (
    <TablesTab
      tables={tables}
      loading={schemaPending}
      schema={selectedSchema}
      onSchema={updateSchema}
      connection={instance.name}
      driver={instance.driver ?? ""}
    />
  );
}
