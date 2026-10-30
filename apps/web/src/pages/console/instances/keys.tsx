import { CacheBrowser } from "@/components/instance/cache-browser";
import { useInstanceContext } from "./layout";

export default function InstanceKeysTab() {
  const { instance } = useInstanceContext();
  return <CacheBrowser driver={instance.driver} connection={instance.name} />;
}
