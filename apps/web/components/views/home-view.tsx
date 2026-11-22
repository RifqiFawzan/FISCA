"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowDownLeft,
  ArrowUpRight,
  MessagesSquare,
  Plus,
  ReceiptText,
  TrendingDown,
  TrendingUp,
  Wallet,
} from "lucide-react";

import { api, hasToken } from "@/lib/api";
import type { CategorySpend, CreateTransactionPayload, Warning } from "@/lib/api";
import { useAction, useApi } from "@/lib/use-api";
import { useIsDesktop } from "@/lib/use-desktop";
import { formatCompact, formatIDR, formatSignedIDR } from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, RetryButton, Skeleton } from "@/components/ui/feedback";
import { Progress, toneForPercent } from "@/components/ui/progress";
import { Segmented, PERIOD_OPTIONS } from "@/components/ui/segmented";
import { KpiCard } from "@/components/ui/kpi-card";
import { Sheet } from "@/components/ui/sheet";
import { EmptyState } from "@/components/ui/empty-state";
import { TransactionList } from "@/components/finance/transaction-row";
import { TransactionTable } from "@/components/finance/transaction-table";
import { TransactionForm } from "@/components/finance/transaction-form";

const ACCOUNT_TYPE_LABELS: Record<string, string> = {
  cash: "Tunai",
  bank: "Bank",
  ewallet: "E-Wallet",
  other: "Lainnya",
};

export function HomeView() {
  const [period, setPeriod] = useState<string>("this_month");
  const [sheetOpen, setSheetOpen] = useState(false);
  const [notice, setNotice] = useState<{ text: string; warnings: Warning[] } | null>(null);
  const isDesktop = useIsDesktop();

  const dashboard = useApi(() => api.analytics.dashboard(period), `dashboard:${period}`);
  const me = useApi(() => api.me.get(), "me");
  const accounts = useApi(() => api.accounts.list(), "accounts");
  const categories = useApi(() => api.categories.list(), "categories");
  const savings = useApi(() => api.savings.list(), "savings-home");
  const create = useAction();

  async function submit(payload: CreateTransactionPayload) {
    const result = await create.run(() => api.transactions.create(payload));
    if (!result) return;
    setNotice({
      text: `${result.transaction.description} berhasil dicatat.`,
      warnings: result.warnings,
    });
    setSheetOpen(false);
    dashboard.reload();
    accounts.reload();
  }

  const data = dashboard.data;
  const totalBalance = (data?.accounts ?? []).reduce((sum, a) => sum + a.balance, 0);
  const goalList = savings.data ?? [];
  const totalSaved = goalList.reduce((sum, goal) => sum + goal.saved, 0);
  const totalTarget = goalList.reduce((sum, goal) => sum + goal.targetAmount, 0);

  if (!hasToken()) {
    return (
      <div className="space-y-6">
        <PageHeader title="Beranda" />
        <Alert tone="warning" title="Belum ada token akses">
          <Link href="/login" className="underline">
            Masuk
          </Link>{" "}
          untuk melanjutkan.
        </Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Design-refs home: the welcome bar lives inside a frosted panel at lg+;
          below lg it stays an unframed header so mobile is untouched. */}
      <section className="lg:rounded-xl lg:border lg:border-line lg:bg-surface/70 lg:p-5 lg:backdrop-blur-[16px]">
        <PageHeader
          title={`Halo, ${me.data?.displayName?.split(" ")[0] ?? "teman"} 👋`}
          description="Ringkasan arus kas Anda dalam satu tempat."
          action={
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-4">
              <Segmented
                value={period}
                onChange={setPeriod}
                options={PERIOD_OPTIONS}
                size="sm"
                className="max-lg:hidden"
              />
              <Button
                icon={<Plus className="size-4" aria-hidden />}
                onClick={() => setSheetOpen(true)}
              >
                Catat transaksi
              </Button>
            </div>
          }
        />
      </section>

      <Segmented
        value={period}
        onChange={setPeriod}
        options={PERIOD_OPTIONS}
        size="sm"
        className="w-full sm:w-auto lg:hidden"
      />

      {notice ? (
        <Alert tone="success" title={notice.text}>
          {notice.warnings.length > 0
            ? notice.warnings.map((warning) => warning.message).join(" ")
            : "Data sudah tersimpan di server."}
        </Alert>
      ) : null}

      {dashboard.loading && !data ? (
        <LoadingSkeleton />
      ) : dashboard.error && !data ? (
        <Alert
          tone="error"
          title="Ringkasan gagal dimuat"
          action={<RetryButton onClick={dashboard.reload} />}
        >
          {dashboard.error}
        </Alert>
      ) : data ? (
        <>
          <Card className="overflow-hidden xl:hidden">
            <div className="relative px-5 py-6 sm:px-7 sm:py-7">
              <div
                aria-hidden
                className="pointer-events-none absolute -right-16 -top-24 size-64 rounded-full bg-primary/10 blur-3xl"
              />
              <div className="relative">
                <p className="flex items-center gap-2 text-xs font-medium uppercase tracking-widest text-muted">
                  <Wallet className="size-3.5" aria-hidden />
                  Total saldo
                </p>
                <p className="financial mt-2 text-3xl font-bold tracking-tight sm:text-4xl">
                  {formatIDR(totalBalance)}
                </p>
                <p className="mt-1 text-xs text-muted">
                  {data.accounts.length} akun aktif · {data.period.label}
                </p>

                <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
                  <MiniStat
                    label="Pemasukan"
                    value={formatIDR(data.summary.income)}
                    icon={<ArrowDownLeft className="size-4" aria-hidden />}
                    tone="income"
                  />
                  <MiniStat
                    label="Pengeluaran"
                    value={formatIDR(data.summary.expense)}
                    icon={<ArrowUpRight className="size-4" aria-hidden />}
                    tone="expense"
                  />
                  <MiniStat
                    label="Selisih"
                    value={formatSignedIDR(data.summary.net)}
                    icon={null}
                    tone={data.summary.net >= 0 ? "income" : "expense"}
                  />
                </div>
              </div>
            </div>
          </Card>

          <div className="hidden gap-4 xl:grid xl:grid-cols-4">
            <KpiCard
              label="Total saldo bersih"
              value={formatIDR(totalBalance)}
              meta={`${data.accounts.length} akun aktif · ${data.period.label}`}
              icon={<Wallet className="size-5" aria-hidden />}
              tone="neutral"
            />
            <KpiCard
              label="Pemasukan"
              value={formatIDR(data.summary.income)}
              meta={`Periode ${data.period.label}`}
              icon={<ArrowDownLeft className="size-5" aria-hidden />}
              tone="income"
            />
            <KpiCard
              label="Pengeluaran"
              value={formatIDR(data.summary.expense)}
              meta={`Periode ${data.period.label}`}
              icon={<ArrowUpRight className="size-5" aria-hidden />}
              tone="expense"
            />
            <KpiCard
              label="Selisih arus kas"
              value={formatSignedIDR(data.summary.net)}
              meta={
                data.summary.net >= 0
                  ? "Pemasukan melebihi pengeluaran"
                  : "Pengeluaran melebihi pemasukan"
              }
              icon={
                data.summary.net >= 0 ? (
                  <TrendingUp className="size-5" aria-hidden />
                ) : (
                  <TrendingDown className="size-5" aria-hidden />
                )
              }
              tone={data.summary.net >= 0 ? "income" : "expense"}
            />
          </div>

          <div className="grid grid-cols-1 gap-6 xl:grid-cols-12 xl:items-start">
            <div className="grid min-w-0 grid-cols-1 gap-4 lg:grid-cols-2 xl:col-span-8 xl:grid-cols-1 xl:gap-6">
              <Card>
                <CardHeader
                  title="Budget bulan ini"
                  description={data.budget ? `Batas ${formatIDR(data.budget.overallLimit)}` : undefined}
                  action={
                    <Link href="/budget" className="text-xs font-semibold text-primary hover:underline">
                      Kelola
                    </Link>
                  }
                />
                <CardBody>
                  {data.budget ? (
                    <div className="space-y-3">
                      <div className="flex items-baseline justify-between gap-3">
                        <span className="financial text-xl font-bold">
                          {formatIDR(data.budget.spent)}
                        </span>
                        <span className="text-xs text-muted">
                          sisa {formatIDR(data.budget.remaining)}
                        </span>
                      </div>
                      <Progress
                        value={data.budget.percent}
                        tone={toneForPercent(data.budget.percent)}
                        label="Persentase budget terpakai"
                      />
                      <p className="text-xs text-muted">
                        {(data.budget.percent * 100).toFixed(0)}% dari limit terpakai.
                        {data.budget.percent >= 0.8
                          ? " Perhatikan pengeluaran Anda."
                          : " Masih terkendali."}
                      </p>
                    </div>
                  ) : (
                    <div className="flex flex-col items-start gap-3 py-2">
                      <p className="text-sm leading-6 text-muted">
                        Belum ada budget. Buat batas bulanan agar FISCA bisa memberi peringatan
                        saat pengeluaran mendekati limit.
                      </p>
                      <Link
                        href="/budget"
                        className="inline-flex items-center gap-2 rounded-lg bg-primary px-3.5 py-2 text-sm font-semibold text-on-primary hover:bg-primary-hover"
                      >
                        Buat budget
                      </Link>
                    </div>
                  )}
                </CardBody>
              </Card>

              <Card>
                <CardHeader
                  title="Kategori terbesar"
                  description={`Distribusi pengeluaran ${data.period.label}`}
                  action={
                    <Link href="/reports" className="text-xs font-semibold text-primary hover:underline">
                      Laporan
                    </Link>
                  }
                />
                <CardBody>
                  {data.topCategories.length > 0 ? (
                    <DistributionBars categories={data.topCategories.slice(0, 5)} />
                  ) : (
                    <p className="py-4 text-sm text-muted">Belum ada pengeluaran pada periode ini.</p>
                  )}
                </CardBody>
              </Card>
            </div>

            <div className="flex min-w-0 flex-col gap-6 xl:col-span-4">
              <Card>
                <CardHeader
                  title="Tabungan"
                  description={
                    goalList.length > 0
                      ? `${goalList.length} target aktif`
                      : "Target dihitung dari saldo akun yang ditautkan"
                  }
                  action={
                    <Link href="/savings" className="text-xs font-semibold text-primary hover:underline">
                      Kelola
                    </Link>
                  }
                />
                <CardBody>
                  {savings.loading && !savings.data ? (
                    <Skeleton className="h-12" />
                  ) : goalList.length > 0 ? (
                    <div className="space-y-3">
                      <div className="flex items-baseline justify-between gap-3">
                        <span className="financial text-xl font-bold">{formatIDR(totalSaved)}</span>
                        <span className="text-sm text-muted">dari {formatIDR(totalTarget)}</span>
                      </div>
                      <Progress
                        value={totalTarget > 0 ? totalSaved / totalTarget : 0}
                        tone={toneForPercent(totalTarget > 0 ? totalSaved / totalTarget : 0)}
                        label="Progres seluruh target tabungan"
                      />
                    </div>
                  ) : (
                    <div className="flex flex-wrap items-center justify-between gap-3 py-1">
                      <p className="text-sm text-muted">
                        Belum ada target. Alokasikan saldo ke sebuah tujuan.
                      </p>
                      <Link
                        href="/savings"
                        className="rounded-lg bg-primary px-3.5 py-2 text-sm font-semibold text-on-primary hover:bg-primary-hover"
                      >
                        Buat target
                      </Link>
                    </div>
                  )}
                </CardBody>
              </Card>

              <Card className="hidden xl:block">
                <CardHeader
                  title="Akun & dompet"
                  description={`${data.accounts.length} akun terdaftar`}
                  action={
                    <Link href="/profile" className="text-xs font-semibold text-primary hover:underline">
                      Kelola
                    </Link>
                  }
                />
                <CardBody>
                  {data.accounts.length > 0 ? (
                    <ul className="space-y-2">
                      {data.accounts.map((account) => (
                        <li
                          key={account.id}
                          className="flex items-center justify-between gap-3 rounded-lg border border-line bg-surface-low px-3.5 py-3"
                        >
                          <div className="flex min-w-0 items-center gap-3">
                            <span
                              aria-hidden
                              className={`grid size-9 shrink-0 place-items-center rounded-lg bg-surface-high text-[11px] font-bold ${
                                account.type === "bank"
                                  ? "text-primary"
                                  : account.type === "ewallet"
                                    ? "text-transfer"
                                    : account.type === "cash"
                                      ? "text-income"
                                      : "text-muted"
                              }`}
                            >
                              {account.name.slice(0, 2).toUpperCase()}
                            </span>
                            <div className="min-w-0">
                              <p className="truncate text-sm font-semibold text-foreground">
                                {account.name}
                              </p>
                              <p className="text-xs text-muted">
                                {ACCOUNT_TYPE_LABELS[account.type] ?? "Lainnya"}
                                {account.isDefault ? " · utama" : ""}
                              </p>
                            </div>
                          </div>
                          <span className="financial shrink-0 text-sm font-bold">
                            {formatIDR(account.balance)}
                          </span>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="py-2 text-sm text-muted">Belum ada akun terdaftar.</p>
                  )}
                </CardBody>
              </Card>
            </div>
          </div>

          <Card>
            <CardHeader
              title="Transaksi terakhir"
              description="Lima transaksi paling baru"
              action={
                <Link href="/history" className="text-xs font-semibold text-primary hover:underline">
                  Lihat semua
                </Link>
              }
            />
            {data.recentTransactions.length > 0 ? (
              isDesktop ? (
                <TransactionTable items={data.recentTransactions.slice(0, 5)} />
              ) : (
                <TransactionList items={data.recentTransactions.slice(0, 5)} />
              )
            ) : (
              <CardBody>
                <EmptyState
                  icon={ReceiptText}
                  title="Belum ada transaksi"
                  description="Catat pengeluaran pertama lewat tombol di atas, atau tanya Asisten FISCA."
                  action={
                    <Link
                      href="/assistant"
                      className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-on-primary"
                    >
                      <MessagesSquare className="size-4" aria-hidden />
                      Buka Asisten
                    </Link>
                  }
                />
              </CardBody>
            )}
          </Card>
        </>
      ) : null}

      <Sheet
        open={sheetOpen}
        onClose={() => setSheetOpen(false)}
        title="Catat transaksi"
        description="Nominal disimpan sebagai rupiah penuh setelah server mengonfirmasi."
      >
        {accounts.loading || categories.loading ? (
          <div className="space-y-3">
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
          </div>
        ) : (
          <TransactionForm
            accounts={accounts.data ?? []}
            categories={categories.data ?? []}
            pending={create.pending}
            error={create.error}
            onSubmit={submit}
            onCancel={() => setSheetOpen(false)}
          />
        )}
      </Sheet>
    </div>
  );
}

/**
 * Stacked spend bar + legend pills, mirroring Stitch's "Distribusi Anggaran".
 * The bar is decorative: every value it shows is repeated as text in the pills,
 * so screen readers get the data without parsing the segments.
 */
function DistributionBars({ categories }: { categories: CategorySpend[] }) {
  const segments = categories
    .map((category) => ({
      name: category.name,
      color: category.color || "#4EDEA3",
      total: category.total,
      percent: Math.max(0, Math.min(1, category.percent || 0)),
    }))
    .filter((segment) => segment.percent > 0);

  const used = segments.reduce((sum, segment) => sum + segment.percent, 0);
  const rest = Math.max(0, 1 - used);

  // Normalise so the bar always fills its track even if the API returns a
  // share above 100% (or categories that do not add up to exactly 1).
  const scale = used + rest > 0 ? 1 / (used + rest) : 1;

  return (
    <div className="space-y-3">
      <div
        aria-hidden
        className="flex h-2 w-full overflow-hidden rounded-full bg-surface-high"
      >
        {segments.map((segment) => (
          <span
            key={segment.name}
            className="h-full transition-[width] duration-300"
            style={{
              backgroundColor: segment.color,
              width: `${segment.percent * scale * 100}%`,
            }}
          />
        ))}
        {rest > 0 ? (
          <span
            className="h-full bg-surface-highest"
            style={{ width: `${rest * scale * 100}%` }}
          />
        ) : null}
      </div>

      <ul className="flex flex-wrap gap-2">
        {segments.map((segment) => (
          <li
            key={segment.name}
            className="inline-flex items-center gap-1.5 rounded-full border border-line bg-surface-low px-2.5 py-1 text-xs"
          >
            <span
              aria-hidden
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: segment.color }}
            />
            <span className="text-foreground">{segment.name}</span>
            <span className="financial font-semibold text-muted">
              {(segment.percent * 100).toFixed(0)}%
            </span>
          </li>
        ))}
        {rest > 0 ? (
          <li className="inline-flex items-center gap-1.5 rounded-full border border-line bg-surface-low px-2.5 py-1 text-xs">
            <span aria-hidden className="size-2 shrink-0 rounded-full bg-surface-highest" />
            <span className="text-muted">Sisa</span>
            <span className="financial font-semibold text-muted">
              {(rest * 100).toFixed(0)}%
            </span>
          </li>
        ) : null}
      </ul>

      <p className="text-xs text-muted">
        {segments.length} kategori terbesar dari {formatCompact(
          segments.reduce((sum, segment) => sum + segment.total, 0),
        )} pengeluaran.
      </p>
    </div>
  );
}

function MiniStat({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: string;
  icon: React.ReactNode;
  tone: "income" | "expense";
}) {
  return (
    <div className="rounded-lg border border-line bg-surface-low/70 px-3.5 py-3">
      <p className="flex items-center gap-1.5 text-xs text-muted">
        <span className={tone === "income" ? "text-income" : "text-expense"}>{icon}</span>
        {label}
      </p>
      <p className="financial mt-1 text-sm font-semibold sm:text-base">{value}</p>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-44" />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Skeleton className="h-40" />
        <Skeleton className="h-40" />
      </div>
      <Skeleton className="h-56" />
    </div>
  );
}
