import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

interface CrumbRegistry {
  labels: Record<string, string>;
  setLabel: (to: string, label: string) => void;
  clearLabel: (to: string) => void;
}

const CrumbContext = createContext<CrumbRegistry | null>(null);

// CrumbLabelProvider lets a page name the record behind an id so the breadcrumb never shows a hash when a name exists.
export function CrumbLabelProvider({ children }: { children: ReactNode }) {
  const [labels, setLabels] = useState<Record<string, string>>({});

  const setLabel = useCallback((to: string, label: string) => {
    setLabels((current) => (current[to] === label ? current : { ...current, [to]: label }));
  }, []);

  const clearLabel = useCallback((to: string) => {
    setLabels((current) => {
      if (!(to in current)) return current;
      const next = { ...current };
      delete next[to];
      return next;
    });
  }, []);

  const value = useMemo(() => ({ labels, setLabel, clearLabel }), [labels, setLabel, clearLabel]);
  return <CrumbContext.Provider value={value}>{children}</CrumbContext.Provider>;
}

export function useCrumbLabels(): CrumbRegistry {
  const value = useContext(CrumbContext);
  if (!value) throw new Error("useCrumbLabels needs a CrumbLabelProvider");
  return value;
}

// useCrumbLabel names the record behind a path while the page is mounted, then hands the crumb back to its fallback.
export function useCrumbLabel(to: string, label?: string) {
  const { setLabel, clearLabel } = useCrumbLabels();
  useEffect(() => {
    if (label) setLabel(to, label);
    return () => clearLabel(to);
  }, [to, label, setLabel, clearLabel]);
}
