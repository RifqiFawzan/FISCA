"use client";

import { useState } from "react";
import { PiggyBank, Plus, Target } from "lucide-react";

import {
  api,
  type PurchaseAdvisor,
  type SavingsGoal,
} from "@/lib/api";
import { useAction, useApi } from "@/lib/use-api";
import { formatDateInput, formatDayMonthYear, formatIDR, parseAmountInput } from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, RetryButton, Skeleton } from "@/components/ui/feedback";
import { Field, Input, Select } from "@/components/ui/field";
import { Badge, Progress, toneForPercent } from "@/components/ui/progress";
import { EmptyState } from "@/components/ui/empty-state";
import { Sheet } from "@/components/ui/sheet";
import type { Account } from "@/lib/api";

const STATUS_META: Record<string, { label: string; tone: "primary" | "warning" | "danger" | "income" }> = {
  on_track: { label: "Sesuai rencana", tone: "primary" },
  at_risk: { label: "Perlu dipantau", tone: "warning" },
  achieved: { label: "Tercapai", tone: "income" },
  overdue: { label: "Lewat tenggat", tone: "danger" },
};

const VERDICT_META: Record<string, { label: string; tone: "income" | "warning" | "danger" }> = {
  affordable: { label: "Terjangkau", tone: "income" },
  tight: { label: "Bisa, tapi mepet", tone: "warning" },
  not_affordable: { label: "Belum terjangkau", tone: "danger" },
};

function localDate(daysFromNow: number): string {
  const date = new Date();
  date.setDate(date.getDate() + daysFromNow);
  return formatDateInput(date.toISOString());
}

export function SavingsView() {
  const goals = useApi(() => api.savings.list(), "savings-goals");
  const accounts = useApi(() => api.accounts.list(), "accounts");
  const action = useAction();
  const advisorAction = useAction();

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editing, setEditing] = useState<SavingsGoal | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [priceRaw, setPriceRaw] = useState("");
  const [priceError, setPriceError] = useState<string | null>(null);
  const [analysis, setAnalysis] = useState<PurchaseAdvisor | null>(null);

  function openCreate() {
    setEditing(null);
    setSheetOpen(true);
  }

  function openEdit(goal: SavingsGoal) {
    setEditing(goal);
    setSheetOpen(true);
  }

  async function archive(goal: SavingsGoal) {
    const result = await action.run(() => api.savings.archive(goal.id), `Target "${goal.name}" diarsipkan.`);
    if (result) {
      goals.reload();
    }
  }

  async function runAdvisor() {
    setAnalysis(null);
    const price = parseAmountInput(priceRaw);
    if (price === null) {
      setPriceError("Harga tidak valid. Contoh: 1500000 atau 1,5jt.");
      return;
    }
    setPriceError(null);
    const result = await advisorAction.run(() => api.savings.advisor(price));
    if (result) setAnalysis(result);
  }

  const goalList = goals.data ?? [];
  const totalSaved = goalList.reduce((sum, goal) => sum + goal.saved, 0);
  const totalTarget = goalList.reduce((sum, goal) => sum + goal.targetAmount, 0);
  const activeGoals = goalList.filter((goal) => goal.status !== "achieved");
  const achievedCount = goalList.length - activeGoals.length;
  const neededMonthly = activeGoals.reduce((sum, goal) => sum + goal.requiredMonthly, 0);

  return (
    <div className="space-y-5">
      <PageHeader
        title="Tabungan"
        description="Target tabungan dihitung dari saldo akun yang ditautkan, jadi uangnya tidak dihitung dua kali."
        action={
          <Button icon={<Plus className="size-4" aria-hidden />} onClick={openCreate}>
            Target baru
          </Button>
        }
      />

      {notice ? <Alert tone="success">{notice}</Alert> : null}
      {action.error ? <Alert tone="error">{action.error}</Alert> : null}

      {goals.data ? (
        <div className="hidden gap-4 xl:grid xl:grid-cols-4">
          <SavingsStat
            label="Total dana terkumpul"
            value={formatIDR(totalSaved)}
            meta={`${goalList.length} target tercatat`}
            accent
          />
          <SavingsStat
            label="Target keseluruhan"
            value={formatIDR(totalTarget)}
            meta={
              <div className="space-y-1.5">
                <div>
                  {totalTarget > 0 ? `${((totalSaved / totalTarget) * 100).toFixed(1)}% tercapai` : "Belum ada target"}
                </div>
                {totalTarget > 0 ? (
                  <Progress
                    value={totalSaved / totalTarget}
                    tone={toneForPercent(totalSaved / totalTarget)}
                    label="Progres seluruh target tabungan"
                  />
                ) : null}
              </div>
            }
          />
          <SavingsStat
            label="Kebutuhan kontribusi /bulan"
            value={formatIDR(neededMonthly)}
            meta={
              activeGoals.length > 0
                ? `Agar ${activeGoals.length} target berjalan tetap tepat waktu`
                : "Tidak ada target berjalan"
            }
          />
          <SavingsStat
            label="Target aktif"
            value={`${activeGoals.length} rencana`}
            meta={
              achievedCount > 0 ? `${achievedCount} target sudah tercapai` : "Belum ada yang tercapai"
            }
          />
        </div>
      ) : null}

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-12 xl:items-start">
        <div className="min-w-0 xl:col-span-4 xl:col-start-9">
          <Card>
            <CardHeader
              title="Simulasi pembelian"
              description="Semua angka diambil dari ledger; kewajiban yang belum dicatat dinyatakan sebagai keterbatasan."
            />
            <CardBody className="space-y-4">
              <div className="flex flex-wrap items-end gap-2">
                <div className="min-w-[200px] flex-1">
                  <Field label="Harga barang" hint="Rp" htmlFor="advisor-price">
                    <Input
                      id="advisor-price"
                      inputMode="decimal"
                      placeholder="1500000"
                      value={priceRaw}
                      onChange={(event) => setPriceRaw(event.target.value)}
                      className="financial"
                    />
                  </Field>
                </div>
                <Button
                  onClick={runAdvisor}
                  loading={advisorAction.pending}
                  icon={<Target className="size-4" aria-hidden />}
                >
                  Cek keterjangkauan
                </Button>
              </div>

              {priceError ? <Alert tone="warning">{priceError}</Alert> : null}
              {advisorAction.error ? <Alert tone="error">{advisorAction.error}</Alert> : null}

              {analysis ? (
                <div className="rounded-lg border border-line bg-surface-low p-4">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="text-sm text-muted">
                      Untuk {formatIDR(analysis.price)}
                    </span>
                    <Badge tone={VERDICT_META[analysis.verdict].tone}>
                      {VERDICT_META[analysis.verdict].label}
                    </Badge>
                  </div>

                  <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-3">
                    <Fact label="Dana bebas" value={formatIDR(analysis.freeToSpend)} />
                    <Fact label="Saldo semua akun" value={formatIDR(analysis.available)} />
                    <Fact label="Ditahan target tabungan" value={formatIDR(analysis.reservedSavings)} />
                    <Fact label="Sisa budget" value={formatIDR(analysis.remainingBudget)} />
                    <Fact label="Rata-rata masuk/bulan" value={formatIDR(analysis.averageMonthlyIncome)} />
                    <Fact label="Rata-rata keluar/bulan" value={formatIDR(analysis.averageMonthlyExpense)} />
                    <Fact label="Saldo setelah beli" value={formatIDR(analysis.projectedAfterPurchase)} />
                    <Fact label="Bulan ber-data" value={`${analysis.monthsWithData} bulan`} />
                  </dl>

                  <div className="mt-4 space-y-2 text-xs leading-5">
                    <p className="font-semibold uppercase tracking-wide text-muted">Asumsi</p>
                    <ul className="list-inside list-disc space-y-1 text-muted">
                      {analysis.assumptions.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                    <p className="font-semibold uppercase tracking-wide text-muted">Keterbatasan</p>
                    <ul className="list-inside list-disc space-y-1 text-muted">
                      {analysis.limitations.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                  </div>
                </div>
              ) : null}
            </CardBody>
          </Card>
        </div>

        <div className="min-w-0 xl:col-span-8 xl:row-start-1">

          {goals.loading && !goals.data ? (
            <div className="space-y-4">
              <Skeleton className="h-44" />
              <Skeleton className="h-44" />
            </div>
          ) : goals.error && !goals.data ? (
            <Alert
              tone="error"
              title="Target tabungan gagal dimuat"
              action={<RetryButton onClick={goals.reload} />}
            >
              {goals.error}
            </Alert>
          ) : goalList.length === 0 ? (
            <EmptyState
              icon={PiggyBank}
              title="Belum ada target tabungan"
              description="Buat target dengan nominal, tenggat, dan akun yang ditautkan. Progresnya dihitung dari saldo akun tersebut."
              action={
                <Button onClick={openCreate} icon={<Plus className="size-4" aria-hidden />}>
                  Buat target pertama
                </Button>
              }
            />
          ) : (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-1">
              {goalList.map((goal) => {
                const meta = STATUS_META[goal.status] ?? STATUS_META.on_track;
                return (
                  <Card key={goal.id}>
                    <CardHeader
                      title={goal.name}
                      description={`${goal.accountName ?? "Tanpa akun"} · tenggat ${formatDayMonthYear(`${goal.deadline}T00:00:00`)}`}
                      icon={<PiggyBank className="size-5" />}
                      action={<Badge tone={meta.tone}>{meta.label}</Badge>}
                    />
                    <CardBody className="space-y-3">
                      <div className="flex items-baseline justify-between gap-3">
                        <span className="financial text-xl font-bold">{formatIDR(goal.saved)}</span>
                        <span className="flex items-baseline gap-2 text-sm text-muted">
                          <span>dari {formatIDR(goal.targetAmount)}</span>
                          <span
                            className={`financial font-bold ${
                              goal.percent >= 1
                                ? "text-income"
                                : goal.percent >= 0.8
                                  ? "text-warning"
                                  : "text-primary"
                            }`}
                          >
                            {(goal.percent * 100).toFixed(1)}%
                          </span>
                        </span>
                      </div>
                      <Progress
                        value={goal.percent}
                        tone={toneForPercent(goal.percent)}
                        label={`Progres ${goal.name}`}
                      />
                      <div className="flex flex-wrap justify-between gap-x-4 gap-y-1 text-xs text-muted">
                        <span>Sisa {formatIDR(goal.remaining)}</span>
                        <span>
                          {goal.monthsLeft} bulan lagi · butuh {formatIDR(goal.requiredMonthly)}/bulan
                        </span>
                      </div>
                      {goal.projection ? (
                        <p className="text-xs text-muted">
                          Simulasi {formatIDR(goal.projection.monthlyContribution)}/bulan →{" "}
                          {goal.projection.monthsToTarget} bulan untuk tercapai;{" "}
                          {goal.projection.achievableByDeadline
                            ? "tepat waktu sebelum tenggat."
                            : "tidak cukup sebelum tenggat."}
                        </p>
                      ) : null}
                      <div className="flex gap-2">
                        <Button size="sm" variant="secondary" onClick={() => openEdit(goal)}>
                          Ubah
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => archive(goal)}
                          loading={action.pending}
                        >
                          Arsipkan
                        </Button>
                      </div>
                    </CardBody>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Mounted only while open: the sheet seeds its defaults from the clock,
          which must not run during static prerendering. */}
      {sheetOpen ? (
        <GoalSheet
          key={editing?.id ?? "new"}
          open
          onClose={() => setSheetOpen(false)}
          accounts={accounts.data ?? []}
          initial={editing}
          onSaved={(message) => {
            setSheetOpen(false);
            setNotice(message);
            goals.reload();
            accounts.reload();
          }}
        />
      ) : null}
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wide text-muted-dim">{label}</dt>
      <dd className="financial font-medium">{value}</dd>
    </div>
  );
}

/** Desktop KPI tile for the four-up savings summary strip (design's top row). */
function SavingsStat({
  label,
  value,
  meta,
  accent = false,
}: {
  label: string;
  value: string;
  meta?: React.ReactNode;
  accent?: boolean;
}) {
  return (
    <div className="rounded-xl border border-line bg-surface px-4 py-4">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted">{label}</p>
      <p
        className={`financial mt-2 text-xl font-bold ${accent ? "text-primary" : "text-foreground"}`}
      >
        {value}
      </p>
      {meta ? <div className="mt-2 text-xs text-muted">{meta}</div> : null}
    </div>
  );
}

function GoalSheet({
  open,
  onClose,
  accounts,
  initial,
  onSaved,
}: {
  open: boolean;
  onClose: () => void;
  accounts: Account[];
  initial: SavingsGoal | null;
  onSaved: (message: string) => void;
}) {
  const action = useAction();
  const isEdit = initial !== null;
  const [name, setName] = useState(initial?.name ?? "");
  const [targetRaw, setTargetRaw] = useState(initial ? String(initial.targetAmount) : "");
  const [deadline, setDeadline] = useState(initial?.deadline ?? localDate(90));
  const [accountId, setAccountId] = useState(initial?.accountId ?? "");
  const [localError, setLocalError] = useState<string | null>(null);

  // The account list can still be loading when the sheet mounts, so fall back
  // to the first account instead of showing an empty selection.
  const effectiveAccount = accountId || accounts[0]?.id || "";

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setLocalError(null);

    const target = parseAmountInput(targetRaw);
    if (!name.trim()) {
      setLocalError("Nama target wajib diisi.");
      return;
    }
    if (target === null) {
      setLocalError("Nominal target tidak valid. Contoh: 5000000 atau 5jt.");
      return;
    }
    if (!deadline) {
      setLocalError("Tenggat wajib diisi.");
      return;
    }
    if (!effectiveAccount) {
      setLocalError("Pilih akun yang ditautkan.");
      return;
    }

    if (isEdit) {
      const result = await action.run(() =>
        api.savings.update(initial.id, {
          version: initial.version,
          name: name.trim(),
          targetAmount: target,
          deadline,
          accountId: effectiveAccount,
        }),
      );
      if (result) onSaved(`Target "${result.name}" diperbarui.`);
      return;
    }

    const result = await action.run(() =>
      api.savings.create({
        name: name.trim(),
        targetAmount: target,
        deadline,
        accountId: effectiveAccount,
      }),
    );
    if (result) onSaved(`Target "${result.name}" dibuat.`);
  }

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title={isEdit ? "Ubah target" : "Target baru"}
      description="Progres dihitung dari saldo akun yang ditautkan."
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="Nama target" htmlFor="goal-name">
          <Input
            id="goal-name"
            placeholder="Liburan akhir tahun"
            value={name}
            onChange={(event) => setName(event.target.value)}
            maxLength={120}
          />
        </Field>

        <Field label="Nominal target" hint="Contoh: 5jt" htmlFor="goal-target">
          <Input
            id="goal-target"
            inputMode="decimal"
            placeholder="5000000"
            value={targetRaw}
            onChange={(event) => setTargetRaw(event.target.value)}
            className="financial"
          />
        </Field>

        <Field label="Tenggat" hint="Harus setelah hari ini" htmlFor="goal-deadline">
          <Input
            id="goal-deadline"
            type="date"
            min={localDate(1)}
            value={deadline}
            onChange={(event) => setDeadline(event.target.value)}
          />
        </Field>

        <Field label="Akun yang ditautkan" hint="Sumber progres" htmlFor="goal-account">
          <Select
            id="goal-account"
            value={effectiveAccount}
            onChange={(event) => setAccountId(event.target.value)}
          >
            <option value="">Pilih akun</option>
            {accounts.map((account) => (
              <option key={account.id} value={account.id}>
                {account.name} ({formatIDR(account.balance)})
              </option>
            ))}
          </Select>
        </Field>

        {accounts.length === 0 ? (
          <Alert tone="info">Buat akun dulu di halaman Profil sebelum menautkan target.</Alert>
        ) : null}
        {localError ? <Alert tone="warning">{localError}</Alert> : null}
        {action.error ? <Alert tone="error">{action.error}</Alert> : null}

        <div className="flex gap-2">
          <Button variant="ghost" onClick={onClose} disabled={action.pending}>
            Batal
          </Button>
          <Button type="submit" loading={action.pending} className="flex-1">
            {isEdit ? "Simpan perubahan" : "Simpan target"}
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
