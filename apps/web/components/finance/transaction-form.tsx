"use client";

import { useMemo, useState } from "react";

import type { Account, Category, CreateTransactionPayload } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field, Input, Select, Textarea } from "@/components/ui/field";
import { Segmented } from "@/components/ui/segmented";
import { Alert } from "@/components/ui/feedback";
import { formatDateInput, parseAmountInput } from "@/lib/format";

type TxType = "expense" | "income" | "transfer";

/**
 * Manual transaction form. Money is parsed from Indonesian shorthand but the
 * value sent to the API is always an integer rupiah. Nothing is shown as saved
 * until the caller's async submit resolves.
 */
export function TransactionForm({
  accounts,
  categories,
  pending,
  error,
  onSubmit,
  onCancel,
  submitLabel = "Simpan transaksi",
}: {
  accounts: Account[];
  categories: Category[];
  pending?: boolean;
  error?: string | null;
  onSubmit: (payload: CreateTransactionPayload) => void;
  onCancel?: () => void;
  submitLabel?: string;
}) {
  const [type, setType] = useState<TxType>("expense");
  const [amountRaw, setAmountRaw] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [accountId, setAccountId] = useState("");
  const [fromAccountId, setFromAccountId] = useState("");
  const [toAccountId, setToAccountId] = useState("");
  const [date, setDate] = useState(formatDateInput(new Date().toISOString()));
  const [notes, setNotes] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const matchingCategories = useMemo(
    () => categories.filter((category) => category.kind === type),
    [categories, type],
  );

  const defaultAccount = useMemo(
    () => accounts.find((account) => account.isDefault)?.id ?? accounts[0]?.id ?? "",
    [accounts],
  );

  function submit(event: React.FormEvent) {
    event.preventDefault();
    setLocalError(null);

    const amount = parseAmountInput(amountRaw);
    if (amount === null) {
      setLocalError("Nominal tidak valid. Contoh: 25000, 25rb, atau 1,5jt.");
      return;
    }
    if (!description.trim()) {
      setLocalError("Deskripsi wajib diisi.");
      return;
    }

    const payload: CreateTransactionPayload = {
      type,
      amount,
      description: description.trim(),
      source: "manual",
      occurredAt: new Date(`${date}T12:00:00`).toISOString(),
      notes: notes.trim() || undefined,
    };

    if (type === "transfer") {
      const from = fromAccountId || defaultAccount;
      const to = toAccountId;
      if (!from || !to) {
        setLocalError("Pilih akun asal dan akun tujuan.");
        return;
      }
      if (from === to) {
        setLocalError("Akun asal dan tujuan harus berbeda.");
        return;
      }
      payload.fromAccountId = from;
      payload.toAccountId = to;
    } else {
      payload.accountId = accountId || defaultAccount || undefined;
      payload.categoryId = categoryId || undefined;
    }

    onSubmit(payload);
  }

  const accountOptions = accounts.map((account) => (
    <option key={account.id} value={account.id}>
      {account.name}
      {account.isDefault ? " (utama)" : ""}
    </option>
  ));

  return (
    <form onSubmit={submit} className="space-y-4">
      <Segmented<TxType>
        value={type}
        onChange={(next) => {
          setType(next);
          setCategoryId("");
        }}
        options={[
          { value: "expense", label: "Pengeluaran" },
          { value: "income", label: "Pemasukan" },
          { value: "transfer", label: "Transfer" },
        ]}
        className="w-full"
      />

      <Field label="Nominal" hint="Rupiah, bisa 25rb atau 1,5jt" htmlFor="tx-amount">
        <div className="relative">
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-muted">
            Rp
          </span>
          <Input
            id="tx-amount"
            inputMode="decimal"
            autoComplete="off"
            placeholder="25000"
            value={amountRaw}
            onChange={(event) => setAmountRaw(event.target.value)}
            className="financial pl-10"
          />
        </div>
        {amountRaw && parseAmountInput(amountRaw) !== null ? (
          <p className="mt-1 text-[11px] text-muted">
            Akan dicatat sebesar Rp{parseAmountInput(amountRaw)!.toLocaleString("id-ID")}
          </p>
        ) : null}
      </Field>

      <Field label="Deskripsi" htmlFor="tx-description">
        <Input
          id="tx-description"
          placeholder={type === "income" ? "Gaji bulanan" : "Makan siang"}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          maxLength={200}
        />
      </Field>

      {type === "transfer" ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Dari akun" htmlFor="tx-from">
            <Select
              id="tx-from"
              value={fromAccountId || defaultAccount}
              onChange={(event) => setFromAccountId(event.target.value)}
            >
              <option value="">Pilih akun</option>
              {accountOptions}
            </Select>
          </Field>
          <Field label="Ke akun" htmlFor="tx-to">
            <Select
              id="tx-to"
              value={toAccountId}
              onChange={(event) => setToAccountId(event.target.value)}
            >
              <option value="">Pilih akun</option>
              {accountOptions}
            </Select>
          </Field>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Kategori" htmlFor="tx-category">
            <Select
              id="tx-category"
              value={categoryId}
              onChange={(event) => setCategoryId(event.target.value)}
            >
              <option value="">Tanpa kategori</option>
              {matchingCategories.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Akun" htmlFor="tx-account">
            <Select
              id="tx-account"
              value={accountId || defaultAccount}
              onChange={(event) => setAccountId(event.target.value)}
            >
              <option value="">Akun utama</option>
              {accountOptions}
            </Select>
          </Field>
        </div>
      )}

      <Field label="Tanggal" htmlFor="tx-date">
        <Input
          id="tx-date"
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
        />
      </Field>

      <Field label="Catatan" hint="Opsional" htmlFor="tx-notes">
        <Textarea
          id="tx-notes"
          placeholder="Catatan tambahan"
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          maxLength={500}
        />
      </Field>

      {localError ? <Alert tone="warning">{localError}</Alert> : null}
      {error ? <Alert tone="error">{error}</Alert> : null}

      <div className="flex gap-2 pt-1">
        {onCancel ? (
          <Button variant="ghost" onClick={onCancel} disabled={pending}>
            Batal
          </Button>
        ) : null}
        <Button type="submit" loading={pending} className="flex-1">
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}
