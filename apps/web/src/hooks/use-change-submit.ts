import { useState } from "react";

export type ChangeSubmitInitial = {
  connection?: string;
  notes?: string;
  query?: string;
  rollback?: string;
};

type Check = { target?: string; ok?: boolean; detail?: string; error?: string };

/** The three fields the submit form resets between visits. */
interface Draft {
  notes: string;
  query: string;
  rollback: string;
}

export interface ChangeSubmitOptions {
  initial?: ChangeSubmitInitial;
  /** The form reset from react-hook-form: opening prefills the draft from `initial`. */
  reset: (values: Draft) => void;
}

// useChangeSubmit owns the submit sheet: whether it is open, whether the request is in flight, and the checks of the last rejection.
export function useChangeSubmit({ initial, reset }: ChangeSubmitOptions) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<Check[]>([]);

  // opening prefills the form and clears the previous verdict; closing only hides the sheet.
  function setDialogOpen(next: boolean) {
    if (next) {
      reset({
        notes: initial?.notes ?? "",
        query: initial?.query ?? "",
        rollback: initial?.rollback ?? "",
      });
      setFailure([]);
    }
    setOpen(next);
  }

  return { open, busy, failure, setDialogOpen, setBusy, setFailure };
}
