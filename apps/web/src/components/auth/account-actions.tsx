import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/app/auth-provider";
import { changePassword } from "@/lib/auth";

const passwordSchema = z.object({
  oldPassword: z.string().min(1, "Current password is required"),
  newPassword: z.string().min(8, "New password must be at least 8 characters"),
});
type PasswordForm = z.infer<typeof passwordSchema>;

// AccountActions is the signed-in account panel: change password (local accounts only) and sign out.
export function AccountActions() {
  const { signout, localEnabled } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { oldPassword: "", newPassword: "" },
  });

  const onValid = handleSubmit(async (values) => {
    setError(null);
    try {
      await changePassword(values.oldPassword, values.newPassword);
      window.location.assign("/login");
    } catch (err) {
      setError(err instanceof Error ? err.message : "change failed");
    }
  });

  return (
    <div className="flex flex-col gap-4">
      {localEnabled && (
        <Card>
          <CardHeader className="gap-1">
            <CardTitle className="text-base">Password</CardTitle>
            <CardDescription>Changing it signs you out everywhere.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={onValid} className="grid gap-3 sm:max-w-sm">
              <Field>
                <FieldLabel htmlFor="old-password">Current password</FieldLabel>
                <Input
                  id="old-password"
                  type="password"
                  autoComplete="current-password"
                  aria-invalid={!!errors.oldPassword}
                  {...register("oldPassword")}
                />
                <FieldError errors={[errors.oldPassword]} />
              </Field>
              <Field>
                <FieldLabel htmlFor="new-password">New password</FieldLabel>
                <Input
                  id="new-password"
                  type="password"
                  autoComplete="new-password"
                  aria-invalid={!!errors.newPassword}
                  {...register("newPassword")}
                />
                <FieldError errors={[errors.newPassword]} />
              </Field>
              {error && <FieldError>{error}</FieldError>}
              <Button type="submit" disabled={isSubmitting} className="w-fit">
                Change password
              </Button>
            </form>
          </CardContent>
        </Card>
      )}
      <Button variant="outline" className="w-fit" onClick={() => void signout()}>
        Sign out
      </Button>
    </div>
  );
}
