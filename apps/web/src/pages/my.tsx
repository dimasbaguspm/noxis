import { NavLink, Outlet } from "react-router-dom";
import { KeyRound, Settings, UserRound } from "lucide-react";
import { Workspace } from "@/components/layout/workspace";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

const ITEMS = [
  { to: PAGE_ROUTES.MY, label: "Profile", icon: UserRound, end: true },
  { to: PAGE_ROUTES.MY_SETTINGS, label: "Settings", icon: Settings, end: false },
  { to: PAGE_ROUTES.MY_ACCESS, label: "Access", icon: KeyRound, end: false },
] as const;

// MyLayout keeps the account areas together: a stable selection on the left and the chosen page on the right.
export default function MyLayout() {
  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="grid gap-1">
            {ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    cn(
                      "flex items-center gap-2 rounded-md px-2 py-2 text-sm text-muted-foreground hover:bg-accent/60 hover:text-foreground",
                      isActive && "bg-accent text-accent-foreground",
                    )
                  }
                >
                  <Icon className="size-3.5" />
                  {item.label}
                </NavLink>
              );
            })}
          </div>
        }
        right={
          <div className="min-h-0 flex-1 overflow-auto">
            <Outlet />
          </div>
        }
      />
    </div>
  );
}
