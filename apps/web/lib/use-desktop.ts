"use client";

import { useEffect, useState } from "react";

/**
 * True once the viewport reaches the desktop breakpoint (DESIGN.md §5.3,
 * 1280px). SSR and the first client render agree on `false`, then the
 * listener keeps it in sync — so exactly one of the mobile/desktop
 * variants is ever mounted, instead of hiding duplicates with CSS.
 */
export function useIsDesktop(minWidth = 1280): boolean {
  const [desktop, setDesktop] = useState(false);

  useEffect(() => {
    const media = window.matchMedia(`(min-width: ${minWidth}px)`);
    const update = () => setDesktop(media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [minWidth]);

  return desktop;
}
