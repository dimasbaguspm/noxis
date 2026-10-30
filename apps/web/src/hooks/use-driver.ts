import { useParams } from "react-router-dom";
import { useDriversQuery } from "@/hooks/use-config";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import type { DriverInfo } from "@/lib/api";

interface DriverState {
  // driver is the engine the console is looking at: the one in the URL, else a fallback the sidebar can link to.
  driver: string;
  drivers: DriverInfo[];
  busy: boolean;
}

// useDriver reads the driver from the route and, outside a driver page, falls back to the first registered instance's
// driver and then to the first catalog entry, so the sidebar always has somewhere to point.
export function useDriver(): DriverState {
  const { driver } = useParams();
  const catalog = useDriversQuery();
  const instances = usePagedInstancesQuery({ perPage: 100 });
  const registered = instances.data?.instances ?? [];
  const fallback = registered[0]?.driver ?? catalog.data?.[0]?.driver ?? "";
  return {
    driver: driver ?? fallback,
    drivers: catalog.data ?? [],
    busy: catalog.isPending || instances.isPending,
  };
}
