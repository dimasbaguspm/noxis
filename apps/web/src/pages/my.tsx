import { NavLink, Outlet } from "react-router-dom";
import { KeyRound, Settings, UserRound } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
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
    <div className="flex h-full flex-col">
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 min-w-0 flex-1">
        <ResizablePanel defaultSize={20} minSize={16}>
          <Card className="mr-2 flex h-full min-h-0 flex-col">
            <CardHeader>
              <CardTitle className="text-base">My account</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-1">
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
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={80} minSize={40} className="min-w-0">
          <div className="ml-2 h-full overflow-auto">
            <Outlet />
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
