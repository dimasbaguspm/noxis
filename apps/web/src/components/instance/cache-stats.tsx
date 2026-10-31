import type { CacheInfo } from "@/lib/api";
import { bytes, compact } from "@/lib/format";

// CacheStats is the server's own numbers, kept in the workspace footer so they are always in view without stealing room
// from the key table. A field the server omits reads as absent rather than invented.
export function CacheStats({ info }: { info?: CacheInfo }) {
  if (!info) return null;
  const lookups = (info.keyspaceHits ?? 0) + (info.keyspaceMisses ?? 0);
  const hit = lookups > 0 ? Math.round(((info.keyspaceHits ?? 0) / lookups) * 100) : 0;
  const keys = (info.databases ?? []).reduce((sum, database) => sum + (database.keys ?? 0), 0);
  return (
    <>
      <span>
        {bytes(info.usedMemoryBytes ?? 0)} used
        {lookups > 0 ? ` · ${hit}% hit` : ""} · {compact(info.connectedClients ?? 0)} clients
      </span>
      <span>
        {compact(keys)} keys · {compact(info.evictedKeys ?? 0)} evicted ·{" "}
        {compact(info.expiredKeys ?? 0)} expired
      </span>
    </>
  );
}
