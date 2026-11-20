import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { api, clearToken, loadConfig, readToken, writeToken } from './client';
import type { Api, Config } from './client';
import type { Identity } from './types';

// One session for the whole console: the API it found, the identity the server confirmed, and the two
// actions that change either. A screen never reads the cookie itself; it asks here.

export type Boot = 'loading' | 'anonymous' | 'ready' | 'failed';

type State = {
  boot: Boot;
  config: Config | null;
  identity: Identity | null;
  error: string | null;
  api: Api | null;
  signIn: (token: string) => Promise<void>;
  signOut: () => void;
};

const SessionContext = createContext<State | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [boot, setBoot] = useState<Boot>('loading');
  const [config, setConfig] = useState<Config | null>(null);
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [error, setError] = useState<string | null>(null);

  const client = useMemo(() => (config ? api(config.api) : null), [config]);

  const verify = useCallback(
    async (client: Api) => {
      const who = await client.whoami();
      setIdentity(who);
      setBoot('ready');
    },
    [],
  );

  useEffect(() => {
    let live = true;
    void (async () => {
      try {
        const found = await loadConfig();
        if (!live) return;
        setConfig(found);
        // A binary started from a signed-in CLI answers with the token that session already holds:
        // the browser inherits it instead of asking for what the terminal knows.
        if (found.token) writeToken(found.token);
        if (!readToken()) {
          setBoot('anonymous');
          return;
        }
        await verify(api(found.api));
      } catch (cause) {
        if (!live) return;
        // A refused token is not a broken console: it is a session that ended, and the card says so.
        setError(cause instanceof Error ? cause.message : String(cause));
        setBoot(readToken() ? 'anonymous' : 'failed');
        clearToken();
      }
    })();
    return () => {
      live = false;
    };
  }, [verify]);

  const signIn = useCallback(
    async (token: string) => {
      if (!config) throw new Error('the console has no API to sign in to');
      writeToken(token.trim());
      const client = api(config.api);
      try {
        await verify(client);
        setError(null);
      } catch (cause) {
        clearToken();
        throw cause;
      }
    },
    [config, verify],
  );

  const signOut = useCallback(() => {
    clearToken();
    setIdentity(null);
    setBoot('anonymous');
  }, []);

  const value: State = { boot, config, identity, error, api: client, signIn, signOut };
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): State {
  const state = useContext(SessionContext);
  if (!state) throw new Error('useSession outside SessionProvider');
  return state;
}

// The scopes a screen asks about, in one place, so a button never guesses its own permission.
export function useCan(scope: string): boolean {
  const { identity } = useSession();
  return identity?.scopes.includes(scope) ?? false;
}
