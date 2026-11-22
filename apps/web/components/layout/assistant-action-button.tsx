import Link from "next/link";
import { MessagesSquare } from "lucide-react";

import { cn } from "@/lib/cn";

type AssistantActionButtonProps = {
  active?: boolean;
  className?: string;
};

/**
 * Elevated center action in the mobile bottom navigation (DESIGN.md §7.1).
 * The Assistant is the most prominent action but must not obscure neighbors.
 */
export function AssistantActionButton({
  active = false,
  className,
}: AssistantActionButtonProps) {
  return (
    <Link
      href="/assistant"
      aria-label="Buka Asisten"
      aria-current={active ? "page" : undefined}
      className={cn(
        "group -mt-7 flex w-16 shrink-0 flex-col items-center gap-1",
        className,
      )}
    >
      <span
        className={cn(
          "grid size-14 place-items-center rounded-full bg-gradient-to-br from-primary to-primary-deep text-on-primary shadow-glow-primary",
          "transition-transform duration-150 group-active:scale-95",
          active && "ring-2 ring-primary/60 ring-offset-2 ring-offset-background",
        )}
      >
        <MessagesSquare className="size-6" strokeWidth={2.2} aria-hidden />
      </span>
      <span
        className={cn(
          "text-[11px] font-semibold leading-4",
          active ? "text-primary" : "text-muted",
        )}
      >
        Asisten
      </span>
    </Link>
  );
}
