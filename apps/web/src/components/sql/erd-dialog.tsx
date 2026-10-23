import { useEffect, useId, useRef, useState } from "react";
import { Network } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useErdQuery } from "@/hooks/use-sql";

interface Props {
  connection?: string;
}

// ErdDialog renders one connection's schema as a Mermaid ER diagram. Mermaid is
// a large dependency and is only needed here, so it is imported when the dialog
// opens instead of in the console's first bundle.
export function ErdDialog({ connection }: Props) {
  const [open, setOpen] = useState(false);
  const erd = useErdQuery(connection, open);
  const [svg, setSvg] = useState("");
  const [error, setError] = useState("");
  const id = useId().replace(/[^a-zA-Z0-9]/g, "");
  const box = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const mermaid = erd.data?.mermaid;
    if (!mermaid) return;
    let cancelled = false;
    setError("");
    void (async () => {
      try {
        const { default: render } = await import("mermaid");
        render.initialize({ startOnLoad: false, theme: "neutral", securityLevel: "strict" });
        const { svg: out } = await render.render(`erd-${id}`, mermaid);
        if (!cancelled) setSvg(out);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "the diagram could not be rendered");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [erd.data?.mermaid, id]);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button variant="outline" disabled={!connection}>
            <Network /> ERD
          </Button>
        }
      />
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Entity relationships</DialogTitle>
          <DialogDescription>
            Tables, keys and declared foreign keys of {connection ?? "the connection"}. Nothing is inferred from column names.
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[70vh]">
          {erd.isPending && <p className="text-sm text-muted-foreground">Loading schema…</p>}
          {error && <p className="text-sm text-destructive">{error}</p>}
          {!error && <div ref={box} className="overflow-auto" dangerouslySetInnerHTML={{ __html: svg }} />}
          {erd.data && (erd.data.tables ?? []).length === 0 && (
            <p className="text-sm text-muted-foreground">This connection has no tables.</p>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
