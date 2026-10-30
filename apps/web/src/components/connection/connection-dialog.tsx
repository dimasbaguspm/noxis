import { useConnectionWizard } from "@/hooks/use-connection-wizard";
import { useForm, type Resolver } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  BadgeCheck,
  ChevronLeft,
  ChevronRight,
  ClipboardPaste,
  Eye,
  EyeOff,
  PlugZap,
  X,
} from "lucide-react";
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
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { SingleSelect } from "@/components/ui/single-select";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSetActiveConnection } from "@/hooks/use-connections";
import { useDriversQuery } from "@/hooks/use-config";
import { queryClient, queryKeys } from "@/lib/query-client";
import { CAPABILITY, capabilitiesOf, type Capability } from "@/lib/console-nav";
import { api, type Connection } from "@/lib/api";
import { parseDsn } from "@/lib/dsn";

// connectionSchema validates shared fields; the server probe enforces driver-specific host or file-path requirements.
const FILE_DRIVERS = new Set(["sqlite"]);
const isFileDriver = (driver?: string) => FILE_DRIVERS.has(driver ?? "");

const connectionSchema = z
  .object({
    name: z.string().trim().min(1, "Name is required"),
    host: z.string().trim().optional(),
    port: z.coerce.number().int().min(0).max(65535, "Port must be between 1 and 65535").optional(),
    user: z.string().trim().optional(),
    password: z.string().optional(),
    database: z.string().trim().min(1, "Database is required"),
    params: z.record(z.string(), z.string()).optional(),
    driver: z.string().optional(),
  })
  .superRefine((v, ctx) => {
    if (isFileDriver(v.driver)) return;
    if (!v.host) ctx.addIssue({ code: "custom", path: ["host"], message: "host is required" });
    if (!v.port)
      ctx.addIssue({ code: "custom", path: ["port"], message: "Port must be between 1 and 65535" });
    if (!v.user) ctx.addIssue({ code: "custom", path: ["user"], message: "user is required" });
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

const BLANK: Record<Capability, Connection> = {
  [CAPABILITY.SQL]: SQL_EMPTY,
  [CAPABILITY.KEYS]: MEMCACHE_EMPTY,
  [CAPABILITY.DOCUMENTS]: MONGO_EMPTY,
};

// port follows the driver; the list itself comes from the server catalog.
const DRIVER_PORT: Record<string, number> = {
  postgres: 5432,
  mysql: 3306,
  redis: 6379,
  valkey: 6379,
  mongo: 27017,
};

const defaultPort = (driver: string) => (isFileDriver(driver) ? 0 : (DRIVER_PORT[driver] ?? 5432));

// ConnectionDialog keeps an instance only after its connection probe succeeds.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
  capability = CAPABILITY.SQL,
  wizard = false,
}: {
  trigger?: string;
  initial?: Connection;
  capability?: Capability;
  wizard?: boolean;
}) {
  const setActive = useSetActiveConnection();
  const { data: drivers = [] } = useDriversQuery();
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<ConnectionForm>({
    defaultValues: (initial ?? BLANK[capability]) as unknown as ConnectionForm,
    // Cast the resolver because z.coerce gives form inputs and parsed output different types.
    resolver: zodResolver(connectionSchema) as unknown as Resolver<ConnectionForm>,
  });
  const {
    open,
    kind,
    step,
    verified,
    busy,
    dsn,
    dsnError,
    showPassword,
    cache,
    mongo,
    setKind,
    setOpen,
    setStep,
    setVerified,
    setBusy,
    setDsn,
    setDsnError,
    onOpenChange,
    choose,
    nextStep,
    backStep,
    togglePassword,
    clearDsn,
    pasteDsn,
  } = useConnectionWizard({ capability, wizard, initial, blank: BLANK, reset });

  const fileDriver = isFileDriver(watch("driver"));
  // the engine path is the driver the form holds, so switching driver after picking a capability changes where the save goes
  const driver = watch("driver") || BLANK[kind].driver;
  const client = api.connections(driver);

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

  // refresh invalidates the cached instance list after a connection write.
  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: queryKeys.instances.all });
  }

  // applyDsn fills fields from a pasted DSN and switches the picker to the capability its driver declares.
  function applyDsn() {
    if (!dsn.trim()) return;
    const parsed = parseDsn(dsn);
    if (!parsed.ok) {
      setDsnError(parsed.error);
      return;
    }
    const v = parsed.value;
    const info = drivers.find((d) => d.driver === v.driver);
    if (!info) {
      setDsnError(`unknown driver "${v.driver}"`);
      return;
    }
    setDsnError(null);
    const next = capabilitiesOf(drivers, v.driver)[0];
    if (next && next !== kind) setKind(next);
    setValue("driver", v.driver);
    setValue("host", v.host);
    setValue("port", v.port);
    setValue("user", v.user);
    setValue("password", v.password);
    setValue("database", v.database);
    setValue("params", v.params);
    setVerified(false);
  }

  const onValid = handleSubmit(async (values) => {
    const form: Connection = { ...values, driver: values.driver || driver } as Connection;
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
        throw new Error(
          `connection refused — not saved: ${err instanceof Error ? err.message : String(err)}`,
        );
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
                    value: CAPABILITY.SQL,
                    label: "SQL database",
                    detail:
                      "Postgres, MySQL or MariaDB. Read and change data through the validated workflow.",
                  },
                  {
                    value: CAPABILITY.KEYS,
                    label: "Memory cache",
                    detail: "Redis or Valkey. Browse keys, inspect values, remove keys.",
                  },
                  {
                    value: CAPABILITY.DOCUMENTS,
                    label: "Document database",
                    detail: "MongoDB. Explore databases and collections, run CRUD requests.",
                  },
                ]}
              />
            </SheetBody>
            <SheetFooter>
              <Button className="ml-auto" onClick={nextStep}>
                Next <ChevronRight className="size-4" />
              </Button>
            </SheetFooter>
          </>
        ) : (
          <>
            <SheetBody>
              <form id="connection-save" onSubmit={onValid} className="grid gap-4">
                <Field>
                  <FieldLabel htmlFor="conn-dsn">Connection string (optional)</FieldLabel>
                  <InputGroup>
                    <InputGroupInput
                      id="conn-dsn"
                      placeholder="postgresql://user:pass@host:5432/db"
                      value={dsn}
                      onChange={(e) => {
                        setDsn(e.target.value);
                        setDsnError(null);
                      }}
                      onBlur={applyDsn}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          applyDsn();
                        }
                      }}
                    />
                    <InputGroupAddon align="inline-end">
                      {dsn && (
                        <InputGroupButton aria-label="Clear connection string" onClick={clearDsn}>
                          <X />
                        </InputGroupButton>
                      )}
                      <InputGroupButton aria-label="Paste connection string" onClick={pasteDsn}>
                        <ClipboardPaste />
                      </InputGroupButton>
                    </InputGroupAddon>
                  </InputGroup>
                  <FieldDescription>Paste a URL to fill the fields below.</FieldDescription>
                  <FieldError>{dsnError}</FieldError>
                </Field>
                <Field>
                  <FieldLabel htmlFor="conn-name">Name</FieldLabel>
                  <Input id="conn-name" placeholder="prod-readonly" {...field("name")} />
                  <FieldError errors={[errors.name]} />
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
                        if (isFileDriver(driver)) {
                          setValue("host", "");
                          setValue("user", "");
                        }
                        setVerified(false);
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Pick a driver" />
                      </SelectTrigger>
                      <SelectContent>
                        {drivers
                          .filter((d) => d.capabilities.includes(kind))
                          .map((d) => (
                            <SelectItem key={d.driver} value={d.driver}>
                              {d.label}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  )}
                </Field>
                {!fileDriver && (
                  <>
                    <Field>
                      <FieldLabel htmlFor="conn-host">Host</FieldLabel>
                      <Input
                        id="conn-host"
                        placeholder={
                          cache ? "cache.internal" : mongo ? "mongo.internal" : "db.internal"
                        }
                        {...field("host")}
                      />
                      <FieldError errors={[errors.host]} />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="conn-port">Port</FieldLabel>
                      <Input id="conn-port" type="number" {...field("port")} />
                      <FieldError errors={[errors.port]} />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="conn-user">
                        {cache ? "User (ACL, optional)" : "User"}
                      </FieldLabel>
                      <Input
                        id="conn-user"
                        placeholder={cache ? "default" : mongo ? "root" : "postgres"}
                        {...field("user")}
                      />
                      <FieldError errors={[errors.user]} />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="conn-password">Password</FieldLabel>
                      <InputGroup>
                        <InputGroupInput
                          id="conn-password"
                          type={showPassword ? "text" : "password"}
                          placeholder={initial ? "Unchanged" : ""}
                          {...field("password")}
                        />
                        <InputGroupAddon align="inline-end">
                          <InputGroupButton
                            aria-label={showPassword ? "Hide password" : "Show password"}
                            onClick={togglePassword}
                          >
                            {showPassword ? <EyeOff /> : <Eye />}
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </Field>
                  </>
                )}
                <Field>
                  <FieldLabel htmlFor="conn-database">
                    {fileDriver
                      ? "File path"
                      : cache
                        ? "DB index"
                        : mongo
                          ? "Database (auth)"
                          : "Database"}
                  </FieldLabel>
                  <Input
                    id="conn-database"
                    placeholder={
                      fileDriver ? "/data/app.db" : cache ? "0" : mongo ? "admin" : "app"
                    }
                    {...field("database")}
                  />
                  <FieldError errors={[errors.database]} />
                </Field>
              </form>
            </SheetBody>

            <SheetFooter className="items-center gap-2">
              {wizard && (
                <Button variant="ghost" className="mr-auto" onClick={backStep}>
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
