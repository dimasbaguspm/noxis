import { useEffect, useRef, useState } from "react";
import { handleCallback, initAuth, loadAuthConfig } from "@/lib/auth";

// AuthCallbackPage finishes the PKCE exchange, then reloads so the AppProvider
// bootstraps with the stored user. It builds the OIDC manager itself: child
// effects run before AppProvider's, so the exchange cannot wait for it.
export default function AuthCallbackPage() {
  const started = useRef(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    void (async () => {
      try {
        initAuth(await loadAuthConfig());
        await handleCallback();
        window.location.replace("/");
      } catch {
        setFailed(true);
      }
    })();
  }, []);

  return (
    <div className="flex h-svh flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
      {failed ? (
        <>
          <p>Sign-in could not be completed.</p>
          <a className="underline" href="/login">Return to sign in</a>
        </>
      ) : (
        "finishing sign-in…"
      )}
    </div>
  );
}
