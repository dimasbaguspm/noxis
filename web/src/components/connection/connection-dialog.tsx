import { useState } from "react";
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

// ConnectionDialog adds (or edits) a connection. Editing omits the password
// to keep the stored one.
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

  const save = useAction(async () => {
    const saved = await api.connections.save({ ...form, port: Number(form.port) || 5432 });
    toast.success(`saved ${saved.name}`);
    setOpen(false);
    setForm(EMPTY);
    await reload();
    setActive(saved.name ?? "");
  });

  function update<K extends keyof Connection>(key: K, value: Connection[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="secondary" />}>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Connection</DialogTitle>
          <DialogDescription>Point noxis at a database that already exists.</DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Name" value={form.name} onChange={(v) => update("name", v)} placeholder="prod-readonly" />
          <div className="grid gap-1.5">
            <Label>Driver</Label>
            <Select
              value={form.driver}
              onValueChange={(v) => {
                const driver = String(v);
                update("driver", driver);
                update("port", driver === "mysql" ? 3306 : 5432);
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

        <DialogFooter>
          <Button onClick={() => void save.run()} disabled={save.loading}>
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
