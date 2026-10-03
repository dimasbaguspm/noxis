import { Leaf } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

// Placeholder for the next engine family: MongoDB support is scoped but not
// built, and the sidebar says so instead of hiding it.
export default function MongoPage() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Leaf className="size-4" /> MongoDB — soon
        </CardTitle>
        <CardDescription>
          document collections, find/aggregate with the same review flow as SQL
        </CardDescription>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        Not implemented yet. The SQL side (connections, query, DML/DDL review, analytics) works today; the
        document engine will follow the same shape: connection registry, guarded reads, validated changes.
      </CardContent>
    </Card>
  );
}
