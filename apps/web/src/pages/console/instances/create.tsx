import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useForm, type Resolver } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronLeft, PlugZap } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { SingleSelect } from "@/components/ui/single-select";
import { DriverMark } from "@/components/brand/driver-mark";
import { useAuth } from "@/app/auth-provider";
import { useDriversQuery } from "@/hooks/use-config";
import { useConnectionCreate } from "@/hooks/use-connection-create";
import { api, type Connection } from "@/lib/api";
import {
  connectionSchema,
  draftFor,
  isFileDriver,
  type ConnectionForm,
} from "@/lib/connection-draft";
import { parseDsn } from "@/lib/dsn";
import { queryClient, queryKeys } from "@/lib/query-client";
import { PAGE_ROUTES } from "@/constants/routes";
import { cn } from "@/lib/utils";

const LABELS = ["Driver", "Connection", "Credentials"] as const;

// StepHeader is one rung of the vertical wizard: the number, its name, and the summary of a step already answered.
function StepHeader({
  index,
  step,
  summary,
  onEdit,
}: {
  index: 1 | 2 | 3;
  step: 1 | 2 | 3;
  summary?: string;
  onEdit: () => void;
}) {
  const done = step > index;
  return (
    <div className="flex items-center gap-3">
      <span
        className={cn(
          "flex size-6 shrink-0 items-center justify-center rounded-full border text-xs",
          step === index && "border-foreground font-medium",
          done && "border-foreground/40 text-muted-foreground",
          step < index && "border-border text-muted-foreground/60",
        )}
      >
        {index}
      </span>
      <div className="min-w-0 flex-1">
        <p className={cn("text-sm", step < index && "text-muted-foreground")}>
          {LABELS[index - 1]}
        </p>
        {done && summary && <p className="truncate text-xs text-muted-foreground">{summary}</p>}
      </div>
      {done && (
        <Button size="xs" variant="ghost" onClick={onEdit}>
          Edit
        </Button>
      )}
    </div>
  );
}

// CreateInstancePage is the console's only creation path: three stacked steps, and nothing is saved until the probe
// answers, so a wrong password never writes a row.
export default function CreateInstancePage() {
  const { isSuper } = useAuth();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { data: drivers = [] } = useDriversQuery();
  const { driver, step, setStep, busy, setBusy, steps, choose } = useConnectionCreate(
    params.get("driver") ?? "",
  );
  const [dsn, setDsn] = useState("");
  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<ConnectionForm>({
    defaultValues: draftFor(driver || "postgres") as unknown as ConnectionForm,
    // Cast the resolver because z.coerce gives form inputs and parsed output different types.
    resolver: zodResolver(connectionSchema) as unknown as Resolver<ConnectionForm>,
  });
  const values = watch();
  const fileDriver = isFileDriver(driver);
  const credentials = steps === 3;

  function pickDriver(next: string) {
    reset(draftFor(next) as unknown as ConnectionForm);
    choose(next);
  }

  // applyDsn fills the fields from a pasted connection string and moves the form to the driver it names.
  function applyDsn() {
    if (!dsn.trim()) return;
    const parsed = parseDsn(dsn);
    if (!parsed.ok) {
      toast.error(parsed.error);
      return;
    }
    const v = parsed.value;
    if (!drivers.some((entry) => entry.driver === v.driver)) {
      toast.error(`unknown driver "${v.driver}"`);
      return;
    }
    reset({ ...v } as unknown as ConnectionForm);
    choose(v.driver);
    setDsn("");
  }

  const submit = handleSubmit(async (form) => {
    setBusy(true);
    try {
      const draft = { ...form, driver } as Connection;
      // probe first: a failed probe is a toast, and nothing is written
      await api.connections(driver).probe(draft);
      const saved = await api.connections(driver).save(draft);
      toast.success(`${saved.name}: verified and saved`);
      void queryClient.invalidateQueries({ queryKey: queryKeys.instances.all });
      navigate(PAGE_ROUTES.INSTANCE_DETAIL(saved.id ?? ""));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  });

  if (!isSuper) {
    return (
      <Card className="mx-auto mt-8 max-w-lg">
        <CardHeader>
          <CardTitle>Only the root admin registers connections</CardTitle>
          <CardDescription>Ask an administrator to register this instance.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const field = (key: keyof ConnectionForm) => ({
    ...register(key),
    "aria-invalid": Boolean(errors[key]),
  });

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" render={<Link to={PAGE_ROUTES.INSTANCES} />}>
          <ChevronLeft /> Instances
        </Button>
      </div>
      <div>
        <h1 className="text-base font-medium">Register an instance</h1>
        <p className="text-sm text-muted-foreground">
          Point noXis at a database that already exists. It is probed before it is saved.
        </p>
      </div>

      <Card>
        <CardContent className="flex flex-col gap-6 pt-6">
          <div className="flex flex-col gap-3">
            <StepHeader
              index={1}
              step={step}
              summary={drivers.find((entry) => entry.driver === driver)?.label}
              onEdit={() => setStep(1)}
            />
            {step === 1 && (
              <SingleSelect
                name="driver"
                value={driver}
                onChange={(value) => pickDriver(String(value))}
                options={drivers.map((entry) => ({
                  value: entry.driver,
                  label: entry.label,
                  detail: entry.driver,
                }))}
              />
            )}
          </div>

          <div className="flex flex-col gap-3 border-t border-border pt-4">
            <StepHeader
              index={2}
              step={step}
              summary={values.name ? `${values.name} · ${driver}` : undefined}
              onEdit={() => setStep(2)}
            />
            {step === 2 && (
              <form
                id="create-connection"
                className="grid gap-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  if (credentials) setStep(3);
                  else void submit();
                }}
              >
                <Field>
                  <FieldLabel htmlFor="conn-name">Name *</FieldLabel>
                  <Input id="conn-name" placeholder="prod-db" {...field("name")} />
                  <FieldError errors={[errors.name]} />
                </Field>
                <Field>
                  <FieldLabel htmlFor="conn-dsn">Connection string (optional)</FieldLabel>
                  <Input
                    id="conn-dsn"
                    placeholder={
                      fileDriver ? "/data/app.db" : "postgresql://user:pass@host:5432/db"
                    }
                    value={dsn}
                    onChange={(e) => setDsn(e.target.value)}
                    onBlur={applyDsn}
                  />
                  <FieldDescription>Paste one to fill the fields below.</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="conn-host">
                    {fileDriver ? "File path *" : "Host *"}
                  </FieldLabel>
                  <Input
                    id="conn-host"
                    placeholder={fileDriver ? "/data/app.db" : "db.internal"}
                    {...field("host")}
                  />
                  <FieldError errors={[errors.host]} />
                </Field>
                {!fileDriver && (
                  <div className="grid grid-cols-2 gap-4">
                    <Field>
                      <FieldLabel htmlFor="conn-port">Port *</FieldLabel>
                      <Input id="conn-port" type="number" {...field("port")} />
                      <FieldError errors={[errors.port]} />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="conn-database">
                        {driver === "redis" || driver === "valkey" ? "Index *" : "Database *"}
                      </FieldLabel>
                      <Input id="conn-database" {...field("database")} />
                      <FieldError errors={[errors.database]} />
                    </Field>
                  </div>
                )}
                {fileDriver && (
                  <Field>
                    <FieldLabel htmlFor="conn-database">Database *</FieldLabel>
                    <Input id="conn-database" placeholder="app" {...field("database")} />
                    <FieldError errors={[errors.database]} />
                  </Field>
                )}
                <div className="flex items-center gap-2">
                  <Button type="submit" size="sm">
                    {credentials ? (
                      "Next"
                    ) : (
                      <>
                        <PlugZap /> Probe and save
                      </>
                    )}
                  </Button>
                </div>
              </form>
            )}
          </div>

          {credentials && (
            <div className="flex flex-col gap-3 border-t border-border pt-4">
              <StepHeader index={3} step={step} onEdit={() => setStep(3)} />
              {step === 3 && (
                <form
                  id="create-credentials"
                  className="grid gap-4"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void submit();
                  }}
                >
                  <Field>
                    <FieldLabel htmlFor="conn-user">User *</FieldLabel>
                    <Input id="conn-user" placeholder="postgres" {...field("user")} />
                    <FieldError errors={[errors.user]} />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="conn-password">Password</FieldLabel>
                    <Input
                      id="conn-password"
                      type="password"
                      autoComplete="new-password"
                      {...field("password")}
                    />
                    <FieldDescription>
                      Stored encrypted when the server has a secret key.
                    </FieldDescription>
                  </Field>
                  <div className="flex items-center gap-2">
                    <Button type="submit" size="sm" disabled={busy}>
                      <PlugZap /> {busy ? "Probing…" : "Probe and save"}
                    </Button>
                  </div>
                </form>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <p className="flex items-center gap-2 text-xs text-muted-foreground">
        {driver && <DriverMark driver={driver} className="size-4 text-[9px]" />}
        {driver
          ? `This instance will be registered under the ${driver} driver.`
          : "Pick a driver to start."}
      </p>
    </div>
  );
}
