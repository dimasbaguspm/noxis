import { Badge } from "@/components/ui/badge";

// roleBadge renders the one global role an account holds: super (the env root),
// admin (instance operator) or member (per-connection grants only).
export function RoleBadge({ role }: { role?: string }) {
  switch (role) {
    case "super":
      return <Badge variant="success">Super</Badge>;
    case "admin":
      return <Badge variant="default">Admin</Badge>;
    default:
      return <Badge variant="secondary">Member</Badge>;
  }
}
