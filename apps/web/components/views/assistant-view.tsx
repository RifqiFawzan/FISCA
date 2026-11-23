"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { MessagesSquare, Mic, Plus, Send, Square, Trash2, UserRound, X } from "lucide-react";

import { api, errorMessage, hasToken } from "@/lib/api";
import type { AssistantMessage, Conversation } from "@/lib/api";
import { useApi } from "@/lib/use-api";
import { relativeTime } from "@/lib/format";
import { BackButton } from "@/components/layout/back-button";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";
import { Alert, Spinner } from "@/components/ui/feedback";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { AssistantCardView } from "@/components/assistant/assistant-card";

const SUGGESTIONS = [
  "catat makan siang 25rb",
  "berapa pengeluaran bulan ini",
  "cari transaksi kopi",
  "transaksi paling besar bulan ini",
  "cek budget bulan ini",
];

export function AssistantView() {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [recording, setRecording] = useState(false);
  const [transcribing, setTranscribing] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  // Pesan yang sedang "di-stream" (penyajian kata demi kata dari balasan yang
  // sudah lengkap dari server — lihat design-refs/motion-streaming-reply.html).
  const [streamId, setStreamId] = useState<string | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);

  const conversations = useApi(() => api.assistant.conversations(30), "assistant:conversations");
  const messages = useApi(
    () => (activeId ? api.assistant.conversation(activeId) : Promise.resolve([] as AssistantMessage[])),
    `assistant:messages:${activeId ?? "none"}`,
  );

  async function send(text: string, source: "text" | "voice" = "text") {
    const trimmed = text.trim();
    if (!trimmed || sending) return;
    setSending(true);
    setError(null);
    setInput("");
    try {
      const reply = await api.assistant.send({
        text: trimmed,
        conversationId: activeId ?? undefined,
        source,
      });
      setActiveId(reply.conversationId);
      // Streaming hanya diputar sekali saat pesan tiba; streamId dibersihkan
      // oleh StreamingReply sendiri setelah kata terakhir selesai animasi.
      setStreamId(reply.messageId);
      messages.reload();
      conversations.reload();
    } catch (err) {
      setError(errorMessage(err));
      setInput(trimmed);
    } finally {
      setSending(false);
    }
  }

  async function toggleRecording() {
    if (recording) {
      recorderRef.current?.stop();
      return;
    }
    setError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream);
      chunksRef.current = [];
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) chunksRef.current.push(event.data);
      };
      recorder.onstop = async () => {
        stream.getTracks().forEach((track) => track.stop());
        setRecording(false);
        setTranscribing(true);
        try {
          const blob = new Blob(chunksRef.current, {
            type: recorder.mimeType || "audio/webm",
          });
          const result = await api.speech.transcribe(blob);
          if (result.transcript) {
            setInput(result.transcript);
            setError(null);
          } else {
            setError("Rekaman tidak menghasilkan teks. Coba lagi atau ketik pesan.");
          }
        } catch (err) {
          setError(errorMessage(err));
        } finally {
          setTranscribing(false);
        }
      };
      recorderRef.current = recorder;
      recorder.start();
      setRecording(true);
    } catch {
      setError("Mikrofon tidak tersedia atau izin ditolak. Ketik pesan sebagai gantinya.");
    }
  }

  async function removeConversation(conversation: Conversation) {
    try {
      await api.assistant.deleteConversation(conversation.id);
      if (activeId === conversation.id) setActiveId(null);
      conversations.reload();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (!hasToken()) {
    return (
      <div className="flex min-h-dvh flex-col">
        <AssistantHeader />
        <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-8">
          <Alert tone="warning" title="Belum ada token akses">
            <Link href="/login" className="underline">
              Masuk
            </Link>{" "}
            untuk menggunakan asisten.
          </Alert>
        </main>
      </div>
    );
  }

  const history = messages.data ?? [];

  return (
    <div className="flex min-h-dvh flex-col">
      <AssistantHeader
        onNew={() => {
          setActiveId(null);
          setSidebarOpen(false);
        }}
        onToggleSidebar={() => setSidebarOpen((open) => !open)}
        sidebarOpen={sidebarOpen}
      />

      <div className="mx-auto flex w-full max-w-6xl flex-1 overflow-hidden">
        <aside
          className={cn(
            "w-full shrink-0 flex-col border-r border-line bg-background md:flex md:w-64 lg:w-72",
            sidebarOpen ? "flex" : "hidden",
          )}
        >
          <div className="border-b border-line p-3">
            <Button
              variant="secondary"
              className="w-full"
              icon={<Plus className="size-4" aria-hidden />}
              onClick={() => {
                setActiveId(null);
                setSidebarOpen(false);
              }}
            >
              Percakapan baru
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            {conversations.loading ? (
              <div className="p-3"><Spinner /></div>
            ) : (conversations.data ?? []).length === 0 ? (
              <p className="px-3 py-4 text-xs text-muted">Belum ada percakapan.</p>
            ) : (
              <ul className="space-y-1">
                {(conversations.data ?? []).map((conversation) => (
                  <li key={conversation.id} className="group flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => {
                        setActiveId(conversation.id);
                        setSidebarOpen(false);
                      }}
                      className={cn(
                        "min-w-0 flex-1 rounded-lg px-3 py-2 text-left text-sm transition-colors",
                        activeId === conversation.id
                          ? "bg-surface-high text-foreground"
                          : "text-muted hover:bg-surface-low hover:text-foreground",
                      )}
                    >
                      <span className="block truncate font-medium">{conversation.title ?? "Percakapan"}</span>
                      <span className="block truncate text-[11px] text-muted-dim">
                        {relativeTime(conversation.lastMessageAt)}
                      </span>
                    </button>
                    <button
                      type="button"
                      aria-label={`Hapus ${conversation.title ?? "percakapan"}`}
                      onClick={() => removeConversation(conversation)}
                      className="grid size-8 shrink-0 place-items-center rounded-md text-muted-dim opacity-0 transition-opacity hover:bg-danger/10 hover:text-danger group-hover:opacity-100 focus:opacity-100"
                    >
                      <Trash2 className="size-3.5" aria-hidden />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col">
          {/* MessageScroller (shadcn): auto-scroll ke anchor + tombol lompat
              ke pesan terbaru, menggantikan bottomRef manual sebelumnya. */}
          <MessageScrollerProvider>
            <MessageScroller className="min-h-0 flex-1 xl:mx-4 xl:my-4 xl:rounded-xl xl:border xl:border-line xl:bg-surface">
              <MessageScrollerViewport className="px-4 py-6">
                <MessageScrollerContent className="mx-auto w-full max-w-2xl">
                  {messages.error ? (
                    <Alert tone="error" title="Percakapan gagal dimuat">{messages.error}</Alert>
                  ) : null}

                  {history.length === 0 && !sending && !activeId ? (
                    <Welcome onPick={(text) => send(text)} />
                  ) : null}

                  {history.map((message) => (
                    <MessageScrollerItem key={message.id}>
                      <MessageBubble
                      message={message}
                      onApproved={messages.reload}
                      streaming={streamId === message.id}
                      onStreamDone={() => setStreamId(null)}
                    />
                    </MessageScrollerItem>
                  ))}

                  {/* Pill "Berpikir…" — status menunggu LLM yang sesungguhnya
                      (request masih berjalan), menggantikan spinner generik. */}
                  {sending ? (
                    <div className="flex items-start gap-2" role="status" aria-live="polite">
                      <span
                        aria-hidden
                        className="hidden size-8 shrink-0 place-items-center rounded-full bg-ai/10 text-ai ring-1 ring-ai/20 xl:grid"
                      >
                        <MessagesSquare className="size-4" />
                      </span>
                      <span className="reply-think rounded-2xl rounded-tl-sm border border-line bg-surface-low px-3.5 py-2.5 text-sm leading-6">
                        Berpikir…
                      </span>
                    </div>
                  ) : null}

                  <MessageScrollerItem scrollAnchor>
                    <div aria-hidden className="h-px w-full" />
                  </MessageScrollerItem>
                </MessageScrollerContent>
              </MessageScrollerViewport>
              <MessageScrollerButton direction="end" size="sm" />
            </MessageScroller>
          </MessageScrollerProvider>

          <div className="pb-safe border-t border-line bg-background px-4 py-3">
            <div className="mx-auto w-full max-w-2xl space-y-2">
              {error ? (
                <Alert tone="error" action={<button type="button" aria-label="Tutup" onClick={() => setError(null)}><X className="size-4" /></button>}>
                  {error}
                </Alert>
              ) : null}

              {history.length > 0 || activeId ? (
                <div className="flex flex-wrap gap-1.5">
                  {SUGGESTIONS.slice(0, 3).map((suggestion) => (
                    <button
                      key={suggestion}
                      type="button"
                      onClick={() => send(suggestion)}
                      disabled={sending}
                      className="rounded-full border border-line bg-surface-low px-3 py-1.5 text-xs text-muted transition-colors hover:border-primary/40 hover:text-foreground disabled:opacity-50"
                    >
                      {suggestion}
                    </button>
                  ))}
                </div>
              ) : null}

              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  send(input);
                }}
                className="flex items-center gap-2 rounded-xl border border-line bg-surface-low px-3 py-2.5 focus-within:border-primary/50"
              >
                <input
                  aria-label="Tulis pesan"
                  placeholder={recording ? "Merekam suara…" : "Tulis: catat kopi 18rb"}
                  value={input}
                  onChange={(event) => setInput(event.target.value)}
                  disabled={recording || transcribing}
                  className="min-w-0 flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-dim focus:outline-none disabled:opacity-60"
                />
                <button
                  type="button"
                  onClick={toggleRecording}
                  disabled={transcribing || sending}
                  aria-label={recording ? "Hentikan rekaman" : "Rekam suara"}
                  className={cn(
                    "grid size-9 shrink-0 place-items-center rounded-lg transition-colors xl:size-11 xl:rounded-xl",
                    recording
                      ? "bg-danger/20 text-danger xl:bg-danger/25"
                      : "text-muted hover:bg-surface-high hover:text-foreground xl:bg-primary xl:text-on-primary xl:shadow-glow-primary xl:hover:bg-primary-hover",
                  )}
                >
                  {transcribing ? (
                    <Spinner />
                  ) : recording ? (
                    <Square className="size-4" aria-hidden />
                  ) : (
                    <Mic className="size-4" aria-hidden />
                  )}
                </button>
                <Button
                  type="submit"
                  size="sm"
                  disabled={!input.trim() || sending || recording || transcribing}
                  loading={sending}
                  icon={<Send className="size-4" aria-hidden />}
                  aria-label="Kirim pesan"
                >
                  <span className="hidden sm:inline">Kirim</span>
                </Button>
              </form>
              <p className="text-center text-[11px] text-muted-dim">
                AI tidak pernah menulis data keuangan langsung. Setiap transaksi divalidasi server sebelum disimpan.
              </p>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

function AssistantHeader({
  onNew,
  onToggleSidebar,
  sidebarOpen,
}: {
  onNew?: () => void;
  onToggleSidebar?: () => void;
  sidebarOpen?: boolean;
}) {
  return (
    <header className="pt-safe sticky top-0 z-30 border-b border-line bg-background">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center gap-2 px-4">
        <BackButton />
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          <span className="grid size-8 shrink-0 place-items-center rounded-full bg-ai/10 text-ai ring-1 ring-ai/20">
            <MessagesSquare className="size-4" aria-hidden />
          </span>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold">Asisten FISCA</p>
            <p className="truncate text-xs text-muted">Catat transaksi &amp; tanya keuangan</p>
          </div>
        </div>
        {onToggleSidebar ? (
          <Button
            size="sm"
            variant="ghost"
            className="md:hidden"
            onClick={onToggleSidebar}
            aria-label={sidebarOpen ? "Tutup daftar percakapan" : "Buka daftar percakapan"}
          >
            Riwayat
          </Button>
        ) : null}
        {onNew ? (
          <Button
            size="sm"
            variant="secondary"
            icon={<Plus className="size-3.5" aria-hidden />}
            onClick={onNew}
          >
            <span className="hidden sm:inline">Baru</span>
          </Button>
        ) : null}
      </div>
    </header>
  );
}

/**
 * Menyajikan balasan yang SUDAH lengkap dari server kata demi kata (pola desain
 * "Streaming Reply" — design-refs/motion-streaming-reply.html): tiap kata muncul
 * naik + tajam selagi bubble memeluk isi yang bertambah. Konten tidak diubah,
 * hanya penyajiannya; prefers-reduced-motion menampilkan teks penuh seketika.
 */
function StreamingReply({ text, onDone }: { text: string; onDone?: () => void }) {
  const words = text.split(" ");
  // Reduced motion: tampil penuh sejak awal (initializer, bukan setState di effect).
  const [shown, setShown] = useState(() =>
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
      ? words.length
      : 0,
  );
  // Sengaja lewat ref (di-update di effect, bukan saat render): identitas
  // onDone berubah tiap render; yang penting callback terbaru tetap dipanggil
  // tepat setelah kata terakhir muncul.
  const doneRef = useRef(onDone);
  useEffect(() => {
    doneRef.current = onDone;
  });

  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      doneRef.current?.();
      return;
    }
    let i = 0;
    const id = window.setInterval(() => {
      i += 1;
      setShown(i);
      if (i >= words.length) {
        window.clearInterval(id);
        // Tunggu animasi kata terakhir (0,4s) sebelum induk mengubah pesan
        // kembali menjadi teks biasa; swap-nya identik jadi tak terlihat.
        window.setTimeout(() => doneRef.current?.(), 600);
      }
    }, 70);
    return () => window.clearInterval(id);
  }, [words.length]);

  return (
    <>
      {words.map((word, index) => (
        <span
          key={index}
          className={index < shown ? "stream-word" : "stream-word-pending"}
        >
          {word}{" "}
        </span>
      ))}
    </>
  );
}

function Welcome({ onPick }: { onPick: (text: string) => void }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center py-12 text-center">
      <span className="grid size-16 place-items-center rounded-full bg-ai/10 text-ai">
        <MessagesSquare className="size-7" aria-hidden />
      </span>
      <h1 className="mt-5 text-xl font-bold">Halo, saya Asisten FISCA</h1>
      <p className="mt-2 max-w-sm text-sm leading-6 text-muted">
        Catat transaksi lewat chat atau suara, tanya ringkasan keuangan, cari riwayat, dan cek budget.
      </p>
      <div className="mt-6 flex w-full max-w-sm flex-wrap justify-center gap-2">
        {SUGGESTIONS.map((suggestion) => (
          <button
            key={suggestion}
            type="button"
            onClick={() => onPick(suggestion)}
            className="rounded-full border border-line bg-surface-low px-3 py-2 text-xs text-muted transition-colors hover:border-primary/40 hover:text-foreground"
          >
            {suggestion}
          </button>
        ))}
      </div>
    </div>
  );
}

function MessageBubble({
  message,
  onApproved,
  streaming = false,
  onStreamDone,
}: {
  message: AssistantMessage;
  onApproved: () => void;
  streaming?: boolean;
  onStreamDone?: () => void;
}) {
  const user = message.role === "user";
  return (
    <div className={cn("flex w-full items-start gap-2", user ? "justify-end" : "justify-start")}>
      {!user ? (
        <span
          aria-hidden
          className="hidden size-8 shrink-0 place-items-center rounded-full bg-ai/10 text-ai ring-1 ring-ai/20 xl:grid"
        >
          <MessagesSquare className="size-4" />
        </span>
      ) : null}
      <div className={cn("w-full max-w-[92%] sm:max-w-[85%]", user && "w-auto")}>
        {user ? (
          <div className="rounded-2xl rounded-br-sm bg-primary px-3.5 py-2.5 text-sm text-on-primary">
            {message.content}
          </div>
        ) : (
          <div>
            {message.content ? (
              // w-fit: bubble memeluk isi dan "tumbuh" mengikuti kata yang muncul
              // saat streaming (design-refs/motion-streaming-reply.html).
              <div className="mb-1 w-fit max-w-full rounded-2xl rounded-tl-sm border border-line bg-surface-low px-3.5 py-2.5 text-sm leading-6 text-foreground">
                {streaming ? (
                  <StreamingReply text={message.content} onDone={onStreamDone} />
                ) : (
                  message.content
                )}
              </div>
            ) : null}
            {message.card ? (
              <AssistantCardView card={message.card} onApproved={onApproved} />
            ) : null}
          </div>
        )}
        <p className={cn("mt-1 text-[10px] text-muted-dim", user ? "text-right" : "text-left")}>
          {relativeTime(message.createdAt)}
        </p>
      </div>
      {user ? (
        <span
          aria-hidden
          className="hidden size-8 shrink-0 place-items-center rounded-full bg-primary text-on-primary xl:grid"
        >
          <UserRound className="size-4" />
        </span>
      ) : null}
    </div>
  );
}
