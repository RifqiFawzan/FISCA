"use client";

import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

/** Back returns to the previous context, falling back to /home (plan.md §10.2). */
export function BackButton() {
  const router = useRouter();

  function goBack() {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/home");
  }

  return (
    <button
      type="button"
      onClick={goBack}
      aria-label="Kembali"
      className="grid size-11 shrink-0 place-items-center rounded-full text-muted transition-colors hover:bg-surface-high hover:text-foreground"
    >
      <ArrowLeft className="size-5" aria-hidden />
    </button>
  );
}
