import { useState } from "react";
import { Navigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { LogoMark } from "@/components/brand/logo-mark";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/app/auth-provider";
import { localLogin, signin } from "@/lib/auth";
import { cn } from "@/lib/utils";
import { H1 } from "@/components/ui/typography";
import { PAGE_ROUTES } from "@/lib/routes";

const credentialsSchema = z.object({
  email: z.email("enter a valid email address"),
  password: z.string().min(1, "password is required"),
});
type Credentials = z.infer<typeof credentialsSchema>;

// LoginPage is the console's entry point when a session is missing.
export default function LoginPage({ className, ...props }: React.ComponentProps<"div">) {
  const { authenticated, localEnabled, oidcEnabled } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Credentials>({
    resolver: zodResolver(credentialsSchema),
    defaultValues: { email: "", password: "" },
  });

  if (authenticated) return <Navigate to={PAGE_ROUTES.INSTANCES} replace />;

  const onValid = handleSubmit(async (values) => {
    setError(null);
    try {
      await localLogin(values.email, values.password);
      window.location.assign(PAGE_ROUTES.INSTANCES); // reload so the provider bootstraps the identity
    } catch (err) {
      setError(err instanceof Error ? err.message : "sign in failed");
    }
  });

  return (
    <div className={cn("flex min-h-svh items-center justify-center bg-background p-6", className)} {...props}>
      <div className="flex w-full max-w-sm flex-col gap-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <a href="/" className="flex flex-col items-center gap-2 font-medium">
            <LogoMark className="size-10" />
            <span className="sr-only">noXis</span>
          </a>
          <H1 className="text-xl font-bold">Welcome to noXis</H1>
          <FieldDescription>
            {localEnabled
              ? "Sign in with your account."
              : "Sign in with your organisation's identity provider."}
          </FieldDescription>
        </div>

        {localEnabled && (
          <form onSubmit={onValid}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="login-email">Email</FieldLabel>
                <Input
                  id="login-email"
                  type="email"
                  autoComplete="email"
                  autoFocus
                  aria-invalid={!!errors.email}
                  {...register("email")}
                />
                <FieldError errors={[errors.email]} />
              </Field>
              <Field>
                <FieldLabel htmlFor="login-password">Password</FieldLabel>
                <Input
                  id="login-password"
                  type="password"
                  autoComplete="current-password"
                  aria-invalid={!!errors.password}
                  {...register("password")}
                />
                <FieldError errors={[errors.password]} />
              </Field>
              {error && <FieldError>{error}</FieldError>}
              <Field>
                <Button type="submit" disabled={isSubmitting}>
                  Sign in
                </Button>
              </Field>
            </FieldGroup>
          </form>
        )}

        {oidcEnabled && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void signin();
            }}
          >
            <FieldGroup>
              <Field>
                <Button variant={localEnabled ? "outline" : "default"} type="submit">
                  Continue with single sign-on
                </Button>
              </Field>
            </FieldGroup>
          </form>
        )}

        <FieldDescription className="text-center">
          Query and change your databases with confidence.
        </FieldDescription>
      </div>
    </div>
  );
}
