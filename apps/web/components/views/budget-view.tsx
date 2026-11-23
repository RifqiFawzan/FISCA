"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { Check, Plus, ShieldCheck, X } from "lucide-react";

import { api, useHasToken } from "@/lib/api";
import type { Budget, BudgetProposal } from "@/lib/api";
import { useAction, useApi } from "@/lib/use-api";
import { formatDayMonthYear, formatIDR, parseAmountInput } from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, RetryButton, Skeleton } from "@/components/ui/feedback";
import { Field, Input, Select } from "@/components/ui/field";
import { Progress, Badge, toneForPercent, statusForPercent } from "@/components/ui/progress";
import { Sheet } from "@/components/ui/sheet";
import { EmptyState } from "@/components/ui/empty-state";

export function BudgetView() {
  const [createOpen, setCreateOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const state = useApi(
    () =>
      Promise.all([
        api.budgets.list(),
        api.proposals.list(),
        api.categories.list("expense"),
      ]).then(([budgets, proposals, categories]) => ({ budgets, proposals, categories })),
    "budget:main",
  );
  const action = useAction();

  const authed = useHasToken();

  if (!authed) {
    return (
      <div className="space-y-6">
        <PageHeader title="Budget" />
        <Alert tone="warning">
          <Link href="/login" className="underline">
            Masuk
          </Link>{" "}
          untuk melanjutkan.
        </Alert>
      </div>
    );
  }

  const data = state.data;
  const pendingProposals = (data?.proposals ?? []).filter((p) => p.status === "pending");

  async function approve(proposal: BudgetProposal) {
    const result = await action.run(() => api.proposals.approve(proposal.id, proposal.baseVersion));
    if (result) {
      setNotice("Proposal disetujui. Budget diperbarui ke versi terbaru.");
      state.reload();
    }
  }

  async function reject(proposal: BudgetProposal) {
    const result = await action.run(() => api.proposals.reject(proposal.id));
    if (result) {
      setNotice("Proposal ditolak. Budget tidak berubah.");
      state.reload();
    }
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title="Budget"
        description="Rencanakan batas pengeluaran bulanan. Pelampauan batas hanya memberi peringatan, tidak menolak transaksi."
        action={
          <Button icon={<Plus className="size-4" aria-hidden />} onClick={() => setCreateOpen(true)}>
            Buat budget
          </Button>
        }
      />

      {notice ? <Alert tone="success">{notice}</Alert> : null}
      {action.error ? <Alert tone="error">{action.error}</Alert> : null}

      {state.loading && !data ? (
        <div className="space-y-4">
          <Skeleton className="h-44" />
          <Skeleton className="h-32" />
        </div>
      ) : state.error && !data ? (
        <Alert tone="error" title="Budget gagal dimuat" action={<RetryButton onClick={state.reload} />}>
          {state.error}
        </Alert>
      ) : data ? (
        <>
          {data.budgets.length === 0 ? (
            <EmptyState
              icon={ShieldCheck}
              title="Belum ada budget"
              description="Buat budget bulanan untuk memantau batas pengeluaran dan mendapat peringatan otomatis."
              action={
                <Button onClick={() => setCreateOpen(true)} icon={<Plus className="size-4" aria-hidden />}>
                  Buat budget pertama
                </Button>
              }
            />
          ) : (
            <div className="space-y-4">
              {data.budgets.map((budget) => (
                <BudgetCard key={budget.id} budget={budget} onChanged={state.reload} />
              ))}
            </div>
          )}

          <Card>
            <CardHeader
              title="Usulan perubahan budget"
              description="Perubahan dari Asisten harus disetujui sebelum berlaku"
            />
            <CardBody>
              {pendingProposals.length === 0 ? (
                <p className="py-2 text-sm text-muted">Tidak ada usulan yang menunggu persetujuan.</p>
              ) : (
                <ul className="space-y-3">
                  {pendingProposals.map((proposal) => (
                    <li key={proposal.id} className="rounded-lg border border-line bg-surface-low px-3.5 py-3">
                      <div className="flex flex-wrap items-start justify-between gap-2">
                        <div className="min-w-0">
                          <p className="text-sm font-semibold">
                            Limit baru: {proposal.changes.overallLimit ? formatIDR(proposal.changes.overallLimit) : "tidak berubah"}
                          </p>
                          <p className="mt-0.5 text-xs text-muted">
                            {proposal.rationale ?? "Usulan perubahan"} · basis versi {proposal.baseVersion} ·
                            berlaku sampai {formatDayMonthYear(proposal.expiresAt)}
                          </p>
                        </div>
                        <Badge tone="warning">Menunggu</Badge>
                      </div>
                      <div className="mt-3 flex flex-wrap gap-2">
                        <Button
                          size="sm"
                          icon={<Check className="size-3.5" aria-hidden />}
                          loading={action.pending}
                          onClick={() => approve(proposal)}
                        >
                          Setujui
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          icon={<X className="size-3.5" aria-hidden />}
                          loading={action.pending}
                          onClick={() => reject(proposal)}
                        >
                          Tolak
                        </Button>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </CardBody>
          </Card>
        </>
      ) : null}

      <CreateBudgetSheet
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        categories={data?.categories ?? []}
        onCreated={() => {
          setCreateOpen(false);
          setNotice("Budget berhasil dibuat.");
          state.reload();
        }}
      />
    </div>
  );
}

function BudgetCard({ budget, onChanged }: { budget: Budget; onChanged: () => void }) {
  const [editing, setEditing] = useState(false);
  const [limitRaw, setLimitRaw] = useState(String(budget.overallLimit));
  const edit = useAction();

  async function save() {
    const value = parseAmountInput(limitRaw);
    if (value === null) return;
    const result = await edit.run(() =>
      api.budgets.update(budget.id, { version: budget.version, overallLimit: value }),
      "Budget diperbarui.",
    );
    if (result) {
      setEditing(false);
      onChanged();
    }
  }

  return (
    <Card>
      <CardHeader
        title={`Budget ${formatDayMonthYear(budget.periodStart)} – ${formatDayMonthYear(budget.periodEnd)}`}
        description={`Versi ${budget.version} · ${budget.status}`}
        action={
          <Badge tone={statusForPercent(budget.percent).tone}>
            {(budget.percent * 100).toFixed(0)}% {statusForPercent(budget.percent).label}
          </Badge>
        }
      />
      <CardBody className="space-y-4">
        <div>
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <p className="financial text-2xl font-bold">{formatIDR(budget.spent)}</p>
            <p className="text-sm text-muted">
              dari {formatIDR(budget.overallLimit)} · sisa{" "}
              <span className={budget.remaining < 0 ? "font-semibold text-danger" : "font-semibold text-primary"}>
                {formatIDR(budget.remaining)}
              </span>
            </p>
          </div>
          <Progress
            value={budget.percent}
            tone={toneForPercent(budget.percent)}
            className="mt-3"
            label="Persentase budget terpakai"
          />
        </div>

        {budget.percent >= 0.8 ? (
          <Alert tone={budget.percent >= 1 ? "error" : "warning"}>
            {budget.percent >= 1
              ? "Budget sudah terlampaui. Transaksi tetap dapat dicatat; ini hanya peringatan."
              : "Pengeluaran mendekati batas budget. Pantau kategori di bawah."}
          </Alert>
        ) : null}

        {budget.categories.length > 0 ? (
          <div>
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
              Alokasi kategori
            </h3>
            <ul className="space-y-2.5">
              {budget.categories.map((category) => (
                <li key={category.categoryId}>
                  <div className="mb-1 flex items-center justify-between gap-3 text-xs">
                    <span className="truncate font-medium">{category.name}</span>
                    <span className="flex shrink-0 items-center gap-2">
                      <span className="financial text-muted">
                        {formatIDR(category.spent)} / {formatIDR(category.limit)}
                      </span>
                      <Badge tone={statusForPercent(category.percent).tone}>
                        {(category.percent * 100).toFixed(0)}%{" "}
                        {statusForPercent(category.percent).label}
                      </Badge>
                    </span>
                  </div>
                  <Progress
                    value={category.percent}
                    tone={toneForPercent(category.percent)}
                    className="h-1.5"
                    label={`${category.name} terpakai`}
                  />
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <p className="text-xs text-muted">
            Belum ada alokasi kategori. Sisa belum dialokasikan: {formatIDR(budget.unallocated)}.
          </p>
        )}

        {edit.error ? <Alert tone="error">{edit.error}</Alert> : null}
        {edit.success ? <Alert tone="success">{edit.success}</Alert> : null}

        {editing ? (
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-[180px] flex-1">
              <Field label="Limit keseluruhan" htmlFor="budget-limit">
                <Input
                  id="budget-limit"
                  value={limitRaw}
                  inputMode="decimal"
                  onChange={(event) => setLimitRaw(event.target.value)}
                  className="financial"
                />
              </Field>
            </div>
            <Button variant="ghost" onClick={() => setEditing(false)}>Batal</Button>
            <Button loading={edit.pending} onClick={save}>Simpan</Button>
          </div>
        ) : (
          <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>
            Ubah limit
          </Button>
        )}
      </CardBody>
    </Card>
  );
}

function CreateBudgetSheet({
  open,
  onClose,
  categories,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  categories: { id: string; name: string }[];
  onCreated: () => void;
}) {
  const [period, setPeriod] = useState("this_month");
  const [limitRaw, setLimitRaw] = useState("");
  const [allocations, setAllocations] = useState<Record<string, string>>({});
  const create = useAction();
  const [localError, setLocalError] = useState<string | null>(null);

  const totalAllocated = useMemo(
    () =>
      Object.values(allocations).reduce((sum, raw) => {
        const value = parseAmountInput(raw);
        return sum + (value ?? 0);
      }, 0),
    [allocations],
  );

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setLocalError(null);
    const limit = parseAmountInput(limitRaw);
    if (limit === null) {
      setLocalError("Nominal limit tidak valid. Contoh: 3000000 atau 3jt.");
      return;
    }
    const parsed = Object.entries(allocations)
      .map(([categoryId, raw]) => ({ categoryId, limitAmount: parseAmountInput(raw) ?? 0 }))
      .filter((item) => item.limitAmount > 0);
    if (parsed.reduce((sum, item) => sum + item.limitAmount, 0) > limit) {
      setLocalError("Total alokasi kategori melebihi limit keseluruhan.");
      return;
    }

    const result = await create.run(() =>
      api.budgets.create({
        period,
        overallLimit: limit,
        allocations: parsed,
      }),
    );
    if (result) onCreated();
  }

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title="Buat budget"
      description="Alokasi kategori bersifat opsional dan tidak boleh melebihi limit keseluruhan."
      wide
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="Periode" htmlFor="budget-period">
          <Select id="budget-period" value={period} onChange={(event) => setPeriod(event.target.value)}>
            <option value="this_month">Bulan ini</option>
            <option value="last_month">Bulan lalu</option>
          </Select>
        </Field>

        <Field label="Limit keseluruhan" hint="Contoh: 3jt" htmlFor="budget-new-limit">
          <Input
            id="budget-new-limit"
            inputMode="decimal"
            placeholder="3000000"
            value={limitRaw}
            onChange={(event) => setLimitRaw(event.target.value)}
            className="financial"
          />
        </Field>

        {categories.length > 0 ? (
          <div>
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
              Alokasi per kategori (opsional)
            </h3>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {categories.map((category) => (
                <label key={category.id} className="flex items-center gap-2 rounded-lg border border-line bg-surface-low px-3 py-2">
                  <span className="min-w-0 flex-1 truncate text-xs text-muted">{category.name}</span>
                  <input
                    aria-label={`Alokasi ${category.name}`}
                    inputMode="decimal"
                    placeholder="0"
                    value={allocations[category.id] ?? ""}
                    onChange={(event) =>
                      setAllocations((prev) => ({ ...prev, [category.id]: event.target.value }))
                    }
                    className="financial h-9 w-28 rounded-md border border-line bg-background px-2 text-right text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20"
                  />
                </label>
              ))}
            </div>
            <p className="financial mt-2 text-right text-xs text-muted">
              Total alokasi: {formatIDR(totalAllocated)}
            </p>
          </div>
        ) : null}

        {localError ? <Alert tone="warning">{localError}</Alert> : null}
        {create.error ? <Alert tone="error">{create.error}</Alert> : null}

        <Button type="submit" loading={create.pending} className="w-full">
          Simpan budget
        </Button>
      </form>
    </Sheet>
  );
}
