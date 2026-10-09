import { useEffect } from "react";
import { handleCallback, initAuth, loadAuthConfig } from "@/lib/auth";

// AuthCallbackPage finishes the PKCE exchange, then reloads so the AppProvider
// bootstraps with the stored user. It builds the OIDC manager itself: child
// effects run before AppProvider's, so the exchange cannot wait for it.
export default function AuthCallbackPage() {
  useEffect(() => {
    (async () => {
      try {
        initAuth(await loadAuthConfig());
        await handleCallback();
      } catch {
        /* fall through; the app sends an unauthenticated caller back to sign-in */
      }
      window.location.replace("/");
    })();
  }, []);

  return (
    <div className="flex h-svh items-center justify-center text-sm text-muted-foreground">
      finishing sign-in…
    </div>
  );
}
