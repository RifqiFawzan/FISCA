import type { ReactNode } from "react";
import { AlertTriangle, CheckCircle2, Info, RotateCw } from "lucide-react";

import { cn } from "@/lib/cn";

export function Alert({
  tone = "error",
  title,
  children,
  action,
  className,
}: {
  tone?: "error" | "warning" | "success" | "info";
  title?: string;
  children?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  const tones = {
    error: "border-danger/30 bg-danger/10 text-danger",
    warning: "border-warning/30 bg-warning/10 text-warning",
    success: "border-success/30 bg-success/10 text-success",
    info: "border-ai/30 bg-ai/10 text-ai",
  };
  const Icon =
    tone === "success" ? CheckCircle2 : tone === "info" ? Info : AlertTriangle;

  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={cn("flex items-start gap-2.5 rounded-lg border px-3.5 py-3", tones[tone], className)}
    >
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1 text-sm leading-5">
        {title ? <p className="font-semibold">{title}</p> : null}
        {children ? <div className={cn(title && "mt-0.5", "text-foreground/90")}>{children}</div> : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  );
}

export function RetryButton({ onClick, label = "Coba lagi" }: { onClick: () => void; label?: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 rounded-md border border-line px-2.5 py-1.5 text-xs font-semibold text-foreground transition-colors hover:bg-surface-high"
    >
      <RotateCw className="size-3.5" aria-hidden />
      {label}
    </button>
  );
}

export function Skeleton({ className }: { className?: string }) {
  return (
    <div
      aria-hidden
      className={cn("animate-pulse rounded-md bg-surface-high/80", className)}
    />
  );
}

export function Spinner({ className }: { className?: string }) {
  return (
    <span
      role="status"
      aria-label="Memuat"
      className={cn(
        "inline-block size-4 animate-spin rounded-full border-2 border-muted border-t-transparent",
        className,
      )}
    />
  );
}
