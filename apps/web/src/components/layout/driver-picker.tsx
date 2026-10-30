import { useNavigate } from "react-router-dom";
import { Check, ChevronsUpDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { PAGE_ROUTES } from "@/constants/routes";
import { capabilitiesOf, capabilityIcon, driverLabel, firstPageOf } from "@/lib/console-nav";
import type { DriverInfo } from "@/lib/api";

// DriverPicker is the sidebar's engine switch: it lists every driver the catalog declares — a driver with no instance is
// still selectable, because the page it opens is where an instance gets registered — and opens that driver's first page.
export function DriverPicker({ driver, drivers }: { driver: string; drivers: DriverInfo[] }) {
  const navigate = useNavigate();
  const Current = capabilityIcon(capabilitiesOf(drivers, driver));
  const label = driverLabel(drivers, driver);

  function open(next: string) {
    const page = firstPageOf(drivers, next);
    if (page) navigate(PAGE_ROUTES.DRIVER(next, page));
  }

  return (
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <SidebarMenuButton tooltip={label || "Pick a driver"} aria-label="Pick a driver" />
          }
        >
          <Current />
          <span className="truncate">{label || "Pick a driver"}</span>
          <ChevronsUpDown className="ml-auto size-4 text-muted-foreground group-data-[collapsible=icon]:hidden" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-56">
          {drivers.map((entry) => {
            const Icon = capabilityIcon(capabilitiesOf(drivers, entry.driver));
            return (
              <DropdownMenuItem key={entry.driver} onClick={() => open(entry.driver)}>
                <Icon />
                <span className="truncate">{entry.label}</span>
                {entry.driver === driver && <Check className="ml-auto size-3.5" />}
              </DropdownMenuItem>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  );
}
