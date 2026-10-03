import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { PlaceholderSpec } from "@/lib/nav";

// PlaceholderPage renders the plan for an engine that is scoped but not
// built: the nav entry exists, and the page says what it will do rather than
// pretending to work.
export function PlaceholderPage({ spec }: { spec: PlaceholderSpec }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <spec.icon className="size-4" />
          {spec.title}
          <Badge variant="outline">soon</Badge>
        </CardTitle>
        <CardDescription>{spec.summary}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <p className="text-xs font-medium tracking-wider text-muted-foreground uppercase">planned</p>
        <ul className="flex flex-col gap-1.5">
          {spec.planned.map((line) => (
            <li key={line} className="flex items-start gap-2 text-sm text-muted-foreground">
              <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-muted-foreground/50" />
              {line}
            </li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-muted-foreground">
          Nothing here is wired yet — the SQL pages (connections, query, DML/DDL review, analytics) are the
          working surface today.
        </p>
      </CardContent>
    </Card>
  );
}
