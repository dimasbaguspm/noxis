import { useState } from "react";
import { BadgeCheck, Boxes, ChevronLeft, Database, Leaf, PlugZap } from "lucide-react";
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
import { FAMILY, type FamilyKey } from "@/lib/families";
import { api, type Connection } from "@/lib/api";
import { cn } from "@/lib/utils";

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
const MONGO_EMPTY: Connection = { ...SQL_EMPTY, driver: "mongo", port: 27017, database: "admin" };

const BLANK: Record<FamilyKey, Connection> = {
  [FAMILY.SQL]: SQL_EMPTY,
  [FAMILY.MEMCACHE]: MEMCACHE_EMPTY,
  [FAMILY.MONGODB]: MONGO_EMPTY,
};

// port follows the driver; the list itself comes from the server catalog.
const DRIVER_PORT: Record<string, number> = {
  postgres: 5432,
  mysql: 3306,
  redis: 6379,
  valkey: 6379,
  mongo: 27017,
};

const defaultPort = (driver: string) => DRIVER_PORT[driver] ?? 5432;

// ConnectionDialog registers an instance. Nothing is kept unless it answers:
// the probe runs first, and a failed probe removes the registry entry it just
// wrote. With `wizard` it starts one step earlier, on the family choice, so
// the console has a single "new connection" entry point.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
  family = FAMILY.SQL,
  wizard = false,
}: {
  trigger?: string;
  initial?: Connection;
  family?: FamilyKey;
  wizard?: boolean;
}) {
  const { reload, setActive, drivers } = useApp();
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<FamilyKey>(family);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [form, setForm] = useState<Connection>(initial ?? BLANK[family]);
  const [verified, setVerified] = useState(false);

  const cache = kind === FAMILY.MEMCACHE;
  const mongo = kind === FAMILY.MONGODB;
  const client = api.family(kind);

  // Saving writes to the shared registry, so refresh the family: the dashboard
  // table reads it from the provider, and a new server must show up there
  // without a reload.
  async function refresh() {
    await reload(kind);
  }

  function choose(next: FamilyKey) {
    setKind(next);
    setForm(BLANK[next]);
    setVerified(false);
    setStep(2);
  }

  const connect = useAction(async () => {
    const saved = await client.save({ ...form, port: Number(form.port) || defaultPort(form.driver) });
    const name = saved.name ?? "";
    try {
      await client.test(name);
    } catch (err) {
      await client.remove(name).catch(() => undefined);
      await refresh();
      setVerified(false);
      throw new Error(`connection refused — not saved: ${err instanceof Error ? err.message : String(err)}`);
    }
    setVerified(true);
    toast.success(`${name}: verified and saved`);
    await refresh();
    if (!cache) setActive(kind, name);
    setOpen(false);
    setForm(BLANK[kind]);
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
              onClick={() => choose(FAMILY.SQL)}
            />
            <Choice
              icon={<Boxes className="size-4" />}
              title="Memory cache"
              detail="Redis or Valkey. Browse keys, inspect values, remove keys."
              onClick={() => choose(FAMILY.MEMCACHE)}
            />
            <Choice
              icon={<Leaf className="size-4" />}
              title="Document database"
              detail="MongoDB. Explore databases and collections, run CRUD requests."
              onClick={() => choose(FAMILY.MONGODB)}
            />
          </div>
        ) : (
          <>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field label="Name" value={form.name} onChange={(v) => update("name", v)} placeholder="prod-readonly" />
              <div className="grid gap-1.5">
                <Label>Driver</Label>
                {mongo ? (
                  <Input value="mongo" readOnly disabled />
                ) : (
                  <Select
                    value={form.driver}
                    onValueChange={(v) => {
                      const driver = String(v);
                      setForm((prev) => ({ ...prev, driver, port: defaultPort(driver) }));
                      setVerified(false);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {drivers
                        .filter((d) => d.family === kind)
                        .map((d) => (
                          <SelectItem key={d.driver} value={d.driver}>
                            {d.label}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
              <Field
                label="Host"
                value={form.host}
                onChange={(v) => update("host", v)}
                placeholder={cache ? "cache.internal" : mongo ? "mongo.internal" : "db.internal"}
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
                placeholder={cache ? "default" : mongo ? "root" : "postgres"}
              />
              <Field
                label="Password"
                value={form.password ?? ""}
                onChange={(v) => update("password", v)}
                type="password"
                placeholder={initial ? "unchanged" : ""}
              />
              <Field
                label={cache ? "DB index" : mongo ? "Database (auth)" : "Database"}
                value={form.database}
                onChange={(v) => update("database", v)}
                placeholder={cache ? "0" : mongo ? "admin" : "app"}
              />
            </div>

            <p className="text-xs text-muted-foreground">
              Driver, host, database and user are fixed once saved. The password can be rotated later from the instance page.
            </p>

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
                <PlugZap /> Save
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
