import { useState } from "react";
import { useForm } from "react-hook-form";
import { BadgeCheck, Boxes, ChevronLeft, Database, Leaf, PlugZap } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetBody,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Field, FieldLabel, FieldRow } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSetActiveConnection } from "@/hooks/use-connections";
import { useDriversQuery } from "@/hooks/use-config";
import { queryClient } from "@/lib/query-client";
import { FAMILY, type FamilyKey } from "@/lib/families";
import { api, type Connection } from "@/lib/api";

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
// the console has a single "new connection" entry point. react-hook-form owns
// the instance fields; the driver select writes the driver and its port.
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
  const setActive = useSetActiveConnection();
  const { data: drivers = [] } = useDriversQuery();
  const { register, handleSubmit, watch, setValue, reset } = useForm<Connection>({
    defaultValues: initial ?? BLANK[family],
  });
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<FamilyKey>(family);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [verified, setVerified] = useState(false);
  const [busy, setBusy] = useState(false);

  const cache = kind === FAMILY.MEMCACHE;
  const mongo = kind === FAMILY.MONGODB;
  const client = api.family(kind);

  // any edit makes a previous "verified" stale until the probe runs again
  const field = (key: "name" | "host" | "port" | "user" | "password" | "database") => {
    const r = register(key);
    return {
      ...r,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) => {
        void r.onChange(e);
        setVerified(false);
      },
    };
  };

  // Saving writes a connection; the instances query already caches the list,
  // so invalidating it is all a refresh takes.
  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ["instances"] });
  }

  function choose(next: FamilyKey) {
    setKind(next);
    reset(BLANK[next]);
    setVerified(false);
    setStep(2);
  }

  const onValid = handleSubmit(async (values) => {
    const form: Connection = { ...values, port: Number(values.port) || defaultPort(values.driver) };
    setBusy(true);
    try {
      const saved = await client.save(form);
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
      reset(BLANK[kind]);
      setStep(wizard ? 1 : 2);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  });

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger render={<Button />}>{trigger}</SheetTrigger>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{step === 1 ? "What are you connecting to?" : "Connection"}</SheetTitle>
          <SheetDescription>
            {step === 1
              ? "Pick the kind of instance. The next step asks for its details."
              : "Point noXis at an instance that already exists. It is verified before it is saved."}
          </SheetDescription>
        </SheetHeader>

        {step === 1 ? (
          <SheetBody className="grid gap-3">
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
          </SheetBody>
        ) : (
          <>
            <SheetBody className="grid gap-4">
              <form id="connection-save" onSubmit={onValid} className="grid gap-4">
            <FieldRow>
              <Field>
                <FieldLabel htmlFor="conn-name">Name</FieldLabel>
                <Input id="conn-name" placeholder="prod-readonly" {...field("name")} />
              </Field>
              <Field>
                <FieldLabel>Driver</FieldLabel>
                {mongo ? (
                  <Input value="mongo" readOnly disabled />
                ) : (
                  <Select
                    value={watch("driver")}
                    onValueChange={(v) => {
                      const driver = String(v);
                      setValue("driver", driver);
                      setValue("port", defaultPort(driver));
                      setVerified(false);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="pick a driver" />
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
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-host">Host</FieldLabel>
                <Input
                  id="conn-host"
                  placeholder={cache ? "cache.internal" : mongo ? "mongo.internal" : "db.internal"}
                  {...field("host")}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-port">Port</FieldLabel>
                <Input id="conn-port" type="number" {...field("port")} />
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-user">{cache ? "User (ACL, optional)" : "User"}</FieldLabel>
                <Input
                  id="conn-user"
                  placeholder={cache ? "default" : mongo ? "root" : "postgres"}
                  {...field("user")}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-password">Password</FieldLabel>
                <Input
                  id="conn-password"
                  type="password"
                  placeholder={initial ? "unchanged" : ""}
                  {...field("password")}
                />
              </Field>
            </FieldRow>
              <Field>
                <FieldLabel htmlFor="conn-database">{cache ? "DB index" : mongo ? "Database (auth)" : "Database"}</FieldLabel>
                <Input
                  id="conn-database"
                  placeholder={cache ? "0" : mongo ? "admin" : "app"}
                  {...field("database")}
                />
              </Field>

              <p className="text-xs text-muted-foreground">
                Driver, host, database and user are fixed once saved. The password can be rotated later from the instance page.
              </p>
              </form>
            </SheetBody>

            <SheetFooter className="items-center gap-2">
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
              <Button type="submit" form="connection-save" disabled={busy}>
                <PlugZap /> Save
              </Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
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
    <Button
      variant="outline"
      size="lg"
      onClick={onClick}
      className="h-auto w-full items-start justify-start gap-3 rounded-md p-3"
    >
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <span className="grid gap-0.5">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-muted-foreground">{detail}</span>
      </span>
    </Button>
  );
}