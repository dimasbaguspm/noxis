import { useState } from "react";
import { BadgeCheck, Boxes, ChevronLeft, Database, PlugZap } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAction } from "@/hooks/use-action";
import { useApp } from "@/app/app-provider";
import { api, type Connection } from "@/lib/api";
import { cn } from "@/lib/utils";

type Family = "sql" | "memcache";

const SQL_EMPTY: Connection = {
  name: "",
  driver: "postgres",
  host: "",
  port: 5432,
  user: "",
  password: "",
  database: "",
};

const MEMCACHE_EMPTY: Connection = { ...SQL_EMPTY, driver: "redis", port: 6379, database: "0" };

// ConnectionDialog registers an instance. Nothing is kept unless it answers:
// the probe runs first, and a failed probe removes the registry entry it just
// wrote. With `wizard` it starts one step earlier, on the family choice, so
// the console has a single "new connection" entry point.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
  family = "sql",
  wizard = false,
  onSaved,
}: {
  trigger?: string;
  initial?: Connection;
  family?: Family;
  wizard?: boolean;
  onSaved?: () => void | Promise<void>;
}) {
  const { reload, setActive } = useApp();
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<Family>(family);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [form, setForm] = useState<Connection>(initial ?? (family === "memcache" ? MEMCACHE_EMPTY : SQL_EMPTY));
  const [verified, setVerified] = useState(false);

  const cache = kind === "memcache";
  const client = cache ? api.memcache : api.connections;

  function choose(next: Family) {
    setKind(next);
    setForm(next === "memcache" ? MEMCACHE_EMPTY : SQL_EMPTY);
    setVerified(false);
    setStep(2);
  }

  const connect = useAction(async () => {
    const saved = await client.save({ ...form, port: Number(form.port) || (cache ? 6379 : 5432) });
    const name = saved.name ?? "";
    try {
      await client.test(name);
    } catch (err) {
      await client.remove(name).catch(() => undefined);
      await reload();
      await onSaved?.();
      setVerified(false);
      throw new Error(`connection refused — not saved: ${err instanceof Error ? err.message : String(err)}`);
    }
    setVerified(true);
    toast.success(`${name}: verified and saved`);
    await reload();
    await onSaved?.();
    if (!cache) setActive(name);
    setOpen(false);
    setForm(cache ? MEMCACHE_EMPTY : SQL_EMPTY);
    setStep(wizard ? 1 : 2);
  });

  function update<K extends keyof Connection>(key: K, value: Connection[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
    setVerified(false);
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{step === 1 ? "What are you connecting to?" : "Connection"}</DialogTitle>
          <DialogDescription>
            {step === 1
              ? "Pick the kind of instance. The next step asks for its details."
              : "Point noXis at an instance that already exists. It is verified before it is saved."}
          </DialogDescription>
        </DialogHeader>

        {step === 1 ? (
          <div className="grid gap-3">
            <Choice
              icon={<Database className="size-4" />}
              title="SQL database"
              detail="Postgres, MySQL or MariaDB. Read and change data through the validated workflow."
              onClick={() => choose("sql")}
            />
            <Choice
              icon={<Boxes className="size-4" />}
              title="Memory cache"
              detail="Redis or Valkey. Browse keys, inspect values, watch INFO metrics."
              onClick={() => choose("memcache")}
            />
          </div>
        ) : (
          <>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field label="Name" value={form.name} onChange={(v) => update("name", v)} placeholder="prod-readonly" />
              <div className="grid gap-1.5">
                <Label>Driver</Label>
                {cache ? (
                  <Select
                    value={form.driver}
                    onValueChange={(v) => {
                      update("driver", String(v));
                      update("port", 6379);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="redis">redis</SelectItem>
                      <SelectItem value="valkey">valkey (redis-compatible)</SelectItem>
                    </SelectContent>
                  </Select>
                ) : (
                  <Select
                    value={form.driver}
                    onValueChange={(v) => {
                      const driver = String(v);
                      setForm((prev) => ({ ...prev, driver, port: driver === "mysql" ? 3306 : 5432 }));
                      setVerified(false);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="postgres">postgres</SelectItem>
                      <SelectItem value="mysql">mysql / mariadb</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              </div>
              <Field
                label="Host"
                value={form.host}
                onChange={(v) => update("host", v)}
                placeholder={cache ? "cache.internal" : "db.internal"}
              />
              <Field
                label="Port"
                value={String(form.port)}
                onChange={(v) => update("port", Number(v))}
                type="number"
              />
              <Field
                label={cache ? "User (ACL, optional)" : "User"}
                value={form.user ?? ""}
                onChange={(v) => update("user", v)}
                placeholder={cache ? "default" : "postgres"}
              />
              <Field
                label="Password"
                value={form.password ?? ""}
                onChange={(v) => update("password", v)}
                type="password"
                placeholder={initial ? "unchanged" : ""}
              />
              <Field
                label={cache ? "DB index" : "Database"}
                value={form.database}
                onChange={(v) => update("database", v)}
                placeholder={cache ? "0" : "app"}
              />
            </div>

            <DialogFooter className="items-center gap-2">
              {wizard && (
                <Button variant="ghost" className="mr-auto" onClick={() => setStep(1)}>
                  <ChevronLeft /> Back
                </Button>
              )}
              {verified && (
                <span className="mr-auto flex items-center gap-1 text-xs text-emerald-500">
                  <BadgeCheck className="size-3.5" /> verified
                </span>
              )}
              <Button onClick={() => void connect.run()} disabled={connect.loading}>
                <PlugZap /> Save (verifies first)
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Choice({
  icon,
  title,
  detail,
  onClick,
}: {
  icon: React.ReactNode;
  title: string;
  detail: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-start gap-3 rounded-md border border-border p-3 text-left",
        "transition-colors hover:border-primary/40 hover:bg-accent/50",
      )}
    >
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <span className="grid gap-0.5">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-muted-foreground">{detail}</span>
      </span>
    </button>
  );
}

function Field({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
}) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <Input value={value} type={type} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} />
    </div>
  );
}
