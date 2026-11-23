"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Globe, LogOut, Plus, Tag, Trash2, UserRound, Wallet } from "lucide-react";

import { api } from "@/lib/api";
import { signOut } from "@/lib/session";
import { useAction, useApi } from "@/lib/use-api";
import { formatIDR, formatDayMonthYear } from "@/lib/format";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, RetryButton, Skeleton } from "@/components/ui/feedback";
import { Field, Input, Select } from "@/components/ui/field";
import { Badge } from "@/components/ui/progress";
import { Sheet } from "@/components/ui/sheet";

export function ProfileView() {
  const state = useApi(
    () =>
      Promise.all([
        api.me.get(),
        api.accounts.list(),
        api.categories.list(),
      ]).then(([me, accounts, categories]) => ({ me, accounts, categories })),
    "profile:main",
  );
  const action = useAction();
  const router = useRouter();
  const [accountOpen, setAccountOpen] = useState(false);
  const [categoryOpen, setCategoryOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);

  const data = state.data;

  async function archiveAccount(id: string) {
    const result = await action.run(() => api.accounts.archive(id), "Akun diarsipkan.");
    if (result) state.reload();
  }

  async function setDefault(id: string) {
    const result = await action.run(() => api.accounts.setDefault(id), "Akun utama diperbarui.");
    if (result) state.reload();
  }

  async function archiveCategory(id: string) {
    const result = await action.run(() => api.categories.archive(id), "Kategori diarsipkan.");
    if (result) state.reload();
  }

  return (
    <div className="space-y-5">
      <PageHeader title="Profil & Pengaturan" description="Kelola identitas, akun, kategori, dan akses API." />

      {action.error ? <Alert tone="error">{action.error}</Alert> : null}
      {action.success ? <Alert tone="success">{action.success}</Alert> : null}

      {state.loading && !data ? (
        <div className="space-y-4">
          <Skeleton className="h-36" />
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      ) : state.error && !data ? (
        <Alert tone="error" title="Profil gagal dimuat" action={<RetryButton onClick={state.reload} />}>
          {state.error}
        </Alert>
      ) : data ? (
        <div className="grid grid-cols-1 gap-5 xl:grid-cols-12 xl:items-start">
          <div className="flex min-w-0 flex-col gap-5 xl:col-span-4 xl:col-start-9">
            <Card>
              <CardBody className="flex flex-wrap items-center gap-4">
                <span className="grid size-14 shrink-0 place-items-center rounded-full bg-gradient-to-br from-primary to-primary-deep text-xl font-bold text-on-primary">
                  {data.me.displayName.slice(0, 1).toUpperCase()}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-lg font-semibold">{data.me.displayName}</p>
                  <p className="truncate text-sm text-muted">{data.me.email || "Akun lokal"}</p>
                  <p className="mt-1 flex flex-wrap gap-1.5">
                    <Badge>{data.me.timezone}</Badge>
                    <Badge>{data.me.baseCurrency}</Badge>
                    <Badge tone="primary">{data.me.locale}</Badge>
                  </p>
                </div>
                <Button variant="secondary" icon={<UserRound className="size-4" aria-hidden />} onClick={() => setProfileOpen(true)}>
                  Ubah profil
                </Button>
              </CardBody>
            </Card>

            <Card>
              <CardHeader
                title="Sesi"
                description="Masuk dan keluar dikelola oleh FISCA; sesi dicabut di server."
              />
              <CardBody className="space-y-3">
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    icon={<LogOut className="size-4" aria-hidden />}
                    onClick={async () => {
                      await signOut();
                      router.push("/login");
                      router.refresh();
                    }}
                  >
                    Keluar
                  </Button>
                </div>
                <p className="text-xs leading-5 text-muted">
                  Keluar mencabut sesi di server, sehingga token lama langsung ditolak, bukan
                  sekadar dihapus dari browser ini.
                </p>
              </CardBody>
            </Card>

            <Card className="hidden xl:block">
              <CardHeader
                title="Preferensi regional"
                description="Mata uang, locale, dan zona waktu akun."
                icon={<Globe className="size-5" />}
              />
              <CardBody>
                <dl className="space-y-2.5 text-sm">
                  <div className="flex items-center justify-between gap-3 rounded-lg border border-line bg-surface-low px-3 py-2.5">
                    <dt className="text-muted">Mata uang utama</dt>
                    <dd className="financial font-semibold">{data.me.baseCurrency}</dd>
                  </div>
                  <div className="flex items-center justify-between gap-3 rounded-lg border border-line bg-surface-low px-3 py-2.5">
                    <dt className="text-muted">Format locale</dt>
                    <dd className="font-semibold">{data.me.locale}</dd>
                  </div>
                  <div className="flex items-center justify-between gap-3 rounded-lg border border-line bg-surface-low px-3 py-2.5">
                    <dt className="text-muted">Zona waktu</dt>
                    <dd className="font-semibold">{data.me.timezone}</dd>
                  </div>
                </dl>
              </CardBody>
            </Card>
          </div>

          <div className="flex min-w-0 flex-col gap-5 xl:col-span-8 xl:row-start-1">
            <Card>
              <CardHeader
                title="Akun & Dompet"
                description="Saldo dihitung dari catatan transaksi, tidak bisa diubah manual."
                icon={<Wallet className="size-5" />}
                action={
                  <Button size="sm" variant="secondary" icon={<Plus className="size-3.5" aria-hidden />} onClick={() => setAccountOpen(true)}>
                    Tambah
                  </Button>
                }
              />
              <CardBody>
                <ul className="divide-y divide-line">
                  {data.accounts.map((account) => (
                    <li key={account.id} className="flex items-center gap-3 py-3">
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
                      {/* Dua baris tetap: identitas + saldo di atas, meta +
                          aksi di bawah. Baris tidak pernah membungkus
                          berantakan di layar sempit (antislop layoutmobile). */}
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-3">
                          <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
                            <span className="truncate">{account.name}</span>
                            {account.isDefault ? <Badge tone="primary">Utama</Badge> : null}
                          </p>
                          <span className="financial shrink-0 text-sm font-semibold">
                            {formatIDR(account.balance)}
                          </span>
                        </div>
                        <div className="mt-1 flex items-center justify-between gap-2">
                          <p className="min-w-0 truncate text-xs capitalize text-muted">
                            {account.type} · dibuat {formatDayMonthYear(account.createdAt)}
                          </p>
                          <div className="flex shrink-0 items-center gap-1">
                            {!account.isDefault ? (
                              <Button size="sm" variant="ghost" onClick={() => setDefault(account.id)}>
                                Jadikan utama
                              </Button>
                            ) : null}
                            <Button
                              size="sm"
                              variant="ghost"
                              className="min-w-11"
                              aria-label={`Arsipkan ${account.name}`}
                              icon={<Trash2 className="size-3.5" aria-hidden />}
                              onClick={() => archiveAccount(account.id)}
                            />
                          </div>
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              </CardBody>
            </Card>

            <Card>
              <CardHeader
                title="Kategori"
                description="Dipakai untuk mengelompokkan pemasukan dan pengeluaran."
                icon={<Tag className="size-5" />}
                action={
                  <Button size="sm" variant="secondary" icon={<Plus className="size-3.5" aria-hidden />} onClick={() => setCategoryOpen(true)}>
                    Tambah
                  </Button>
                }
              />
              <CardBody>
                <ul className="grid grid-cols-1 gap-2 md:grid-cols-2">
                  {data.categories.map((category) => (
                    <li
                      key={category.id}
                      className="flex min-w-0 items-center gap-3 rounded-lg border border-line bg-surface-low/60 px-3 py-2.5"
                    >
                      <span
                        aria-hidden
                        className="size-3 shrink-0 rounded-full"
                        style={{ backgroundColor: category.color ?? (category.kind === "income" ? "#34d399" : "#fb7185") }}
                      />
                      <span className="min-w-0 flex-1 truncate text-sm">{category.name}</span>
                      <Badge tone={category.kind === "income" ? "income" : "neutral"}>
                        {category.kind === "income" ? "Pemasukan" : "Pengeluaran"}
                      </Badge>
                      {!category.isDefault ? (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="min-w-11"
                          aria-label={`Arsipkan ${category.name}`}
                          icon={<Trash2 className="size-3.5" aria-hidden />}
                          onClick={() => archiveCategory(category.id)}
                        />
                      ) : null}
                    </li>
                  ))}
                </ul>
              </CardBody>
            </Card>
          </div>
        </div>
      ) : null}

      <AddAccountSheet
        open={accountOpen}
        onClose={() => setAccountOpen(false)}
        onCreated={() => {
          setAccountOpen(false);
          state.reload();
        }}
      />
      <AddCategorySheet
        open={categoryOpen}
        onClose={() => setCategoryOpen(false)}
        onCreated={() => {
          setCategoryOpen(false);
          state.reload();
        }}
      />
      <EditProfileSheet
        key={data?.me.id ?? "profile"}
        open={profileOpen}
        onClose={() => setProfileOpen(false)}
        initial={{
          displayName: data?.me.displayName ?? "",
          timezone: data?.me.timezone ?? "Asia/Jakarta",
          locale: data?.me.locale ?? "id-ID",
        }}
        onSaved={() => {
          setProfileOpen(false);
          state.reload();
        }}
      />
    </div>
  );
}

function AddAccountSheet({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [type, setType] = useState("cash");
  const [openingRaw, setOpeningRaw] = useState("0");
  const create = useAction();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const openingBalance = Number(openingRaw.replace(/[^0-9]/g, "")) || 0;
    const result = await create.run(() =>
      api.accounts.create({ name: name.trim(), type, openingBalance }),
    );
    if (result) onCreated();
  }

  return (
    <Sheet open={open} onClose={onClose} title="Tambah akun" description="Saldo awal bersifat opsional.">
      <form onSubmit={submit} className="space-y-4">
        <Field label="Nama akun" htmlFor="account-name">
          <Input id="account-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Bank BCA" />
        </Field>
        <Field label="Jenis" htmlFor="account-type">
          <Select id="account-type" value={type} onChange={(e) => setType(e.target.value)}>
            <option value="cash">Tunai</option>
            <option value="bank">Bank</option>
            <option value="ewallet">Dompet digital</option>
            <option value="other">Lainnya</option>
          </Select>
        </Field>
        <Field label="Saldo awal" hint="Rupiah" htmlFor="account-opening">
          <Input id="account-opening" inputMode="numeric" value={openingRaw} onChange={(e) => setOpeningRaw(e.target.value)} className="financial" />
        </Field>
        {create.error ? <Alert tone="error">{create.error}</Alert> : null}
        <Button type="submit" loading={create.pending} className="w-full" disabled={!name.trim()}>
          Simpan akun
        </Button>
      </form>
    </Sheet>
  );
}

function AddCategorySheet({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [kind, setKind] = useState<"expense" | "income">("expense");
  const [color, setColor] = useState("#4EDEA3");
  const create = useAction();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const result = await create.run(() =>
      api.categories.create({ name: name.trim(), kind, color }),
    );
    if (result) onCreated();
  }

  return (
    <Sheet open={open} onClose={onClose} title="Tambah kategori">
      <form onSubmit={submit} className="space-y-4">
        <Field label="Nama kategori" htmlFor="category-name">
          <Input id="category-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Langganan" />
        </Field>
        <Field label="Jenis" htmlFor="category-kind">
          <Select id="category-kind" value={kind} onChange={(e) => setKind(e.target.value as "expense" | "income")}>
            <option value="expense">Pengeluaran</option>
            <option value="income">Pemasukan</option>
          </Select>
        </Field>
        <Field label="Warna" htmlFor="category-color">
          <div className="flex items-center gap-3">
            <input
              id="category-color"
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              className="size-11 cursor-pointer rounded-lg border border-line bg-surface-low p-1"
            />
            <span className="financial text-sm text-muted">{color}</span>
          </div>
        </Field>
        {create.error ? <Alert tone="error">{create.error}</Alert> : null}
        <Button type="submit" loading={create.pending} className="w-full" disabled={!name.trim()}>
          Simpan kategori
        </Button>
      </form>
    </Sheet>
  );
}

function EditProfileSheet({
  open,
  onClose,
  initial,
  onSaved,
}: {
  open: boolean;
  onClose: () => void;
  initial: { displayName: string; timezone: string; locale: string };
  onSaved: () => void;
}) {
  const [displayName, setDisplayName] = useState(initial.displayName);
  const [timezone, setTimezone] = useState(initial.timezone);
  const save = useAction();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const result = await save.run(() =>
      api.me.update({ displayName: displayName.trim(), timezone }),
    );
    if (result) onSaved();
  }

  return (
    <Sheet open={open} onClose={onClose} title="Ubah profil">
      <form onSubmit={submit} className="space-y-4">
        <Field label="Nama tampilan" htmlFor="profile-name">
          <Input id="profile-name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        </Field>
        <Field label="Zona waktu" hint="Mempengaruhi periode laporan" htmlFor="profile-tz">
          <Select id="profile-tz" value={timezone} onChange={(e) => setTimezone(e.target.value)}>
            <option value="Asia/Jakarta">Asia/Jakarta (WIB)</option>
            <option value="Asia/Makassar">Asia/Makassar (WITA)</option>
            <option value="Asia/Jayapura">Asia/Jayapura (WIT)</option>
            <option value="UTC">UTC</option>
          </Select>
        </Field>
        {save.error ? <Alert tone="error">{save.error}</Alert> : null}
        <Button type="submit" loading={save.pending} className="w-full" disabled={!displayName.trim()}>
          Simpan perubahan
        </Button>
      </form>
    </Sheet>
  );
}
