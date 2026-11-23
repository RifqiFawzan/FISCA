"use client";

import Link from "next/link";
import { Check } from "lucide-react";

import { api } from "@/lib/api";
import type { AssistantCard, Warning } from "@/lib/api";
import { formatIDR, formatMonthLabel } from "@/lib/format";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Badge, Progress, toneForPercent } from "@/components/ui/progress";
import { Alert } from "@/components/ui/feedback";
import { Button } from "@/components/ui/button";
import { useAction } from "@/lib/use-api";
import { TransactionList } from "@/components/finance/transaction-row";

/** Renders the assistant card contract (plan.md §7.3). */
export function AssistantCardView({
  card,
  onApproved,
}: {
  card: AssistantCard;
  onApproved?: () => void;
}) {
  const approve = useAction();
  const trend = card.trend ?? [];

  async function approveProposal() {
    if (!card.proposal) return;
    const result = await approve.run(() =>
      api.proposals.approve(card.proposal!.id, card.proposal!.baseVersion),
    );
    if (result) onApproved?.();
  }

  return (
    <Card className="mt-2 overflow-hidden border-ai/20">
      <CardHeader
        title={card.title || "Hasil"}
        description={card.type.replace(/_/g, " ")}
        action={
          <Badge tone={card.type === "error" ? "danger" : "ai"}>
            {card.type === "transaction_result" ? "Tercatat" : "Asisten"}
          </Badge>
        }
      />
      <CardBody className="space-y-3 pt-3">
        {card.body ? <p className="text-sm leading-6 text-muted">{card.body}</p> : null}

        {card.transaction ? (
          <div className="rounded-lg border border-line bg-surface-low px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-3">
              <span className="min-w-0 truncate text-sm font-semibold">{card.transaction.description}</span>
              <span className="financial shrink-0 text-sm font-bold">{formatIDR(card.transaction.amount)}</span>
            </div>
            <p className="mt-1 text-xs text-muted">
              {card.transaction.categoryName ?? "Tanpa kategori"} · {card.transaction.accountName ?? "Akun utama"}
            </p>
          </div>
        ) : null}

        {card.summary ? (
          <div className="grid grid-cols-3 gap-2">
            <SummaryTile label="Masuk" value={card.summary.income} tone="text-income" />
            <SummaryTile label="Keluar" value={card.summary.expense} tone="text-expense" />
            <SummaryTile
              label="Bersih"
              value={card.summary.net}
              tone={card.summary.net >= 0 ? "text-income" : "text-expense"}
            />
          </div>
        ) : null}

        {/* Chart tren mini dari monthlyTrend backend (angka dihitung server;
            legend + label bulan + teks KPI di atas adalah padanan teksnya). */}
        {trend.length > 0 ? (
          <div className="rounded-lg border border-line bg-surface-low p-3">
            <div className="mb-2 flex items-center justify-between gap-2">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted">
                Tren masuk vs keluar
              </p>
              <span className="flex items-center gap-3 text-[11px] text-muted">
                <span className="flex items-center gap-1">
                  <span aria-hidden className="size-2 rounded-sm bg-income" /> Masuk
                </span>
                <span className="flex items-center gap-1">
                  <span aria-hidden className="size-2 rounded-sm bg-expense" /> Keluar
                </span>
              </span>
            </div>
            <div
              role="img"
              aria-label={`Grafik batang pemasukan dan pengeluaran ${Math.min(12, trend.length)} periode terakhir`}
              className="flex h-20 items-end justify-between gap-1"
            >
              {trend.slice(-12).map((point) => {
                const max = Math.max(
                  1,
                  ...trend.map((x) => Math.max(x.income, x.expense)),
                );
                return (
                  <div
                    key={point.month}
                    title={`${formatMonthLabel(point.month)}: masuk ${formatIDR(point.income)}, keluar ${formatIDR(point.expense)}`}
                    className="flex h-full min-w-0 flex-1 items-end justify-center gap-px"
                  >
                    <span
                      aria-hidden
                      className="w-1/2 rounded-t-sm bg-income/80"
                      style={{ height: `${Math.max(4, (point.income / max) * 100)}%` }}
                    />
                    <span
                      aria-hidden
                      className="w-1/2 rounded-t-sm bg-expense/80"
                      style={{ height: `${Math.max(4, (point.expense / max) * 100)}%` }}
                    />
                  </div>
                );
              })}
            </div>
            <div className="mt-1.5 flex justify-between text-[10px] text-muted">
              <span>{formatMonthLabel(trend[0].month)}</span>
              <span>{formatMonthLabel(trend[trend.length - 1].month)}</span>
            </div>
          </div>
        ) : null}

        {card.period ? (
          <p className="text-xs text-muted">Periode: {card.period.label}</p>
        ) : null}

        {card.categories && card.categories.length > 0 ? (
          <>
            {/* Bar bertumpuk dekoratif; setiap nilai diulang di daftar di
                bawahnya supaya pembaca layar tetap mendapat angkanya. */}
            <div
              aria-hidden
              className="flex h-2 w-full overflow-hidden rounded-full bg-surface-high"
            >
              {card.categories.slice(0, 5).map((category) => (
                <span
                  key={`bar-${category.categoryId ?? category.name}`}
                  className="h-full transition-[width] duration-300"
                  style={{
                    backgroundColor: category.color || "#4EDEA3",
                    width: `${Math.min(1, Math.max(0, category.percent)) * 100}%`,
                  }}
                />
              ))}
            </div>
            <ul className="space-y-2">
              {card.categories.slice(0, 5).map((category) => (
                <li key={category.categoryId ?? category.name}>
                  <div className="mb-1 flex justify-between gap-2 text-xs">
                    <span className="truncate">{category.name}</span>
                    <span className="financial font-semibold">{formatIDR(category.total)}</span>
                  </div>
                  <Progress value={Math.min(1, category.percent)} className="h-1.5" />
                </li>
              ))}
            </ul>
          </>
        ) : null}

        {card.items && card.items.length > 0 ? (
          <div className="-mx-4 border-t border-line sm:-mx-5">
            <TransactionList items={card.items.slice(0, 8)} />
          </div>
        ) : null}

        {card.budget ? (
          <div className="space-y-2">
            <div className="flex justify-between gap-2 text-sm">
              <span className="font-semibold">{formatIDR(card.budget.spent)} terpakai</span>
              <span className="text-muted">dari {formatIDR(card.budget.overallLimit)}</span>
            </div>
            <Progress value={card.budget.percent} tone={toneForPercent(card.budget.percent)} />
            <p className="text-xs text-muted">Sisa {formatIDR(card.budget.remaining)}</p>
          </div>
        ) : null}

        {card.proposal ? (
          <div className="space-y-2">
            <Alert tone="info">
              Perubahan budget belum berlaku. Setujui terlebih dahulu untuk menerapkannya.
            </Alert>
            {approve.error ? <Alert tone="error">{approve.error}</Alert> : null}
            {approve.success ? <Alert tone="success">{approve.success}</Alert> : null}
            <Button
              size="sm"
              icon={<Check className="size-3.5" aria-hidden />}
              loading={approve.pending}
              onClick={approveProposal}
            >
              Setujui usulan
            </Button>
          </div>
        ) : null}

        {card.unusual ? (
          <div className="space-y-2">
            {card.unusual.sufficientData ? (
              card.unusual.findings.length > 0 ? (
                card.unusual.findings.map((finding) => (
                  <Alert key={finding.categoryName} tone="warning">
                    {finding.message}
                  </Alert>
                ))
              ) : (
                <p className="text-sm text-muted">Tidak ada anomali pada periode ini.</p>
              )
            ) : (
              <Alert tone="info">{card.unusual.limitations.join(" ") || "Data belum cukup untuk analisis."}</Alert>
            )}
          </div>
        ) : null}

        {card.warnings && card.warnings.length > 0 ? (
          <div className="space-y-1.5">
            {card.warnings.map((warning: Warning, index) => (
              <Alert key={`${warning.type}-${index}`} tone="warning">
                {warning.message}
              </Alert>
            ))}
          </div>
        ) : null}

        {/* CTA ke halaman nyata (bukan tombol mati): kartu analisis
            mengarah ke laporan penuh, kartu budget ke halaman budget. */}
        {card.type === "financial_summary" ? (
          <Link
            href="/reports"
            className="inline-flex items-center text-xs font-semibold text-primary hover:underline"
          >
            Lihat laporan lengkap
          </Link>
        ) : card.type === "budget_status" && card.budget ? (
          <Link
            href="/budget"
            className="inline-flex items-center text-xs font-semibold text-primary hover:underline"
          >
            Kelola budget
          </Link>
        ) : null}

        {card.missingFields && card.missingFields.length > 0 ? (
          <p className="text-xs text-muted">
            Informasi yang dibutuhkan: {card.missingFields.join(", ")}. Balas pesan ini untuk melengkapi.
          </p>
        ) : null}
      </CardBody>
    </Card>
  );
}

function SummaryTile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: string;
}) {
  return (
    <div className="rounded-lg border border-line bg-surface-low px-2.5 py-2">
      <p className="text-[11px] text-muted">{label}</p>
      <p className={`financial mt-0.5 truncate text-xs font-semibold sm:text-sm ${tone}`}>
        {formatIDR(value)}
      </p>
    </div>
  );
}
