import { useState } from "react";
import { stepsFor } from "@/lib/connection-draft";

// useConnectionCreate is the create page's own state: which driver, which step, and whether a submit is in flight.
// Arriving with a driver already chosen opens step two, so a driver page's empty state skips the first question.
export function useConnectionCreate(initialDriver: string) {
  const [driver, setDriver] = useState(initialDriver);
  const [step, setStep] = useState<1 | 2 | 3>(initialDriver ? 2 : 1);
  const [busy, setBusy] = useState(false);
  const steps = stepsFor(driver);

  // choosing a driver reopens step two: the fields and their defaults belong to that driver
  function choose(next: string) {
    setDriver(next);
    setStep(2);
  }

  // next advances to the credentials step, or submits when the driver has none
  function next(hasCredentials: boolean) {
    if (hasCredentials) setStep(3);
  }

  return { driver, step, setStep, busy, setBusy, steps, choose, next };
}
