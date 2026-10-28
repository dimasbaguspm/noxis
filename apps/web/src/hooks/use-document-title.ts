import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { matchRoute } from "@/constants/routes";

const BRAND = "noXis";

// useDocumentTitle names the browser tab after the route the user is on.
export function useDocumentTitle() {
  const { pathname } = useLocation();

  useEffect(() => {
    const meta = matchRoute(pathname);
    document.title = meta ? `${meta.title} · ${BRAND}` : BRAND;
  }, [pathname]);
}
