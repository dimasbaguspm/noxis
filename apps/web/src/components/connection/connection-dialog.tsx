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
import { useDriversQuery } from "@/hooks/use-config";
import { queryClient, queryKeys } from "@/lib/query-client";
import { CAPABILITY, capabilitiesOf } from "@/lib/console-nav";
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

// port follows the driver; the list itself comes from the server catalog.
const DRIVER_PORT: Record<string, number> = {
  postgres: 5432,
  mysql: 3306,
  mariadb: 3306,
  redis: 6379,
  valkey: 6379,
  mongo: 27017,
};

const CACHE_DRIVERS = new Set(["redis", "valkey"]);

// draftFor is the empty draft of one driver: its port, and the database it usually listens on.
function draftFor(driver: string): Connection {
  return {
    name: "",
    driver,
    host: "",
    port: isFileDriver(driver) ? 0 : (DRIVER_PORT[driver] ?? 5432),
    user: "",
    password: "",
    database: CACHE_DRIVERS.has(driver) ? "0" : driver === "mongo" ? "admin" : "",
  };
}

// ConnectionDialog keeps an instance only after its connection probe succeeds.
export function ConnectionDialog({
  trigger = "New connection",
  initial,
  driver: defaultDriver = "postgres",
  wizard = false,
}: {
  trigger?: string;
  initial?: Connection;
  driver?: string;
  wizard?: boolean;
}) {
  const { data: drivers = [] } = useDriversQuery();
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<ConnectionForm>({
    defaultValues: (initial ?? draftFor(defaultDriver)) as unknown as ConnectionForm,
    // Cast the resolver because z.coerce gives form inputs and parsed output different types.
    resolver: zodResolver(connectionSchema) as unknown as Resolver<ConnectionForm>,
  });
  const {
    open,
    step,
    verified,
    busy,
    dsn,
    dsnError,
    showPassword,
    setOpen,
    setStep,
    setVerified,
    setBusy,
    setDsn,
    setDsnError,
    onOpenChange,
    nextStep,
    backStep,
    togglePassword,
    clearDsn,
    pasteDsn,
  } = useConnectionWizard({ wizard, initial, blank: draftFor(defaultDriver), reset });

  const fileDriver = isFileDriver(watch("driver"));
  // the engine path is the driver the form holds; its capability decides which fields and placeholders it shows
  const driver = watch("driver") || defaultDriver;
  const kind = capabilitiesOf(drivers, driver)[0];
  const cache = kind === CAPABILITY.KEYS;
  const mongo = kind === CAPABILITY.DOCUMENTS;

  // choosing a driver starts from its own draft, so the port and the database are already right
  function chooseDriver(next: string) {
    setValue("driver", next);
    reset(draftFor(next));
    setVerified(false);
    setStep(2);
  }
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

  // applyDsn fills the fields from a pasted DSN and moves the form to the driver it names.
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
    setDsnError(null);
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
      setOpen(false);
      reset(draftFor(watch("driver") || defaultDriver));
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
          <SheetTitle>{step === 1 ? "Which driver?" : "Connection"}</SheetTitle>
          <SheetDescription>
            {step === 1
              ? "Pick the engine. The next step asks for its details."
              : "Point noXis at an instance that already exists. It is verified before it is saved."}
          </SheetDescription>
        </SheetHeader>

        {step === 1 ? (
          <>
            <SheetBody>
              <SingleSelect
                name="driver"
                value={driver}
                onChange={(value) => chooseDriver(String(value))}
                options={drivers.map((d) => ({
                  value: d.driver,
                  label: d.label,
                  detail: d.driver,
                }))}
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
                        setValue("port", isFileDriver(driver) ? 0 : (DRIVER_PORT[driver] ?? 5432));
                        if (isFileDriver(driver)) {
                          setValue("host", "");
                          setValue("user", "");
                        }
                        setVerified(false);
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue>
                          {(value) =>
                            drivers.find((d) => d.driver === value)?.label ?? String(value)
                          }
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {drivers.map((d) => (
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
