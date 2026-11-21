import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { clearToken, createClient, loadConfig, readToken, unwrap, writeToken } from './http';
import type { Config } from './http';
import type { Identity } from './types';

// One session for the console: which API it found, who the server says it is, and the two actions that
// change either. A screen reads this; nothing else touches the cookie.

export type Boot = 'loading' | 'anonymous' | 'ready';

type Session = {
  boot: Boot;
  api: string | null;
  config: Config | null;
  identity: Identity | null;
  signIn: (token: string) => Promise<void>;
  signOut: () => void;
};

const Context = createContext<Session | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [boot, setBoot] = useState<Boot>('loading');
  const [config, setConfig] = useState<Config | null>(null);
  const [identity, setIdentity] = useState<Identity | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      const found = await loadConfig();
      if (!live) return;
      setConfig(found);
      // Nothing is handed to the browser any more: no session is inherited, so a token in the cookie is
      // one somebody pasted, and a page with none shows the sign-in screen.
      if (!readToken()) {
        setBoot('anonymous');
        return;
      }
      try {
        const who = await unwrap<Identity>(createClient(found.api).get('/v1/whoami'));
        if (!live) return;
        setIdentity(who);
        setBoot('ready');
      } catch {
        if (!live) return;
        clearToken();
        setBoot('anonymous');
      }
    })();
    return () => {
      live = false;
    };
  }, []);

  const signIn = useCallback(
    async (token: string) => {
      if (!config) throw new Error('the console has no API to sign in to');
      const clean = token.trim();
      // The token goes with the request that proves it, not through the cookie written a line before.
      const who = await unwrap<Identity>(createClient(config.api, clean).get('/v1/whoami'));
      writeToken(clean);
      setIdentity(who);
      setBoot('ready');
    },
    [config],
  );

  const signOut = useCallback(() => {
    clearToken();
    setIdentity(null);
    setBoot('anonymous');
  }, []);

  const value = useMemo<Session>(
    () => ({ boot, api: config?.api ?? null, config, identity, signIn, signOut }),
    [boot, config, identity, signIn, signOut],
  );

  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useSession(): Session {
  const session = useContext(Context);
  if (!session) throw new Error('useSession outside SessionProvider');
  return session;
}

/** useCan answers "may this token do it?", so a button never guesses its own permission. */
export function useCan(scope: string): boolean {
  const { identity } = useSession();
  return identity?.scopes.includes(scope) ?? false;
}
