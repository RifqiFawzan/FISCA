"use client";

import { Fragment } from "react";
import { ArrowDownLeft, ArrowLeftRight, ArrowUpRight } from "lucide-react";

import { cn } from "@/lib/cn";
import { dayGroupKey, dayGroupLabel, formatDayMonth, formatIDR, transactionSign } from "@/lib/format";
import type { Transaction } from "@/lib/api";

/**
 * Desktop transaction table mirroring the design-refs ("Transaksi Terbaru"
 * with Merchant / Kategori / Akun / Waktu / Nominal / Status columns).
 * Rows are display-only unless onSelect is given; `grouped` inserts
 * date-group headers (DESIGN.md §8.3). The mobile counterpart stays
 * TransactionList — callers render exactly one of the two via useIsDesktop.
 */
export function TransactionTable({
  items,
  onSelect,
  grouped = false,
}: {
  items: Transaction[];
  onSelect?: (transaction: Transaction) => void;
  grouped?: boolean;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-left">
        <thead>
          <tr className="border-b border-line text-xs uppercase tracking-wider text-muted">
            <th className="px-4 py-3 font-semibold">Transaksi</th>
            <th className="px-4 py-3 font-semibold">Kategori</th>
            <th className="px-4 py-3 font-semibold">Akun</th>
            <th className="px-4 py-3 font-semibold">Waktu</th>
            <th className="px-4 py-3 text-right font-semibold">Nominal</th>
            <th className="px-4 py-3 text-center font-semibold">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {items.map((tx, index) => {
            const group = grouped ? dayGroupKey(tx.occurredAt) : "";
            const prevGroup =
              grouped && index > 0 ? dayGroupKey(items[index - 1].occurredAt) : "";
            const header =
              group && group !== prevGroup ? (
                <tr key={`${group}-header`}>
                  <td
                    colSpan={6}
                    className="bg-surface-low px-4 py-2 text-xs font-semibold uppercase tracking-wider text-muted"
                  >
                    <span className="inline-flex items-center gap-2">
                      <span aria-hidden className="size-1.5 rounded-full bg-primary" />
                      {dayGroupLabel(tx.occurredAt)}
                    </span>
                  </td>
                </tr>
              ) : null;

            const voided = tx.status === "voided";
            const Icon =
              tx.type === "income" ? ArrowDownLeft : tx.type === "transfer" ? ArrowLeftRight : ArrowUpRight;
            const tile =
              tx.type === "income"
                ? "bg-income/10 text-income"
                : tx.type === "transfer"
                  ? "bg-transfer/10 text-transfer"
                  : "bg-expense/10 text-expense";
            const amountTone =
              tx.type === "income"
                ? "text-income"
                : tx.type === "transfer"
                  ? "text-transfer"
                  : "text-foreground";
            const account = tx.accountName ?? tx.fromAccountName ?? "–";

            return (
              <Fragment key={tx.id}>
                {header}
                <tr
                  className={cn(onSelect && "cursor-pointer hover:bg-surface-high/40")}
                  onClick={onSelect ? () => onSelect(tx) : undefined}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <span aria-hidden className={`grid size-9 shrink-0 place-items-center rounded-lg ${tile}`}>
                        <Icon className="size-4" />
                      </span>
                      {onSelect ? (
                        <button
                          type="button"
                          onClick={() => onSelect(tx)}
                          className={cn(
                            "truncate text-left text-sm font-medium hover:text-primary",
                            voided ? "text-muted-dim line-through" : "text-foreground",
                          )}
                        >
                          {tx.description}
                        </button>
                      ) : (
                        <span
                          className={cn(
                            "truncate text-sm font-medium",
                            voided ? "text-muted-dim line-through" : "text-foreground",
                          )}
                        >
                          {tx.description}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex max-w-40 truncate rounded-full border border-line bg-surface-low px-2.5 py-0.5 text-xs">
                      {tx.categoryName ?? (tx.type === "transfer" ? "Transfer" : "Tanpa kategori")}
                    </span>
                  </td>
                  <td className="truncate px-4 py-3 text-xs text-muted">{account}</td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-muted">
                    {formatDayMonth(tx.occurredAt)}
                  </td>
                  <td
                    className={`financial whitespace-nowrap px-4 py-3 text-right font-semibold tabular-nums ${amountTone} ${voided ? "text-muted-dim line-through" : ""}`}
                  >
                    {transactionSign(tx.type)}
                    {formatIDR(tx.amount)}
                  </td>
                  <td className="px-4 py-3 text-center">
                    {voided ? (
                      <span className="rounded-sm bg-danger/15 px-1.5 py-0.5 text-[10px] font-semibold text-danger">
                        Dibatalkan
                      </span>
                    ) : tx.source !== "manual" ? (
                      <span className="rounded-sm bg-ai/15 px-1.5 py-0.5 text-[10px] font-semibold text-ai">
                        {tx.source === "voice" ? "Voice" : "Chat"}
                      </span>
                    ) : (
                      <span className="text-xs text-muted">Manual</span>
                    )}
                  </td>
                </tr>
              </Fragment>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
