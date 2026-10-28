import { useState } from "react";
import type { FormEvent } from "react";
import type { useDecideRequest } from "@/hooks/use-access-requests";
import type { useRemovePermission, useSetPermission } from "@/hooks/use-access";
import type { AccessRequest, Permission } from "@/lib/api";

export type AccessSheet =
  | { kind: "add" }
  | { kind: "grant"; permission: Permission }
  | { kind: "request"; request: AccessRequest }
  | null;

export interface InstanceAccessOptions {
  add: ReturnType<typeof useSetPermission>;
  remove: ReturnType<typeof useRemovePermission>;
  decide: ReturnType<typeof useDecideRequest>;
}

// useInstanceAccess owns the access sheet: which of the three modes is open, its draft fields, and the mutations that close it once they succeed.
export function useInstanceAccess({ add, remove, decide }: InstanceAccessOptions) {
  const [sheet, setSheet] = useState<AccessSheet>(null);
  const [memberType, setMemberType] = useState<string>("group");
  const [member, setMember] = useState("");
  const [level, setLevel] = useState("read");
  const [note, setNote] = useState("");

  function openAdd() {
    setMemberType("group");
    setMember("");
    setLevel("read");
    setSheet({ kind: "add" });
  }

  function openGrant(permission: Permission) {
    setMemberType(permission.memberType ?? "group");
    setMember(permission.member);
    setLevel(permission.level ?? "read");
    setSheet({ kind: "grant", permission });
  }

  function openRequest(request: AccessRequest) {
    setNote("");
    setSheet({ kind: "request", request });
  }

  function closeSheet() {
    setSheet(null);
    setNote("");
  }

  function submitGrant(event: FormEvent) {
    event.preventDefault();
    if (!member.trim()) return;
    add.mutate({ member: member.trim(), memberType, level }, { onSuccess: closeSheet });
  }

  function revoke() {
    if (sheet?.kind !== "grant") return;
    remove.mutate(sheet.permission.member, { onSuccess: closeSheet });
  }

  function decideRequest(approve: boolean) {
    if (sheet?.kind !== "request") return;
    decide.mutate({ id: sheet.request.id ?? "", approve, note }, { onSuccess: closeSheet });
  }

  return {
    sheet,
    memberType,
    setMemberType,
    member,
    setMember,
    level,
    setLevel,
    note,
    setNote,
    openAdd,
    openGrant,
    openRequest,
    closeSheet,
    submitGrant,
    revoke,
    decideRequest,
  };
}
