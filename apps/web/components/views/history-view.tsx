"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowDownLeft,
  ArrowUpRight,
  Plus,
  Search,
  SlidersHorizontal,
  TrendingDown,
  TrendingUp,
  X,
} from "lucide-react";

import { api, errorMessage, hasToken } from "@/lib/api";
import type { Transaction } from "@/lib/api";
import { useAction, useApi } from "@/lib/use-api";
import { useIsDesktop } from "@/lib/use-desktop";
import { formatDayMonthYear, formatIDR, formatSignedIDR } from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, RetryButton, Skeleton, Spinner } from "@/components/ui/feedback";
import { Input, Field, Select } from "@/components/ui/field";
import { Segmented, PERIOD_OPTIONS } from "@/components/ui/segmented";
import { KpiCard } from "@/components/ui/kpi-card";
import { Sheet } from "@/components/ui/sheet";
import { EmptyState } from "@/components/ui/empty-state";
import { TransactionList } from "@/components/finance/transaction-row";
import { TransactionTable } from "@/components/finance/transaction-table";
import { TransactionForm } from "@/components/finance/transaction-form";

type TypeFilter = "" | "expense" | "income" | "transfer";

export function HistoryView() {
  const [type, setType] = useState<TypeFilter>("");
  const [period, setPeriod] = useState("this_month");
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [nonce, setNonce] = useState(0);
  const [more, setMore] = useState<{
    key: string;
    items: Transaction[];
    cursor: string | null;
  }>({ key: "", items: [], cursor: null });
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Transaction | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  // >=1280px: inspector rail + desktop table; below: Sheet + list (DESIGN.md §5.3).
  const desktop = useIsDesktop();

  const accounts = useApi(() => api.accounts.list(), "accounts");
  const categories = useApi(() => api.categories.list(), "categories");
  const dashboard = useApi(() => api.analytics.dashboard(period), `history-kpi:${period}`);
  const create = useAction();

  const searching = submittedQuery.trim().length > 0;
  const requestKey = `history:${type}:${period}:${searching}:${submittedQuery}:${nonce}`;

  const page = useApi(async () => {
    if (searching) {
      const result = await api.transactions.search(submittedQuery.trim(), 50);
      return { items: result.items, cursor: null, interpreted: result.filters.interpreted };
    }
    const result = await api.transactions.list({ type, period, limit: 20 });
    return { items: result.items, cursor: result.nextCursor, interpreted: null };
  }, requestKey);

  // Extra cursor pages are keyed to the same request so changing a filter
  // discards them without needing a synchronously-fired effect.
  const hasMore = more.key === requestKey;
  const items = [...(page.data?.items ?? []), ...(hasMore ? more.items : [])];
  const cursor = hasMore ? more.cursor : (page.data?.cursor ?? null);
  const interpreted = page.data?.interpreted ?? null;
  const loading = page.loading;
  const error = page.error ?? loadError;

  async function loadMore() {
    if (!cursor) return;
    setLoadingMore(true);
    setLoadError(null);
    try {
      const result = await api.transactions.list({ type, period, limit: 20, cursor });
      setMore((prev) => ({
        key: requestKey,
        items: prev.key === requestKey ? [...prev.items, ...result.items] : result.items,
        cursor: result.nextCursor,
      }));
    } catch (err) {
      setLoadError(errorMessage(err));
    } finally {
      setLoadingMore(false);
    }
  }

  async function submitCreate(payload: Parameters<typeof api.transactions.create>[0]) {
    const result = await create.run(() => api.transactions.create(payload));
    if (!result) return;
    setCreateOpen(false);
    setNonce((n) => n + 1);
  }

  if (!hasToken()) {
    return (
      <div className="space-y-6">
        <PageHeader title="Riwayat" />
        <Alert tone="warning">
          <Link href="/login" className="underline">
            Masuk
          </Link>{" "}
          untuk melanjutkan.
        </Alert>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title="Riwayat"
        description="Cari, tinjau, dan kelola transaksi Anda."
        action={
          <Button icon={<Plus className="size-4" aria-hidden />} onClick={() => setCreateOpen(true)}>
            Catat transaksi
          </Button>
        }
      />

      {desktop && dashboard.data ? (
        <div className="grid grid-cols-3 gap-4">
          <KpiCard
            label="Pemasukan"
            value={formatIDR(dashboard.data.summary.income)}
            meta={`Periode ${dashboard.data.period.label}`}
            icon={<ArrowDownLeft className="size-5" aria-hidden />}
            tone="income"
          />
          <KpiCard
            label="Pengeluaran"
            value={formatIDR(dashboard.data.summary.expense)}
            meta={`Periode ${dashboard.data.period.label}`}
            icon={<ArrowUpRight className="size-5" aria-hidden />}
            tone="expense"
          />
          <KpiCard
            label="Selisih arus kas"
            value={formatSignedIDR(dashboard.data.summary.net)}
            meta={
              dashboard.data.summary.net >= 0
                ? "Pemasukan melebihi pengeluaran"
                : "Pengeluaran melebihi pemasukan"
            }
            icon={
              dashboard.data.summary.net >= 0 ? (
                <TrendingUp className="size-5" aria-hidden />
              ) : (
                <TrendingDown className="size-5" aria-hidden />
              )
            }
            tone={dashboard.data.summary.net >= 0 ? "income" : "expense"}
          />
        </div>
      ) : null}

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-3">
        <form
          onSubmit={(event) => {
            event.preventDefault();
            setSubmittedQuery(query.trim());
          }}
          className="flex min-w-0 flex-1 gap-2 lg:max-w-lg"
        >
          <div className="relative min-w-0 flex-1">
            <Search
              className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-dim"
              aria-hidden
            />
            <Input
              aria-label="Cari transaksi"
              placeholder="Cari: pengeluaran bulan ini di atas 10rb"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="pl-9"
            />
          </div>
          <Button
            type="submit"
            variant="secondary"
            aria-label="Cari"
            icon={<Search className="size-4" aria-hidden />}
          >
            <span className="hidden sm:inline">Cari</span>
          </Button>
          {searching ? (
            <Button
              variant="ghost"
              aria-label="Hapus pencarian"
              icon={<X className="size-4" aria-hidden />}
              onClick={() => {
                setQuery("");
                setSubmittedQuery("");
              }}
            />
          ) : null}
        </form>

        {!searching ? (
          <div className="flex flex-wrap items-center gap-2">
            <SlidersHorizontal className="size-4 text-muted-dim" aria-hidden />
            <Segmented<TypeFilter>
              value={type}
              onChange={setType}
              size="sm"
              options={[
                { value: "", label: "Semua" },
                { value: "expense", label: "Pengeluaran" },
                { value: "income", label: "Pemasukan" },
                { value: "transfer", label: "Transfer" },
              ]}
            />
            <Segmented
              value={period}
              onChange={setPeriod}
              size="sm"
              options={PERIOD_OPTIONS}
            />
          </div>
        ) : (
          <div className="min-w-0 flex-1">
            <Alert tone="info">{interpreted ?? "Hasil pencarian"}</Alert>
          </div>
        )}
      </div>

      {error ? (
        <Alert tone="error" title="Data gagal dimuat" action={<RetryButton onClick={() => setNonce((n) => n + 1)} />}>
          {error}
        </Alert>
      ) : null}

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-12 xl:items-start">
        <div className="min-w-0 xl:col-span-8">
          <Card>
            {loading ? (
              <div className="space-y-3 p-5">
                <Skeleton className="h-12" />
                <Skeleton className="h-12" />
                <Skeleton className="h-12" />
                <Skeleton className="h-12" />
              </div>
            ) : items.length === 0 ? (
              <EmptyState
                icon={Search}
                title={searching ? "Tidak ada transaksi yang cocok" : "Belum ada transaksi"}
                description={
                  searching
                    ? "Coba kata kunci lain, misalnya: kopi bulan ini, atau pengeluaran di atas 50rb."
                    : "Transaksi yang Anda catat lewat chat, suara, atau manual akan muncul di sini."
                }
              />
            ) : (
              <>
                {desktop ? (
                  <TransactionTable items={items} onSelect={setSelected} grouped />
                ) : (
                  <TransactionList items={items} onSelect={setSelected} grouped />
                )}
                {cursor ? (
                  <div className="border-t border-line p-3 text-center">
                    <Button variant="secondary" onClick={loadMore} loading={loadingMore}>
                      Muat lebih banyak
                    </Button>
                  </div>
                ) : null}
              </>
            )}
          </Card>
        </div>

        <aside className="hidden xl:col-span-4 xl:block" aria-label="Detail transaksi terpilih">
          <div className="sticky top-20 rounded-xl border border-line bg-surface p-5">
            {selected && desktop ? (
              <>
                <header className="mb-4 border-b border-line pb-3">
                  <h2 className="text-base font-semibold">Detail transaksi</h2>
                  <p className="mt-0.5 text-xs leading-5 text-muted">
                    {detailSubtitle(selected)}
                  </p>
                </header>
                <DetailPanel
                  transaction={selected}
                  onChanged={() => {
                    setSelected(null);
                    setNonce((n) => n + 1);
                  }}
                />
              </>
            ) : (
              <div className="flex min-h-40 flex-col items-center justify-center gap-2 px-4 py-8 text-center">
                <p className="text-sm font-medium text-foreground">Detail transaksi</p>
                <p className="text-xs leading-5 text-muted">
                  Pilih salah satu transaksi di daftar untuk melihat rinciannya di sini.
                </p>
              </div>
            )}
          </div>
        </aside>
      </div>

      <Sheet
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Catat transaksi"
        description="Transaksi baru muncul setelah server mengonfirmasi."
      >
        <TransactionForm
          accounts={accounts.data ?? []}
          categories={categories.data ?? []}
          pending={create.pending}
          error={create.error}
          onSubmit={submitCreate}
          onCancel={() => setCreateOpen(false)}
        />
      </Sheet>

      {selected && !desktop ? (
        <DetailSheet
          transaction={selected}
          onClose={() => setSelected(null)}
          onChanged={() => {
            setSelected(null);
            setNonce((n) => n + 1);
          }}
        />
      ) : null}
    </div>
  );
}

function detailSubtitle(transaction: Transaction) {
  const typeLabel =
    transaction.type === "income"
      ? "Pemasukan"
      : transaction.type === "transfer"
        ? "Transfer"
        : "Pengeluaran";
  return `${typeLabel} · ${formatDayMonthYear(transaction.occurredAt)}`;
}

function DetailPanel({
  transaction,
  onChanged,
}: {
  transaction: Transaction;
  onChanged: () => void;
}) {
  const detail = useApi(() => api.transactions.get(transaction.id), `detail:${transaction.id}`);
  const action = useAction();
  const [editing, setEditing] = useState(false);
  const [description, setDescription] = useState(transaction.description);
  const [amount, setAmount] = useState(String(transaction.amount));
  const [reason, setReason] = useState("");
  const [voiding, setVoiding] = useState(false);

  const current = detail.data?.transaction ?? transaction;
  const audit = detail.data?.audit ?? [];

  async function saveEdit() {
    const value = Number(amount);
    if (!Number.isInteger(value) || value <= 0) {
      return;
    }
    const result = await action.run(() =>
      api.transactions.update(current.id, {
        version: current.version,
        description: description.trim() || current.description,
        amount: value,
      }),
    );
    if (result) onChanged();
  }

  async function voidTransaction() {
    const result = await action.run(() => api.transactions.void(current.id, reason.trim() || "Dibatalkan"));
    if (result) onChanged();
  }

  return (
      <div className="space-y-5">
        <div className="rounded-lg border border-line bg-surface-low px-4 py-4 text-center">
          <p className="financial text-2xl font-bold">{formatIDR(current.amount)}</p>
          <p className="mt-1 text-sm text-muted">{current.description}</p>
          <div className="mt-3 flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-muted">
            <span>Kategori: {current.categoryName ?? "–"}</span>
            <span>Akun: {current.accountName ?? current.fromAccountName ?? "–"}</span>
            <span>Sumber: {current.source}</span>
            <span>Status: {current.status}</span>
            <span>Versi: {current.version}</span>
          </div>
        </div>

        {action.error ? <Alert tone="error">{action.error}</Alert> : null}
        {action.success ? <Alert tone="success">{action.success}</Alert> : null}

        {editing ? (
          <div className="space-y-3">
            <Field label="Deskripsi" htmlFor="edit-description">
              <Input
                id="edit-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </Field>
            <Field label="Nominal (rupiah)" htmlFor="edit-amount">
              <Input
                id="edit-amount"
                inputMode="numeric"
                value={amount}
                onChange={(event) => setAmount(event.target.value)}
                className="financial"
              />
            </Field>
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setEditing(false)}>Batal</Button>
              <Button loading={action.pending} onClick={saveEdit} className="flex-1">
                Simpan perubahan
              </Button>
            </div>
          </div>
        ) : voiding ? (
          <div className="space-y-3">
            <Field label="Alasan pembatalan" htmlFor="void-reason">
              <Select
                id="void-reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              >
                <option value="">Pilih alasan</option>
                <option value="Salah input">Salah input</option>
                <option value="Duplikat">Duplikat</option>
                <option value="Transaksi dibatalkan">Transaksi dibatalkan</option>
                <option value="Lainnya">Lainnya</option>
              </Select>
            </Field>
            <p className="text-xs leading-5 text-muted">
              Transaksi tidak dihapus: statusnya berubah menjadi dibatalkan dan tetap tercatat di
              riwayat audit.
            </p>
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setVoiding(false)}>Batal</Button>
              <Button variant="danger" loading={action.pending} onClick={voidTransaction} className="flex-1">
                Batalkan transaksi
              </Button>
            </div>
          </div>
        ) : (
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              disabled={current.status === "voided"}
              onClick={() => setEditing(true)}
            >
              Ubah
            </Button>
            <Button
              variant="danger"
              disabled={current.status === "voided"}
              onClick={() => setVoiding(true)}
            >
              Batalkan
            </Button>
          </div>
        )}

        <div>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
            Riwayat perubahan
          </h3>
          {detail.loading ? (
            <Spinner />
          ) : audit.length === 0 ? (
            <p className="text-xs text-muted">Belum ada perubahan.</p>
          ) : (
            <ol className="space-y-2">
              {audit.map((entry) => (
                <li key={entry.id} className="rounded-lg border border-line bg-surface-low px-3 py-2 text-xs">
                  <span className="font-semibold capitalize text-foreground">{entry.action}</span>
                  <span className="text-muted"> · {formatDayMonthYear(entry.createdAt)} · {entry.actor}</span>
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
  );
}

function DetailSheet({
  transaction,
  onClose,
  onChanged,
}: {
  transaction: Transaction;
  onClose: () => void;
  onChanged: () => void;
}) {
  return (
    <Sheet
      open
      onClose={onClose}
      title="Detail transaksi"
      description={detailSubtitle(transaction)}
    >
      <DetailPanel transaction={transaction} onChanged={onChanged} />
    </Sheet>
  );
}
