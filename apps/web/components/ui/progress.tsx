import { cn } from "@/lib/cn";

export function Progress({
  value,
  tone,
  className,
  label,
}: {
  /** 0..1; values above 1 are clamped for the bar but still reported. */
  value: number;
  tone?: "primary" | "info" | "warning" | "danger" | "income" | "transfer";
  className?: string;
  label?: string;
}) {
  const clamped = Math.max(0, Math.min(1, value));
  const toneClass =
    tone === "danger"
      ? "bg-danger"
      : tone === "warning"
        ? "bg-warning"
        : tone === "income"
          ? "bg-income"
          : tone === "transfer"
            ? "bg-transfer"
            : tone === "info"
              ? "bg-ai"
              : "bg-primary";

  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(clamped * 100)}
      aria-label={label}
      className={cn("h-2 w-full overflow-hidden rounded-full bg-surface-high", className)}
    >
      <div
        className={cn("h-full rounded-full transition-[width] duration-300", toneClass)}
        style={{ width: `${clamped * 100}%` }}
      />
    </div>
  );
}

/**
 * Three-step budget status, mirroring the Stitch "Dark Carbon Glass" design:
 * Normal (<80%) / Mendekati Limit (80–100%) / Over Limit (>100%).
 * Returns the Badge tone and the label to render.
 */
export function statusForPercent(percent: number): {
  tone: "primary" | "info" | "danger";
  label: string;
} {
  if (percent >= 1) return { tone: "danger", label: "Over Limit" };
  if (percent >= 0.8) return { tone: "info", label: "Mendekati Limit" };
  return { tone: "primary", label: "Normal" };
}

/**
 * Bar colour follows the same three steps as statusForPercent so a card's
 * badge and its progress bar never disagree (Stitch colours them identically).
 */
export function toneForPercent(percent: number): "primary" | "info" | "danger" {
  return statusForPercent(percent).tone;
}

export function Badge({
  children,
  tone = "neutral",
}: {
  children: React.ReactNode;
  tone?: "neutral" | "primary" | "info" | "warning" | "danger" | "income" | "ai";
}) {
  const tones = {
    neutral: "bg-surface-high text-muted",
    primary: "bg-primary/15 text-primary",
    // Cyan = "mendekati limit"; reuses --color-ai (#4CD7F6, Stitch secondary).
    info: "bg-ai/15 text-ai",
    warning: "bg-warning/15 text-warning",
    danger: "bg-danger/15 text-danger",
    income: "bg-income/15 text-income",
    ai: "bg-ai/15 text-ai",
  };
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-sm px-2 py-0.5 text-[11px] font-semibold",
        tones[tone],
      )}
    >
      {children}
    </span>
  );
}
