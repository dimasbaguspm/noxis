import { useState } from "react";
import { BadgeCheck, PlugZap } from "lucide-react";
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

const EMPTY: Connection = {
  name: "",
  driver: "postgres",
  host: "",
  port: 5432,
  user: "",
  password: "",
  database: "",
};

const REDIS_EMPTY: Connection = { ...EMPTY, driver: "redis", port: 6379, database: "0" };

// ConnectionDialog adds (or edits) a connection. Nothing is kept unless the
// instance answers: the probe runs first, and a failed probe removes the
// registry entry it just wrote. The family decides which namespace it talks
// to and which fields make sense.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
  family = "sql",
  onSaved,
}: {
  trigger?: string;
  initial?: Connection;
  family?: "sql" | "redis";
  onSaved?: () => void | Promise<void>;
}) {
  const redis = family === "redis";
  const blank = redis ? REDIS_EMPTY : EMPTY;
  const client = redis ? api.redis : api.connections;
  const { reload, setActive } = useApp();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<Connection>(initial ?? blank);
  const [verified, setVerified] = useState(false);

  const connect = useAction(async () => {
    const saved = await client.save({ ...form, port: Number(form.port) || (redis ? 6379 : 5432) });
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
    if (!redis) setActive(name);
    setOpen(false);
    setForm(blank);
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
          <DialogTitle>Connection</DialogTitle>
          <DialogDescription>
            {redis
              ? "Point noXis at a Redis or Valkey server that already exists. It is verified before it is saved."
              : "Point noXis at a database that already exists. It is verified before it is saved."}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Name" value={form.name} onChange={(v) => update("name", v)} placeholder="prod-readonly" />
          <div className="grid gap-1.5">
            <Label>Driver</Label>
            {redis ? (
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
            placeholder={redis ? "cache.internal" : "db.internal"}
          />
          <Field label="Port" value={String(form.port)} onChange={(v) => update("port", Number(v))} type="number" />
          <Field
            label={redis ? "User (ACL, optional)" : "User"}
            value={form.user ?? ""}
            onChange={(v) => update("user", v)}
            placeholder={redis ? "default" : "postgres"}
          />
          <Field
            label="Password"
            value={form.password ?? ""}
            onChange={(v) => update("password", v)}
            type="password"
            placeholder={initial ? "unchanged" : ""}
          />
          <Field
            label={redis ? "DB index" : "Database"}
            value={form.database}
            onChange={(v) => update("database", v)}
            placeholder={redis ? "0" : "app"}
          />
        </div>

        <DialogFooter className="items-center gap-2">
          {verified && (
            <span className="mr-auto flex items-center gap-1 text-xs text-emerald-500">
              <BadgeCheck className="size-3.5" /> verified
            </span>
          )}
          <Button onClick={() => void connect.run()} disabled={connect.loading}>
            <PlugZap /> Save (verifies first)
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
