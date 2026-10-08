import { useEffect } from "react";
import { handleCallback } from "@/lib/auth";

// AuthCallbackPage finishes the PKCE exchange, then reloads so the AppProvider
// bootstraps with the stored user.
export default function AuthCallbackPage() {
  useEffect(() => {
    handleCallback()
      .catch(() => undefined)
      .finally(() => window.location.replace("/"));
  }, []);

  return (
    <div className="flex h-svh items-center justify-center text-sm text-muted-foreground">
      finishing sign-in…
    </div>
  );
}
