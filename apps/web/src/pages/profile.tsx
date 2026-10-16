import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/app/auth-provider";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import { familyByKey, type FamilyKey } from "@/lib/families";

const LEVEL_LABEL: Record<string, string> = { read: "Read", write: "Editor", admin: "Admin" };

// ProfilePage shows the signed-in identity (from the OIDC provider) and the
// effective level it holds on every instance.
export default function ProfilePage() {
  const { identity, isAdmin, authEnabled } = useAuth();
  const { data } = usePagedInstancesQuery();
  const instances = data?.instances ?? [];

  if (!identity) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          title={authEnabled ? "Not signed in" : "Open mode"}
          description={
            authEnabled
              ? "Sign in to see your profile."
              : "Authentication is disabled, so there is no identity to show."
          }
        />
      </div>
    );
  }

  const levels = identity.connections ?? {};

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">{identity.name || identity.email || identity.sub}</CardTitle>
          <CardDescription>{identity.email || identity.sub}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-muted-foreground">Subject</dt>
              <dd className="font-mono text-xs">{identity.sub}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Role</dt>
              <dd>
                {isAdmin ? <Badge variant="success">Super admin</Badge> : <Badge variant="secondary">Member</Badge>}
              </dd>
            </div>
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Groups</dt>
              <dd className="mt-1 flex flex-wrap gap-1.5">
                {(identity.groups ?? []).length === 0 ? (
                  <span className="text-muted-foreground">none in the token</span>
                ) : (
                  (identity.groups ?? []).map((group) => (
                    <Badge key={group} variant="outline">
                      {group}
                    </Badge>
                  ))
                )}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Access</CardTitle>
          <CardDescription>the level your groups hold on each instance</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Instance</TableHead>
                <TableHead>Family</TableHead>
                <TableHead className="w-[120px]">Level</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {instances.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-muted-foreground">
                    no instance registered
                  </TableCell>
                </TableRow>
              )}
              {instances.map((instance) => {
                const level = levels[instance.name];
                return (
                  <TableRow key={instance.id}>
                    <TableCell className="text-sm">{instance.name}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {familyByKey[instance.family as FamilyKey]?.label ?? instance.family}
                    </TableCell>
                    <TableCell>
                      {level ? (
                        <Badge variant="secondary">{LEVEL_LABEL[level] ?? level}</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}