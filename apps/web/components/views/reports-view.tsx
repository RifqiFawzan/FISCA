"use client";

import { useState } from "react";
import Link from "next/link";
import { AlertTriangle, ArrowDownRight, ArrowUpRight, ChartNoAxesColumn } from "lucide-react";

import { api, hasToken } from "@/lib/api";
import { useApi } from "@/lib/use-api";
import {
  formatDayMonth,
  formatIDR,
  formatMonthLabel,
  formatSignedIDR,
} from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Alert, RetryButton, Skeleton } from "@/components/ui/feedback";
import { Segmented, PERIOD_OPTIONS } from "@/components/ui/segmented";
import { EmptyState } from "@/components/ui/empty-state";
import { Badge } from "@/components/ui/progress";
import { TransactionList } from "@/components/finance/transaction-row";

export function ReportsView() {
  const [period, setPeriod] = useState("this_month");

  const reports = useApi(
    () =>
      Promise.all([
        api.analytics.summary(period),
        api.analytics.comparison(period),
        api.analytics.cashFlow({ period, bucket: "day" }),
        api.analytics.largest({ period, limit: 5 }),
        api.analytics.unusual({ period }),
      ]).then(([summary, comparison, cashFlow, largest, unusual]) => ({
        summary,
        comparison,
        cashFlow,
        largest,
        unusual,
      })),
    `reports:${period}`,
  );

  if (!hasToken()) {
    return (
      <div className="space-y-6">
        <PageHeader title="Laporan" />
        <Alert tone="warning">
          <Link href="/login" className="underline">
            Masuk
          </Link>{" "}
          untuk melanjutkan.
        </Alert>
      </div>
    );
  }

  const data = reports.data;
  const savingsRate =
    data && data.summary.summary.income > 0
      ? data.summary.summary.net / data.summary.summary.income
      : null;
  const previousSummary = data?.summary.previous;
  const previousRate =
    previousSummary && previousSummary.income > 0
      ? previousSummary.net / previousSummary.income
      : null;
  const savingsRateDeltaPts =
    savingsRate !== null && previousRate !== null
      ? (savingsRate - previousRate) * 100
      : null;

  return (
    <div className="space-y-5">
      <PageHeader
        title="Laporan"
        description="Pahami ke mana uang Anda pergi."
        action={
          <Segmented
            value={period}
            onChange={setPeriod}
            options={PERIOD_OPTIONS}
            size="sm"
            className="max-lg:hidden"
          />
        }
      />

      <Segmented
        value={period}
        onChange={setPeriod}
        options={PERIOD_OPTIONS}
        size="sm"
        className="w-full sm:w-auto lg:hidden"
      />

      {reports.loading && !data ? (
        <div className="space-y-4">
          <Skeleton className="h-32" />
          <Skeleton className="h-64" />
          <Skeleton className="h-56" />
        </div>
      ) : reports.error && !data ? (
        <Alert tone="error" title="Laporan gagal dimuat" action={<RetryButton onClick={reports.reload} />}>
          {reports.error}
        </Alert>
      ) : data ? (
        <>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            <SummaryCard
              label="Pemasukan"
              value={data.summary.summary.income}
              delta={data.summary.delta.income}
              tone="income"
            />
            <SummaryCard
              label="Pengeluaran"
              value={data.summary.summary.expense}
              delta={data.summary.delta.expense}
              tone="expense"
            />
            <SummaryCard
              label="Arus kas bersih"
              value={data.summary.summary.net}
              delta={data.summary.delta.net}
              tone={data.summary.summary.net >= 0 ? "income" : "expense"}
            />
            <Card className="hidden xl:block">
              <CardBody>
                <p className="text-xs font-medium text-muted">Rasio tabungan</p>
                <p className="financial mt-1.5 text-xl font-bold sm:text-2xl">
                  {savingsRate === null ? "–" : `${(savingsRate * 100).toFixed(1)}%`}
                </p>
                <p
                  className={`mt-1.5 text-xs font-medium ${
                    savingsRateDeltaPts === null
                      ? "text-muted"
                      : savingsRateDeltaPts >= 0
                        ? "text-income"
                        : "text-expense"
                  }`}
                >
                  {savingsRateDeltaPts !== null
                    ? `${savingsRateDeltaPts >= 0 ? "+" : ""}${savingsRateDeltaPts.toFixed(1)} pt vs periode lalu`
                    : `Net dibagi pemasukan · ${data.summary.period.label}`}
                </p>
              </CardBody>
            </Card>
          </div>

          <div className="grid grid-cols-1 gap-5 xl:grid-cols-12 xl:items-start">
            <div className="flex min-w-0 flex-col gap-5 xl:col-span-8">
              <Card>
                <CardHeader
                  title="Arus kas"
                  description={`${data.summary.period.label} · per ${data.cashFlow.bucket === "day" ? "hari" : data.cashFlow.bucket === "week" ? "minggu" : "bulan"}`}
                />
                <CardBody>
                  {data.cashFlow.points.length > 0 ? (
                    <CashFlowChart
                      points={data.cashFlow.points.map((point) => ({
                        label: formatDayMonth(point.bucketStart),
                        income: point.income,
                        expense: point.expense,
                      }))}
                    />
                  ) : (
                    <p className="py-6 text-center text-sm text-muted">Belum ada arus kas pada periode ini.</p>
                  )}
                </CardBody>
              </Card>

              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-1">
                <Card>
                  <CardHeader title="Pengeluaran per kategori" description={data.summary.period.label} />
                  <CardBody>
                    {data.summary.categories.length > 0 ? (
                      <ul className="space-y-3">
                        {data.summary.categories.map((category) => (
                          <li key={category.categoryId ?? category.name}>
                            <div className="mb-1 flex items-baseline justify-between gap-3 text-sm">
                              <span className="flex min-w-0 items-center gap-2">
                                <span
                                  aria-hidden
                                  className="size-2.5 shrink-0 rounded-full"
                                  style={{ backgroundColor: category.color || "#4EDEA3" }}
                                />
                                <span className="truncate">{category.name}</span>
                                <span className="shrink-0 text-xs text-muted">
                                  {(category.percent * 100).toFixed(0)}%
                                </span>
                              </span>
                              <span className="financial shrink-0 font-semibold">
                                {formatIDR(category.total)}
                              </span>
                            </div>
                            <div className="h-1.5 overflow-hidden rounded-full bg-surface-high">
                              <div
                                className="h-full rounded-full"
                                style={{
                                  width: `${Math.min(100, category.percent * 100)}%`,
                                  backgroundColor: category.color || "#4EDEA3",
                                }}
                              />
                            </div>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="py-4 text-sm text-muted">Belum ada pengeluaran.</p>
                    )}
                  </CardBody>
                </Card>

                <Card>
                  <CardHeader title="Tren bulanan" description="12 bulan terakhir" />
                  <CardBody>
                    {data.summary.monthlyTrend.length > 0 ? (
                      <div className="flex h-40 items-end gap-1.5">
                        {data.summary.monthlyTrend.slice(-12).map((month) => {
                          const max = Math.max(
                            1,
                            ...data.summary.monthlyTrend.map((m) => Math.max(m.income, m.expense)),
                          );
                          return (
                            <div key={month.month} className="flex min-w-0 flex-1 flex-col items-center gap-1">
                              <div className="flex h-32 w-full items-end justify-center gap-0.5">
                                <div
                                  title={`Pemasukan ${formatIDR(month.income)}`}
                                  className="w-1/2 rounded-t-sm bg-income/80"
                                  style={{ height: `${Math.max(2, (month.income / max) * 100)}%` }}
                                />
                                <div
                                  title={`Pengeluaran ${formatIDR(month.expense)}`}
                                  className="w-1/2 rounded-t-sm bg-expense/80"
                                  style={{ height: `${Math.max(2, (month.expense / max) * 100)}%` }}
                                />
                              </div>
                              <span className="w-full truncate text-center text-[9px] text-muted">
                                {formatMonthLabel(month.month).replace(" ", "'")}
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <p className="py-4 text-sm text-muted">Belum ada data tren.</p>
                    )}
                    <div className="mt-3 flex items-center gap-4 text-[11px] text-muted">
                      <span className="flex items-center gap-1.5">
                        <span className="size-2 rounded-sm bg-income" /> Pemasukan
                      </span>
                      <span className="flex items-center gap-1.5">
                        <span className="size-2 rounded-sm bg-expense" /> Pengeluaran
                      </span>
                    </div>
                  </CardBody>
                </Card>
              </div>

              <Card>
                <CardHeader
                  title="Perbandingan dengan periode sebelumnya"
                  description={data.comparison.previousPeriod.label}
                />
                <CardBody>
                  {data.comparison.categories.length > 0 ? (
                    <ul className="divide-y divide-line">
                      {data.comparison.categories.slice(0, 8).map((row) => (
                        <li key={row.categoryId ?? row.name} className="flex items-center justify-between gap-3 py-2.5 text-sm">
                          <span className="min-w-0 truncate">{row.name}</span>
                          <span className="financial shrink-0 text-muted">{formatIDR(row.current)}</span>
                          <span
                            className={`financial flex w-28 shrink-0 items-center justify-end gap-1 font-semibold ${
                              row.delta > 0 ? "text-expense" : row.delta < 0 ? "text-income" : "text-muted"
                            }`}
                          >
                            {row.delta > 0 ? (
                              <ArrowUpRight className="size-3.5" aria-hidden />
                            ) : row.delta < 0 ? (
                              <ArrowDownRight className="size-3.5" aria-hidden />
                            ) : null}
                            {formatSignedIDR(row.delta)}
                          </span>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="py-2 text-sm text-muted">Belum ada data pembanding.</p>
                  )}
                </CardBody>
              </Card>
            </div>

            <div className="flex min-w-0 flex-col gap-5 xl:col-span-4">

              <FundFlowCard
                income={data.summary.summary.income}
                expense={data.summary.summary.expense}
                net={data.summary.summary.net}
                periodLabel={data.summary.period.label}
              />

              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-1">
                <Card>
                  <CardHeader title="Transaksi terbesar" description={data.largest.period.label} />
                  {data.largest.items.length > 0 ? (
                    <TransactionList items={data.largest.items} />
                  ) : (
                    <CardBody>
                      <p className="py-4 text-sm text-muted">Belum ada transaksi.</p>
                    </CardBody>
                  )}
                </Card>

                <Card>
                  <CardHeader
                    title="Pengeluaran tidak biasa"
                    description={data.unusual.method || "Analisis median + MAD"}
                  />
                  <CardBody>
                    {!data.unusual.sufficientData ? (
                      <Alert tone="info" title="Data belum cukup">
                        {data.unusual.limitations.join(" ") ||
                          "Butuh lebih banyak riwayat bulanan untuk mendeteksi anomali."}
                      </Alert>
                    ) : data.unusual.findings.length === 0 ? (
                      <p className="py-2 text-sm text-muted">
                        Tidak ada pengeluaran yang menyimpang dari pola {data.unusual.monthsAnalyzed} bulan
                        terakhir.
                      </p>
                    ) : (
                      <ul className="space-y-2">
                        {data.unusual.findings.map((finding) => (
                          <li key={finding.categoryId ?? finding.categoryName} className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2.5">
                            <p className="flex items-center gap-2 text-sm font-semibold">
                              <AlertTriangle className="size-4 text-warning" aria-hidden />
                              {finding.categoryName}
                              <Badge tone="warning">{finding.deviation.toFixed(1)}× median</Badge>
                            </p>
                            <p className="mt-1 text-xs leading-5 text-muted">{finding.message}</p>
                            <p className="financial mt-1 text-xs text-muted">
                              {formatIDR(finding.currentAmount)} vs median {formatIDR(finding.baselineMedian)}
                            </p>
                          </li>
                        ))}
                      </ul>
                    )}
                  </CardBody>
                </Card>
              </div>
            </div>
          </div>

          <p className="pb-2 text-center text-xs text-muted-dim">
            Semua angka dihitung oleh server dari transaksi aktif. Periode: {data.summary.period.label}.
          </p>
        </>
      ) : (
        <EmptyState icon={ChartNoAxesColumn} title="Belum ada laporan" description="Mulai catat transaksi lewat Asisten." action={<Link href="/assistant" className="text-sm font-semibold text-primary">Buka Asisten →</Link>} />
      )}
    </div>
  );
}

function SummaryCard({
  label,
  value,
  delta,
  tone,
}: {
  label: string;
  value: number;
  delta: number;
  tone: "income" | "expense";
}) {
  return (
    <Card>
      <CardBody>
        <p className="text-xs font-medium text-muted">{label}</p>
        <p className="financial mt-1.5 text-xl font-bold sm:text-2xl">{formatIDR(value)}</p>
        <p className={`mt-1.5 flex items-center gap-1 text-xs font-medium ${delta >= 0 ? (tone === "income" ? "text-income" : "text-expense") : "text-income"}`}>
          {delta >= 0 ? <ArrowUpRight className="size-3.5" aria-hidden /> : <ArrowDownRight className="size-3.5" aria-hidden />}
          {formatSignedIDR(delta)} <span className="font-normal text-muted">vs periode lalu</span>
        </p>
      </CardBody>
    </Card>
  );
}

/**
 * Desktop-only fund-flow widget (design's right rail): income in, expense out,
 * what stays. Every figure comes straight from the period summary; the
 * retention rate is net divided by income for the same period.
 */
function FundFlowCard({
  income,
  expense,
  net,
  periodLabel,
}: {
  income: number;
  expense: number;
  net: number;
  periodLabel: string;
}) {
  const retention = income > 0 ? (net / income) * 100 : null;

  return (
    <Card className="hidden xl:block">
      <CardHeader title="Aliran dana" description={periodLabel} />
      <CardBody>
        <ol className="space-y-1.5">
          <li className="flex items-center justify-between rounded-lg border border-line bg-surface-low px-3 py-2.5 text-sm">
            <span className="flex items-center gap-2">
              <span aria-hidden className="size-2.5 shrink-0 rounded-full bg-income" />
              <span className="font-medium">Pemasukan</span>
            </span>
            <span className="financial font-semibold text-income">{formatIDR(income)}</span>
          </li>
          <li aria-hidden className="flex justify-center text-xs text-muted-dim">
            ↓
          </li>
          <li className="flex items-center justify-between rounded-lg border border-line bg-surface-low px-3 py-2.5 text-sm">
            <span className="flex items-center gap-2">
              <span aria-hidden className="size-2.5 shrink-0 rounded-full bg-expense" />
              <span className="font-medium">Pengeluaran</span>
            </span>
            <span className="financial font-semibold text-expense">−{formatIDR(expense)}</span>
          </li>
          <li aria-hidden className="flex justify-center text-xs text-muted-dim">
            ↓
          </li>
          <li className="flex items-center justify-between rounded-lg border border-primary/25 bg-primary/5 px-3 py-2.5 text-sm">
            <span className="flex items-center gap-2">
              <span aria-hidden className="size-2.5 shrink-0 rounded-full bg-primary" />
              <span className="font-semibold">Alokasi bersih</span>
            </span>
            <span
              className={`financial font-bold ${net >= 0 ? "text-income" : "text-expense"}`}
            >
              {formatSignedIDR(net)}
            </span>
          </li>
        </ol>
        <p className="mt-3 text-xs text-muted">
          {retention !== null
            ? `Tingkat retensi kas: ${retention.toFixed(1)}% dari pemasukan periode ini.`
            : "Retensi kas tidak tersedia tanpa pemasukan pada periode ini."}
        </p>
      </CardBody>
    </Card>
  );
}

function CashFlowChart({
  points,
}: {
  points: { label: string; income: number; expense: number }[];
}) {
  const max = Math.max(1, ...points.map((point) => Math.max(point.income, point.expense)));
  const visible = points.slice(-30);
  return (
    <div>
      <div className="flex h-36 items-end justify-center gap-1 overflow-x-auto pb-1">
        {visible.map((point, index) => (
          <div key={`${point.label}-${index}`} className="flex h-full min-w-[10px] max-w-10 flex-1 flex-col items-center justify-end gap-0.5" title={`${point.label}: masuk ${formatIDR(point.income)}, keluar ${formatIDR(point.expense)}`}>
            <div className="flex h-full w-full items-end justify-center gap-px">
              <div className="w-1/2 rounded-t-sm bg-income/80" style={{ height: `${Math.max(1, (point.income / max) * 100)}%` }} />
              <div className="w-1/2 rounded-t-sm bg-expense/80" style={{ height: `${Math.max(1, (point.expense / max) * 100)}%` }} />
            </div>
          </div>
        ))}
      </div>
      <div className="mt-2 flex items-center justify-between text-[11px] text-muted">
        <span>{visible[0]?.label}</span>
        <span className="flex items-center gap-3">
          <span className="flex items-center gap-1"><span className="size-2 rounded-sm bg-income" /> Masuk</span>
          <span className="flex items-center gap-1"><span className="size-2 rounded-sm bg-expense" /> Keluar</span>
        </span>
        <span>{visible[visible.length - 1]?.label}</span>
      </div>
      <p className="mt-1 text-right text-[11px] text-muted">Skala maksimum {formatIDR(max)}</p>
    </div>
  );
}
