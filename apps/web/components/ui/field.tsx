import type {
  InputHTMLAttributes,
  Ref,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

import { cn } from "@/lib/cn";

const base =
  // text-base (16px): iOS Safari auto-zoom membidik input <16px saat fokus,
  // membuat halaman "lompat" di iPhone; skala body-md tetap sah (DESIGN.md §4.2).
  "w-full rounded-lg border border-line bg-surface-low px-3 text-base text-foreground placeholder:text-muted-dim focus:border-primary/60 focus:outline-none focus:ring-2 focus:ring-primary/20 disabled:opacity-60";

export function Label({
  htmlFor,
  children,
  hint,
}: {
  htmlFor?: string;
  children: string;
  hint?: string;
}) {
  return (
    <label htmlFor={htmlFor} className="mb-1.5 flex items-baseline justify-between gap-2">
      <span className="text-xs font-semibold uppercase tracking-wide text-muted">
        {children}
      </span>
      {hint ? <span className="text-[11px] text-muted-dim">{hint}</span> : null}
    </label>
  );
}

export function Input({
  className,
  ref,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { ref?: Ref<HTMLInputElement> }) {
  return <input ref={ref} className={cn(base, "h-11", className)} {...props} />;
}

export function Textarea({
  className,
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(base, "min-h-20 py-2.5", className)} {...props} />;
}

export function Select({
  className,
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={cn(base, "h-11 appearance-none pr-8", className)} {...props}>
      {children}
    </select>
  );
}

export function Field({
  label,
  hint,
  htmlFor,
  children,
}: {
  label: string;
  hint?: string;
  htmlFor?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <Label htmlFor={htmlFor} hint={hint}>
        {label}
      </Label>
      {children}
    </div>
  );
}
