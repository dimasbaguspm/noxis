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

// ConnectionDialog adds (or edits) a connection. Nothing is kept unless the
// instance answers: the probe runs first, and a failed probe removes the
// registry entry it just wrote.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
}: {
  trigger?: string;
  initial?: Connection;
}) {
  const { reload, setActive } = useApp();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<Connection>(initial ?? EMPTY);
  const [verified, setVerified] = useState(false);

  const connect = useAction(async (close: boolean) => {
    const saved = await api.connections.save({ ...form, port: Number(form.port) || 5432 });
    const name = saved.name ?? "";
    try {
      await api.connections.test(name);
    } catch (err) {
      await api.connections.remove(name).catch(() => undefined);
      await reload();
      setVerified(false);
      throw new Error(`connection refused — not saved: ${err instanceof Error ? err.message : String(err)}`);
    }
    setVerified(true);
    toast.success(`${name}: verified and saved`);
    await reload();
    setActive(name);
    if (close) {
      setOpen(false);
      setForm(EMPTY);
    }
  });

  function update<K extends keyof Connection>(key: K, value: Connection[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
    setVerified(false);
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="secondary" />}>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Connection</DialogTitle>
          <DialogDescription>
            Point noxis at a database that already exists — it is verified before it is saved.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Name" value={form.name} onChange={(v) => update("name", v)} placeholder="prod-readonly" />
          <div className="grid gap-1.5">
            <Label>Driver</Label>
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
          </div>
          <Field label="Host" value={form.host} onChange={(v) => update("host", v)} placeholder="db.internal" />
          <Field label="Port" value={String(form.port)} onChange={(v) => update("port", Number(v))} type="number" />
          <Field label="User" value={form.user} onChange={(v) => update("user", v)} placeholder="postgres" />
          <Field
            label="Password"
            value={form.password ?? ""}
            onChange={(v) => update("password", v)}
            type="password"
            placeholder={initial ? "unchanged" : ""}
          />
          <Field label="Database" value={form.database} onChange={(v) => update("database", v)} placeholder="app" />
        </div>

        <DialogFooter className="items-center gap-2">
          {verified && (
            <span className="mr-auto flex items-center gap-1 text-xs text-emerald-500">
              <BadgeCheck className="size-3.5" /> verified
            </span>
          )}
          <Button variant="outline" onClick={() => void connect.run(false)} disabled={connect.loading}>
            <PlugZap /> Test
          </Button>
          <Button onClick={() => void connect.run(true)} disabled={connect.loading}>
            Save
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
