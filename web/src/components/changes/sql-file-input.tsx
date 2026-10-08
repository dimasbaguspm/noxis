import { useRef } from "react";
import { Paperclip } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { statementCount } from "@/lib/statements";

// SqlFileInput attaches one .sql file to a field. One file is one statement,
// so a script is refused with a clear message instead of quietly loading.
export function SqlFileInput({
  onLoad,
  label = "Attach .sql",
}: {
  onLoad: (sql: string, name: string) => void;
  label?: string;
}) {
  const input = useRef<HTMLInputElement>(null);

  async function pick(file: File | undefined) {
    if (!file) return;
    const text = await file.text();
    if (statementCount(text) !== 1) {
      toast.error(`${file.name}: one file must hold exactly one statement`);
      return;
    }
    onLoad(text, file.name);
  }

  return (
    <>
      <input
        ref={input}
        type="file"
        accept=".sql,text/plain"
        className="hidden"
        onChange={(e) => {
          void pick(e.target.files?.[0]);
          e.target.value = "";
        }}
      />
      <Button type="button" size="xs" variant="outline" onClick={() => input.current?.click()}>
        <Paperclip /> {label}
      </Button>
    </>
  );
}
