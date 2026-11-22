"use client";

import {
  ArrowDownLeft,
  ArrowLeftRight,
  ArrowUpRight,
  Undo2,
} from "lucide-react";

import { cn } from "@/lib/cn";
import { formatDayMonth, formatIDR, transactionSign } from "@/lib/format";
import type { Transaction } from "@/lib/api";

const typeStyle = {
  income: { icon: ArrowDownLeft, tile: "bg-income/15 text-income" },
  expense: { icon: ArrowUpRight, tile: "bg-expense/15 text-expense" },
  transfer: { icon: ArrowLeftRight, tile: "bg-transfer/15 text-transfer" },
};

/**
 * Stitch colours the icon tile by category, not by transaction type. Category
 * colour is uncontrolled input, so a tint is only built for a plain 6-digit
 * hex; anything else falls back to the type colour above.
 */
function categoryTile(color: string | null): { background: string; color: string } | null {
  if (!color || !/^#[0-9a-fA-F]{6}$/.test(color)) return null;
  // "26" ≈ 15% alpha, matching the bg-*/15 used by the type fallback and Badge.
  return { background: `${color}26`, color };
}

/**
 * DESIGN.md §8.3 asks each transaction item to show its optional source label.
 * `manual` is the default and stays unlabelled so the row is not noisy; voice
 * and chat are AI-assisted provenance, which is what the label exists for.
 */
function SourceBadge({ source }: { source: Transaction["source"] }) {
  if (source === "manual") return null;
  return (
    <span
      className={cn(
        "shrink-0 rounded-sm px-1.5 py-0.5 text-[10px] font-semibold leading-4",
        "bg-ai/15 text-ai",
      )}
    >
      {source === "voice" ? "Voice" : "Chat"}
    </span>
  );
}

export function TransactionRow({
  transaction,
  onClick,
  className,
}: {
  transaction: Transaction;
  onClick?: () => void;
  className?: string;
}) {
  const style = typeStyle[transaction.type as keyof typeof typeStyle] ?? typeStyle.expense;
  const Icon = style.icon;
  const tile = categoryTile(transaction.categoryColor);
  const voided = transaction.status === "voided";
  const sign = transactionSign(transaction.type);
  const amountClass =
    transaction.type === "income"
      ? "text-income"
      : transaction.type === "expense"
        ? "text-foreground"
        : "text-transfer";

  const content = (
    <>
      <span
        className={cn(
          "grid size-12 shrink-0 place-items-center rounded-lg",
          !tile && style.tile,
        )}
        style={tile ?? undefined}
      >
        <Icon className="size-5" aria-hidden />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 truncate text-sm font-medium text-foreground">
          <span className="truncate">{transaction.description}</span>
          {voided ? (
            <Undo2 className="size-3.5 shrink-0 text-muted-dim" aria-label="Dibatalkan" />
          ) : null}
        </span>
        <span className="mt-0.5 flex items-center gap-1.5 truncate text-xs text-muted">
          <span className="truncate">
            {transaction.categoryName ??
              (transaction.type === "transfer" ? "Transfer" : "Tanpa kategori")}
          </span>
          <SourceBadge source={transaction.source} />
          <span aria-hidden>·</span>
          <span>{formatDayMonth(transaction.occurredAt)}</span>
          {transaction.accountName ? (
            <>
              <span aria-hidden>·</span>
              <span className="truncate">{transaction.accountName}</span>
            </>
          ) : null}
        </span>
      </span>
      <span
        className={cn(
          "financial shrink-0 text-right text-sm font-semibold tabular-nums",
          amountClass,
          voided && "text-muted-dim line-through",
        )}
      >
        {sign}
        {formatIDR(transaction.amount)}
      </span>
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-surface-high/40 sm:px-5",
          className,
        )}
      >
        {content}
      </button>
    );
  }

  return (
    <div className={cn("flex items-center gap-3 px-4 py-3 sm:px-5", className)}>
      {content}
    </div>
  );
}

export function TransactionList({
  items,
  onSelect,
}: {
  items: Transaction[];
  onSelect?: (transaction: Transaction) => void;
}) {
  return (
    <ul className="divide-y divide-line">
      {items.map((transaction) => (
        <li key={transaction.id}>
          <TransactionRow
            transaction={transaction}
            onClick={onSelect ? () => onSelect(transaction) : undefined}
          />
        </li>
      ))}
    </ul>
  );
}
