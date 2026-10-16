import { Navigate } from "react-router-dom";
import { GalleryVerticalEnd } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldGroup } from "@/components/ui/field";
import { useAuth } from "@/app/auth-provider";
import { authEnabled, signin } from "@/lib/auth";
import { cn } from "@/lib/utils";

// LoginPage is the console's own entry point when a session is missing: no
// silent bounce to the identity provider. The single CTA starts the OIDC
// flow; the callback route finishes the exchange and lands back in the shell.
export default function LoginPage({ className, ...props }: React.ComponentProps<"div">) {
  const { authenticated } = useAuth();
  if (!authEnabled() || authenticated) return <Navigate to="/" replace />;

  return (
    <div className={cn("flex min-h-svh items-center justify-center bg-background p-6", className)} {...props}>
      <div className="flex w-full max-w-sm flex-col gap-6">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void signin();
          }}
        >
          <FieldGroup>
            <div className="flex flex-col items-center gap-2 text-center">
              <a href="/" className="flex flex-col items-center gap-2 font-medium">
                <div className="flex size-8 items-center justify-center rounded-md">
                  <GalleryVerticalEnd className="size-6" />
                </div>
                <span className="sr-only">noXis</span>
              </a>
              <h1 className="text-xl font-bold">Welcome to noXis</h1>
              <FieldDescription>
                Sign in with your organisation's identity provider to continue.
              </FieldDescription>
            </div>
            <Field>
              <Button type="submit">Sign in</Button>
            </Field>
          </FieldGroup>
        </form>
        <FieldDescription className="text-center">
          Query and change your databases with confidence.
        </FieldDescription>
      </div>
    </div>
  );
}