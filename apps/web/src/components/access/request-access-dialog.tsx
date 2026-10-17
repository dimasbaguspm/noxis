import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCreateRequest } from "@/hooks/use-access-requests";
import { useLevelsQuery } from "@/hooks/use-config";
import { usePagedInstancesQuery } from "@/hooks/use-instances";

const schema = z.object({
  connection: z.string().min(1, "pick an instance"),
  level: z.string().min(1, "pick a level"),
  reason: z.string().optional(),
});
type FormValues = z.infer<typeof schema>;

// RequestAccessDialog asks for a level on one connection. The request lands
// pending; the super admin or that instance's admin decides and the grant then
// runs for the configured lifetime. Levels come from the backend catalog.
export function RequestAccessDialog({ connection }: { connection?: string }) {
  const [open, setOpen] = useState(false);
  const { data } = usePagedInstancesQuery();
  const instances = data?.instances ?? [];
  const levels = useLevelsQuery().data?.levels ?? [];
  const create = useCreateRequest();

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { connection: connection ?? "", level: "read", reason: "" },
  });
  const picked = watch("connection");
  const level = watch("level");

  const onSubmit = handleSubmit((values) =>
    create.mutate(values, {
      onSuccess: () => {
        setOpen(false);
        reset();
      },
    }),
  );

  return (
    <>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <Plus /> Request access
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Request access{connection ? ` to ${connection}` : ""}</DialogTitle>
            <DialogDescription>
              Pick the level you need and say why. The super admin or this instance's admin will
              decide.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={onSubmit} className="flex flex-col gap-3">
            {!connection && (
              <Field>
                <FieldLabel>Instance</FieldLabel>
                <Select value={picked} onValueChange={(v) => setValue("connection", v ?? "", { shouldValidate: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="pick an instance" />
                  </SelectTrigger>
                  <SelectContent>
                    {instances.map((instance) => (
                      <SelectItem key={instance.id} value={instance.name}>
                        {instance.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FieldError>{errors.connection?.message}</FieldError>
              </Field>
            )}
            <Field>
              <FieldLabel>Level</FieldLabel>
              <Select value={level} onValueChange={(v) => setValue("level", v ?? "read", { shouldValidate: true })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {levels.map((l) => (
                    <SelectItem key={l.value ?? ""} value={l.value ?? ""}>
                      {l.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldError>{errors.level?.message}</FieldError>
            </Field>
            <Field>
              <FieldLabel htmlFor="reason">Reason</FieldLabel>
              <Input id="reason" placeholder="Why do you need it?" {...register("reason")} />
            </Field>
            <DialogFooter>
              <Button type="submit" size="sm" disabled={create.isPending}>
                Send request
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
