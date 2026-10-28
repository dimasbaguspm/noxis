import { useState } from "react";

/** The three fields the create form resets on every open. */
interface Draft {
  notes: string;
  query: string;
  rollback: string;
}

export interface ChangeSheetOptions {
  reset: (values: Draft) => void;
}

// useChangeSheet owns the create sheet: whether it is open, whether the request is in flight, and the server's last validation message.
export function useChangeSheet({ reset }: ChangeSheetOptions) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [serverError, setServerError] = useState<string>();

  // every open starts from a blank draft; either way the previous server verdict is dropped.
  function setSheetOpen(next: boolean) {
    if (next) reset({ notes: "", query: "", rollback: "" });
    setOpen(next);
    setServerError(undefined);
  }

  return { open, busy, serverError, setSheetOpen, setBusy, setServerError };
}
