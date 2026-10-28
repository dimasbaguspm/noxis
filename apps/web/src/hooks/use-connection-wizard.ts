import { useState } from "react";
import { toast } from "sonner";
import { FAMILY, type FamilyKey } from "@/lib/families";
import type { Connection } from "@/lib/api";

export interface ConnectionWizardOptions {
  family: FamilyKey;
  wizard: boolean;
  initial?: Connection;
  /** The dialog owns the empty drafts because the form's defaultValues are built from the same map. */
  blank: Record<FamilyKey, Connection>;
  reset: (values: Connection) => void;
}

// useConnectionWizard drives the connection sheet: which step and family are showing, whether the last probe verified, and the DSN/password helpers.
export function useConnectionWizard({
  family,
  wizard,
  initial,
  blank,
  reset,
}: ConnectionWizardOptions) {
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<FamilyKey>(family);
  const [step, setStep] = useState<1 | 2>(wizard && !initial ? 1 : 2);
  const [verified, setVerified] = useState(false);
  const [busy, setBusy] = useState(false);
  const [dsn, setDsn] = useState("");
  const [dsnError, setDsnError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  // Closing resets the form so the next visit starts with a clean family-specific draft.
  function onOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) {
      setKind(family);
      reset(blank[family]);
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

  function choose(next: FamilyKey) {
    setKind(next);
    reset(blank[next]);
    setVerified(false);
    setStep(2);
  }

  const nextStep = () => setStep(2);
  const backStep = () => setStep(1);
  const togglePassword = () => setShowPassword((value) => !value);

  return {
    open,
    setOpen,
    kind,
    setKind,
    step,
    setStep,
    verified,
    setVerified,
    busy,
    setBusy,
    dsn,
    setDsn,
    setDsnError,
    dsnError,
    showPassword,
    togglePassword,
    cache: kind === FAMILY.MEMCACHE,
    mongo: kind === FAMILY.MONGODB,
    onOpenChange,
    choose,
    nextStep,
    backStep,
    pasteDsn,
    clearDsn,
  };
}
