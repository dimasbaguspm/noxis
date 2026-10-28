import type { Identity, Level } from "@/lib/api";

// The policy module is the whole permission story: pure functions of the identity, so the provider only holds state.

const LEVEL_RANK: Record<string, number> = { read: 1, write: 2, admin: 3 };

// AccountTarget is the part of an account the console's guards read.
export interface AccountTarget {
  subject: string;
  role?: string;
}

// canAccess answers whether the identity holds a level on one connection; the root and the disabled-auth case pass.
export function canAccess(
  identity: Identity | null,
  authEnabled: boolean,
  connection: string | undefined,
  need: Level,
): boolean {
  if (!authEnabled) return true;
  if (!identity) return false;
  if (identity.role === "super") return true;
  const have = (connection ? identity.connections?.[connection]?.level : "") ?? "";
  return (LEVEL_RANK[have] ?? 0) >= LEVEL_RANK[need];
}

// canManage is the shared guard for the account controls: nobody edits themselves and the env root is untouchable.
export function canManage(identity: Identity | null, target: AccountTarget): boolean {
  return Boolean(identity) && target.subject !== identity?.sub && target.role !== "super";
}

// canBan: an admin acts on members only; the root may also act on admins.
export function canBan(
  identity: Identity | null,
  isSuper: boolean,
  target: AccountTarget,
): boolean {
  return canManage(identity, target) && (isSuper || target.role === "member");
}

// canSetRole: only the root promotes or demotes.
export function canSetRole(
  identity: Identity | null,
  isSuper: boolean,
  target: AccountTarget,
): boolean {
  return isSuper && canManage(identity, target);
}
