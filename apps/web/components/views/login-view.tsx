"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Eye, EyeOff, LogIn } from "lucide-react";

import { signIn, signUp, useToken, SessionError } from "@/lib/session";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/field";
import { Alert } from "@/components/ui/feedback";

/**
 * Kata sandi dengan tombol mata, supaya pengguna bisa memeriksa apa yang sedang
 * diketik. Tombolnya `type="button"` agar tidak ikut meng-submit form, punya
 * `aria-label` dan `aria-pressed` untuk pembaca layar, dan tingginya mengikuti
 * input (44px) agar tetap nyaman disentuh.
 */
function PasswordInput({
  id,
  autoComplete,
  value,
  onChange,
}: {
  id: string;
  autoComplete: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <Input
        id={id}
        type={visible ? "text" : "password"}
        autoComplete={autoComplete}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="pr-11"
      />
      <button
        type="button"
        onClick={() => setVisible((current) => !current)}
        aria-label={visible ? "Sembunyikan kata sandi" : "Lihat kata sandi"}
        aria-pressed={visible}
        className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-muted transition-colors hover:text-foreground"
      >
        {visible ? (
          <EyeOff className="size-4" aria-hidden />
        ) : (
          <Eye className="size-4" aria-hidden />
        )}
      </button>
    </div>
  );
}

/**
 * Masuk / daftar dengan auth lokal (ISSUE-15): kredensial dikirim ke API
 * FISCA sendiri, bukan ke penyedia pihak ketiga. Tidak ada lagi jalur tempel
 * token dev — backend bahkan tidak menerima token `dev:<uuid>` lagi.
 *
 * Pendaftaran meminta nama depan & belakang karena nama itulah yang dipakai
 * aplikasi menyapa pengguna ("Halo, Raditya"). Nama belakang bersifat opsional
 * agar pengguna bernama satu kata tidak terjebak; backend turut menerima
 * `displayName` kosong dan mengisinya dari bagian sebelum "@" pada email.
 */
export function LoginView() {
  const router = useRouter();
  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const token = useToken();
  const alreadySignedIn = token.length > 0;

  function goHome() {
    router.push("/home");
    router.refresh();
  }

  function toggleMode() {
    setMode(mode === "signin" ? "signup" : "signin");
    setError(null);
    setNotice(null);
    // Kata sandi tidak dipindahkan antar mode: yang satu adalah kredensial
    // lama, yang lain baru dibuat — mencampurnya hanya membingungkan.
    setPassword("");
    setConfirmPassword("");
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setNotice(null);

    if (!email.trim() || !password) {
      setError("Email dan kata sandi wajib diisi.");
      return;
    }
    if (mode === "signup") {
      if (!firstName.trim()) {
        setError("Nama depan wajib diisi, karena dipakai untuk menyapa Anda.");
        return;
      }
      if (password !== confirmPassword) {
        setError("Kata sandi tidak sama. Ketik ulang dengan hati-hati.");
        return;
      }
      if (password.length < 8) {
        setError("Kata sandi minimal 8 karakter.");
        return;
      }
    }

    setPending(true);
    try {
      if (mode === "signin") {
        await signIn(email.trim(), password);
      } else {
        const displayName = [firstName.trim(), lastName.trim()].filter(Boolean).join(" ");
        const message = await signUp(email.trim(), password, displayName);
        if (message) {
          setNotice(message);
          return;
        }
      }
      goHome();
    } catch (err) {
      setError(
        err instanceof SessionError
          ? err.message
          : "Autentikasi gagal. Coba lagi sebentar lagi.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="flex min-h-dvh items-center justify-center bg-canvas px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-primary to-primary-deep text-sm font-extrabold text-on-primary">
            F
          </span>
          <div>
            <p className="text-lg font-extrabold tracking-tight">FISCA</p>
            <p className="text-xs text-muted">Keuangan pribadi yang lebih jernih</p>
          </div>
        </div>

        <div className="glass-panel rounded-xl px-5 py-6">
          <h1 className="text-xl font-bold">
            {mode === "signin" ? "Masuk" : "Buat akun"}
          </h1>
          <p className="mt-1 text-sm text-muted">
            {mode === "signin"
              ? "Lanjutkan dengan email Anda."
              : "Daftar dengan nama, email, dan kata sandi."}
          </p>

          {alreadySignedIn ? (
            <div className="mt-4">
              <Alert tone="info" title="Anda sudah masuk">
                Kembali ke aplikasi, atau keluar dulu dari Profil sebelum ganti akun.
              </Alert>
            </div>
          ) : null}

          {error ? (
            <div className="mt-4">
              <Alert tone="error">{error}</Alert>
            </div>
          ) : null}
          {notice ? (
            <div className="mt-4">
              <Alert tone="success">{notice}</Alert>
            </div>
          ) : null}

          <form onSubmit={submit} className="mt-5 space-y-4">
            {mode === "signup" ? (
              <>
                <Field label="Nama depan" htmlFor="signup-firstname">
                  <Input
                    id="signup-firstname"
                    type="text"
                    autoComplete="given-name"
                    placeholder="Raditya"
                    value={firstName}
                    onChange={(event) => setFirstName(event.target.value)}
                  />
                </Field>
                <Field label="Nama belakang" htmlFor="signup-lastname" hint="Opsional">
                  <Input
                    id="signup-lastname"
                    type="text"
                    autoComplete="family-name"
                    placeholder="Pratama"
                    value={lastName}
                    onChange={(event) => setLastName(event.target.value)}
                  />
                </Field>
              </>
            ) : null}

            <Field label="Email" htmlFor="login-email">
              <Input
                id="login-email"
                type="email"
                autoComplete="email"
                placeholder="nama@email.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </Field>

            <Field
              label="Kata sandi"
              htmlFor="login-password"
              hint={mode === "signin" ? undefined : "Minimal 8 karakter"}
            >
              <PasswordInput
                id="login-password"
                autoComplete={mode === "signin" ? "current-password" : "new-password"}
                value={password}
                onChange={setPassword}
              />
            </Field>

            {mode === "signup" ? (
              <Field label="Ulangi kata sandi" htmlFor="signup-confirm">
                <PasswordInput
                  id="signup-confirm"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={setConfirmPassword}
                />
              </Field>
            ) : null}

            <Button
              type="submit"
              loading={pending}
              className="w-full"
              icon={<LogIn className="size-4" aria-hidden />}
            >
              {mode === "signin" ? "Masuk" : "Daftar"}
            </Button>
            <button
              type="button"
              onClick={toggleMode}
              className="w-full py-3 text-center text-sm text-primary hover:underline"
            >
              {mode === "signin" ? "Belum punya akun? Daftar" : "Sudah punya akun? Masuk"}
            </button>
          </form>
        </div>

        <p className="mt-5 text-center text-xs text-muted">
          Data keuangan Anda tidak pernah ditulis model AI.
        </p>
      </div>
    </main>
  );
}
