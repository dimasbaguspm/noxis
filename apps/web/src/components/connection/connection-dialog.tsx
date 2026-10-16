import { useState } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { BadgeCheck, ChevronLeft, ChevronRight, PlugZap } from "lucide-react";
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { SingleSelect } from "@/components/ui/single-select";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSetActiveConnection } from "@/hooks/use-connections";
import { useDriversQuery } from "@/hooks/use-config";
import { queryClient } from "@/lib/query-client";
import { FAMILY, type FamilyKey } from "@/lib/families";
import { api, type Connection } from "@/lib/api";

// connectionSchema is the client-side contract for the registration form;
// the server still enforces driver specifics on probe.
const connectionSchema = z.object({
  name: z.string().trim().min(1, "name is required"),
  host: z.string().trim().min(1, "host is required"),
  port: z.coerce.number().int().min(1, "port must be between 1 and 65535").max(65535, "port must be between 1 and 65535"),
  user: z.string().trim().min(1, "user is required"),
  password: z.string().optional(),
  database: z.string().trim().min(1, "database is required"),
  driver: z.string().optional(),
});
type ConnectionForm = z.infer<typeof connectionSchema>;

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
  const { register, handleSubmit, watch, setValue, reset, formState: { errors } } = useForm<ConnectionForm>({
    defaultValues: (initial ?? BLANK[family]) as unknown as ConnectionForm,
    // z.coerce makes the schema's input type differ from its output type;
    // the resolver is cast so the form values are the coerced shape.
    resolver: zodResolver(connectionSchema) as unknown as Resolver<ConnectionForm>,
  });
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<FamilyKey>(family);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [verified, setVerified] = useState(false);
  const [busy, setBusy] = useState(false);

  // closing never keeps a half-filled form: the next visit starts over at the
  // kind choice with pristine values for the dialog's own family
  function onOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      setKind(family);
      reset(BLANK[family]);
      setVerified(false);
      setStep(wizard && !initial ? 1 : 2);
    }
  }

  const cache = kind === FAMILY.MEMCACHE;
  const mongo = kind === FAMILY.MONGODB;
  const client = api.family(kind);

  // any edit makes a previous "verified" stale until the probe runs again
  const field = (key: "name" | "host" | "port" | "user" | "password" | "database") => {
    const r = register(key);
    return {
      ...r,
      "aria-invalid": Boolean(errors[key]),
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
    const form: Connection = { ...values, family: kind } as unknown as Connection;
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
    <Sheet open={open} onOpenChange={onOpenChange}>
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
          <>
            <SheetBody>
              <SingleSelect
                name="kind"
                value={kind}
                onChange={(value) => choose(value)}
                options={[
                  {
                    value: FAMILY.SQL,
                    label: "SQL database",
                    detail: "Postgres, MySQL or MariaDB. Read and change data through the validated workflow.",
                  },
                  {
                    value: FAMILY.MEMCACHE,
                    label: "Memory cache",
                    detail: "Redis or Valkey. Browse keys, inspect values, remove keys.",
                  },
                  {
                    value: FAMILY.MONGODB,
                    label: "Document database",
                    detail: "MongoDB. Explore databases and collections, run CRUD requests.",
                  },
                ]}
              />
            </SheetBody>
            <SheetFooter>
              <Button className="ml-auto" onClick={() => setStep(2)}>
                Next <ChevronRight className="size-4" />
              </Button>
            </SheetFooter>
          </>
        ) : (
          <>
            <SheetBody>
              <form id="connection-save" onSubmit={onValid} className="grid gap-4">
              <Field>
                <FieldLabel htmlFor="conn-name">Name</FieldLabel>
                <Input id="conn-name" placeholder="prod-readonly" {...field("name")} />
                <FieldError>{errors.name?.message}</FieldError>
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
                <FieldError>{errors.host?.message}</FieldError>
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-port">Port</FieldLabel>
                <Input id="conn-port" type="number" {...field("port")} />
                <FieldError>{errors.port?.message}</FieldError>
              </Field>
              <Field>
                <FieldLabel htmlFor="conn-user">{cache ? "User (ACL, optional)" : "User"}</FieldLabel>
                <Input
                  id="conn-user"
                  placeholder={cache ? "default" : mongo ? "root" : "postgres"}
                  {...field("user")}
                />
                <FieldError>{errors.user?.message}</FieldError>
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
              <Field>
                <FieldLabel htmlFor="conn-database">{cache ? "DB index" : mongo ? "Database (auth)" : "Database"}</FieldLabel>
                <Input
                  id="conn-database"
                  placeholder={cache ? "0" : mongo ? "admin" : "app"}
                  {...field("database")}
                />
                <FieldError>{errors.database?.message}</FieldError>
              </Field>

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
