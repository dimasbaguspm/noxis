import { Database, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useAction } from "@/hooks/use-action";
import { useApp } from "@/app/app-provider";
import { api } from "@/lib/api";

// ConnectionList shows the registered instances with test/delete actions.
export function ConnectionList() {
  const { connections, active, setActive, reload } = useApp();

  const test = useAction(async (name: string) => {
    await api.connections.test(name);
    toast.success(`${name}: reachable`);
  });
  const remove = useAction(async (name: string) => {
    await api.connections.remove(name);
    toast.success(`deleted ${name}`);
    await reload();
  });

  if (connections.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>No connections yet</CardTitle>
          <CardDescription>Add one to start browsing and querying.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {connections.map((c) => (
        <Card key={c.name} className={c.name === active ? "border-primary/40" : undefined}>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Database className="size-4 text-muted-foreground" />
              {c.name}
              {c.name === active && <Badge variant="secondary">active</Badge>}
            </CardTitle>
            <CardDescription>
              {c.driver} · {c.user}@{c.host}:{c.port}/{c.database}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex gap-2">
            <Button size="sm" variant="secondary" onClick={() => setActive(c.name ?? "")}>
              Use
            </Button>
            <Button size="sm" variant="outline" onClick={() => void test.run(c.name ?? "")}>
              <Zap /> Test
            </Button>
            <Button size="sm" variant="ghost" onClick={() => void remove.run(c.name ?? "")}>
              <Trash2 /> Delete
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
