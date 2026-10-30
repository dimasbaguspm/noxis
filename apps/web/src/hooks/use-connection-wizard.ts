import { useState } from "react";
import { toast } from "sonner";
import type { Connection } from "@/lib/api";

export interface ConnectionWizardOptions {
  wizard: boolean;
  initial?: Connection;
  /** The dialog owns the draft, because the port and the database differ per driver. */
  blank: Connection;
  reset: (values: Connection) => void;
}

// useConnectionWizard drives the connection sheet: which step is showing, whether the last probe verified, and the
// DSN and password helpers. The draft belongs to the dialog, since only it knows which driver was picked.
export function useConnectionWizard({ wizard, initial, blank, reset }: ConnectionWizardOptions) {
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [verified, setVerified] = useState(false);
  const [busy, setBusy] = useState(false);
  const [dsn, setDsn] = useState("");
  const [dsnError, setDsnError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  // Closing resets the form so the next visit starts from a clean draft.
  function onOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) {
      reset(blank);
      setVerified(false);
      setStep(wizard && !initial ? 1 : 2);
    }
  }

  async function pasteDsn() {
    try {
      const text = await navigator.clipboard.readText();
      if (!text) return;
      setDsn(text);
      setDsnError(null);
    } catch {
      toast.error("Could not read the clipboard");
    }
  }

  function clearDsn() {
    setDsn("");
    setDsnError(null);
  }

  const nextStep = () => setStep(2);
  const backStep = () => setStep(1);
  const togglePassword = () => setShowPassword((value) => !value);

  return {
    open,
    setOpen,
    step,
    setStep,
    verified,
    setVerified,
    busy,
    setBusy,
    dsn,
    setDsn,
    dsnError,
    setDsnError,
    showPassword,
    onOpenChange,
    nextStep,
    backStep,
    togglePassword,
    clearDsn,
    pasteDsn,
  };
}
