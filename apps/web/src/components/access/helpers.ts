import type { MemberUser, Permission } from "@/lib/api";

// knownGroups is the sorted union of a group list, for the datalist.
export function knownGroups(groups?: string[]): string[] {
  return [...new Set(groups ?? [])].sort();
}

// memberLabel renders a permission's member: a user's label or the group name.
export function memberLabel(
  p: Pick<Permission, "member" | "memberType">,
  users?: MemberUser[],
): string {
  return p.memberType === "user"
    ? (users?.find((u) => u.subject === p.member)?.label ?? p.member)
    : p.member;
}
