/**
 * KPI tile for the desktop summary strips (design-refs: tinted icon tile,
 * uppercase label, tabular value, quiet meta line). Values always come from
 * the API; this component only formats what it is given.
 */
export function KpiCard({
  label,
  value,
  meta,
  icon,
  tone,
}: {
  label: string;
  value: string;
  meta: React.ReactNode;
  icon: React.ReactNode;
  tone: "income" | "expense" | "neutral";
}) {
  const valueTone =
    tone === "income" ? "text-income" : tone === "expense" ? "text-expense" : "text-foreground";
  // Design-refs tints the icon tile per semantic color (bg-*/10), not a neutral.
  const iconTone =
    tone === "income"
      ? "bg-income/10 text-income"
      : tone === "expense"
        ? "bg-expense/10 text-expense"
        : "bg-primary/10 text-primary";

  return (
    <div className="rounded-xl border border-line bg-surface px-5 py-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-xs font-semibold uppercase tracking-wider text-muted">
            {label}
          </p>
          <p className={`financial mt-2 text-2xl font-bold tracking-tight ${valueTone}`}>
            {value}
          </p>
        </div>
        <span
          aria-hidden
          className={`grid size-10 shrink-0 place-items-center rounded-xl ${iconTone}`}
        >
          {icon}
        </span>
      </div>
      <p className="mt-3 text-xs text-muted">{meta}</p>
    </div>
  );
}
