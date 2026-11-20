import { createContext, useContext, useMemo } from 'react';
import type { ReactNode } from 'react';
import { useSession } from './session-context';

// Permissions are values, not calls inside a button: the provider derives them once from the identity
// the server confirmed, a screen asks the provider, and `<Can>` renders when the answer is yes. A
// component that hard-codes `useCan('apply')` is doing the provider's job at the leaf.

export type Permission =
  | 'read'
  | 'propose'
  | 'approve'
  | 'apply'
  | 'admin'
  | 'update:target'
  | 'delete:target'
  | 'review:proposal'
  | 'execute:proposal';

type Permissions = {
  scopes: string[];
  can: Record<Permission, boolean>;
  allows: (permission: Permission) => boolean;
};

// One table from "what the operator asks" to "what the token must carry". Admin implies everything,
// which is the one rule worth writing down because it is the one a route table encodes.
const needs: Record<Permission, string | null> = {
  read: 'read',
  propose: 'propose',
  approve: 'approve',
  apply: 'apply',
  admin: 'admin',
  'update:target': 'admin',
  'delete:target': 'admin',
  'review:proposal': 'approve',
  'execute:proposal': 'apply',
};

const Context = createContext<Permissions | null>(null);

export function PermissionProvider({ children }: { children: ReactNode }) {
  const { identity } = useSession();

  const value = useMemo<Permissions>(() => {
    const scopes = identity?.scopes ?? [];
    const admin = scopes.includes('admin');
    const can = Object.fromEntries(
      Object.entries(needs).map(([permission, scope]) => [
        permission,
        scope === null || admin || scopes.includes(scope),
      ]),
    ) as Record<Permission, boolean>;
    return { scopes, can, allows: (permission) => can[permission] ?? false };
  }, [identity]);

  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function usePermissions(): Permissions {
  const permissions = useContext(Context);
  if (!permissions) throw new Error('usePermissions outside PermissionProvider');
  return permissions;
}

/** Can renders its children only when the permission holds, and nothing otherwise. */
export function Can({
  do: permission,
  children,
  fallback = null,
}: {
  do: Permission;
  children: ReactNode;
  fallback?: ReactNode;
}) {
  const { allows } = usePermissions();
  return <>{allows(permission) ? children : fallback}</>;
}
