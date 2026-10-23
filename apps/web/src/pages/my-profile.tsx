import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { RoleBadge } from "@/components/access/role-badge";
import { useAuth } from "@/app/auth-provider";
import { formatDateTime } from "@/lib/dates";
import { PAGE_ROUTES } from "@/lib/routes";
import { H1, P } from "@/components/ui/typography";

// MyProfilePage is the signed-in account: who you are, your role and groups,
// and the shortcuts to your own settings and access.
export default function MyProfilePage() {
  const { identity, authEnabled } = useAuth();

  if (!identity) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          title={authEnabled ? "Not signed in" : "Authentication disabled"}
          description={authEnabled ? "Sign in to see your profile." : "There is no identity to show."}
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>My profile</H1>
        <P>The account you are signed in with.</P>
      </div>

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
                <RoleBadge role={identity.role} />
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Last login</dt>
              <dd>{formatDateTime(identity.lastLoginAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Groups</dt>
              <dd className="mt-1 flex flex-wrap gap-1.5">
                {(identity.groups ?? []).length === 0 ? (
                  <span className="text-muted-foreground">none</span>
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
          <div className="flex gap-2">
            <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.MY_SETTINGS} />}>
              Settings
            </Button>
            <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.MY_ACCESS} />}>
              My access
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
