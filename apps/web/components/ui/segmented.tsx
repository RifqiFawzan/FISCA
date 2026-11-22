"use client";

import { cn } from "@/lib/cn";

export type SegmentOption<T extends string> = { value: T; label: string };

export function Segmented<T extends string>({
  value,
  options,
  onChange,
  className,
  size = "md",
}: {
  value: T;
  options: readonly SegmentOption<T>[];
  onChange: (value: T) => void;
  className?: string;
  size?: "sm" | "md";
}) {
  return (
    <div
      role="group"
      className={cn(
        "inline-flex max-w-full gap-1 overflow-x-auto rounded-lg border border-line bg-surface-low p-1",
        className,
      )}
    >
      {options.map((option) => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(option.value)}
            className={cn(
              "shrink-0 whitespace-nowrap rounded-md font-medium transition-colors",
              size === "sm" ? "px-2 py-1 text-[11px]" : "px-3 py-1.5 text-sm",
              active
                ? "bg-surface-highest text-foreground shadow-sm"
                : "text-muted hover:text-foreground",
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}

export const PERIOD_OPTIONS = [
  { value: "this_month", label: "Bulan ini" },
  { value: "last_month", label: "Bulan lalu" },
  { value: "30d", label: "30 hari" },
  { value: "90d", label: "90 hari" },
  { value: "this_year", label: "Tahun ini" },
] as const;

export type PeriodValue = (typeof PERIOD_OPTIONS)[number]["value"] | string;
