"use client";

import { useEffect, useMemo, useRef, useState } from "react";

/**
 * Menyajikan balasan asisten sebagai markdown ringkas yang aman: hanya
 * paragraf, daftar "- " / "1. ", dan **tebal**. Teks selalu dirender sebagai
 * node React (tanpa dangerouslySetInnerHTML), jadi tidak ada injeksi HTML.
 *
 * Saat streaming, kata demi kata diungkap DI DALAM strukturnya (pola desain
 * "Streaming Reply" - design-refs/motion-streaming-reply.html): daftar dan
 * tebal langsung terlihat rapi, sintaks markdown tidak pernah tampil mentah.
 * prefers-reduced-motion menampilkan isi penuh seketika. Setelah onDone,
 * induk mengganti pesan ke mode statis (tanpa span animasi) sehingga riwayat
 * tidak ikut teranimasi ulang.
 */

type Piece = { text: string; bold: boolean };
type Block =
  | { kind: "p"; pieces: Piece[] }
  | { kind: "ul"; items: Piece[][] }
  | { kind: "ol"; items: Piece[][] };

const UNORDERED = /^(?:-|\u2022|\*)\s+/;
const ORDERED = /^\d+\.\s+/;

function parseBold(line: string): Piece[] {
  const pieces: Piece[] = [];
  const parts = line.split(/\*\*([^*]+)\*\*/g);
  parts.forEach((part, index) => {
    if (part) pieces.push({ text: part, bold: index % 2 === 1 });
  });
  return pieces;
}

function parseReply(raw: string): Block[] {
  const blocks: Block[] = [];
  let para: string[] = [];
  let list: { ordered: boolean; items: Piece[][] } | null = null;

  const flushPara = () => {
    if (para.length > 0) {
      blocks.push({ kind: "p", pieces: parseBold(para.join(" ")) });
      para = [];
    }
  };
  const flushList = () => {
    if (list) {
      blocks.push(
        list.ordered ? { kind: "ol", items: list.items } : { kind: "ul", items: list.items },
      );
      list = null;
    }
  };

  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      flushPara();
      flushList();
      continue;
    }
    const ordered = ORDERED.test(trimmed);
    const unordered = !ordered && UNORDERED.test(trimmed);
    if (ordered || unordered) {
      flushPara();
      if (list && list.ordered !== ordered) flushList();
      if (!list) list = { ordered, items: [] };
      const content = ordered
        ? trimmed.replace(ORDERED, "")
        : trimmed.replace(UNORDERED, "");
      list.items.push(parseBold(content));
      continue;
    }
    flushList();
    para.push(trimmed);
  }
  flushPara();
  flushList();
  return blocks;
}

function countWords(blocks: Block[]): number {
  let count = 0;
  const add = (pieces: Piece[]) => {
    for (const piece of pieces) {
      count += piece.text.split(/\s+/).filter(Boolean).length;
    }
  };
  for (const block of blocks) {
    if (block.kind === "p") add(block.pieces);
    else for (const item of block.items) add(item);
  }
  return count;
}

export function RichReply({
  text,
  streaming = false,
  onDone,
}: {
  text: string;
  streaming?: boolean;
  onDone?: () => void;
}) {
  const blocks = useMemo(() => parseReply(text), [text]);
  const total = useMemo(() => countWords(blocks), [blocks]);
  const [shown, setShown] = useState(() => {
    if (!streaming) return Number.MAX_SAFE_INTEGER;
    if (
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    ) {
      return Number.MAX_SAFE_INTEGER;
    }
    return 0;
  });

  // Sengaja lewat ref (di-update di effect): identitas onDone berubah tiap
  // render; callback terbaru tetap dipanggil tepat setelah kata terakhir.
  const doneRef = useRef(onDone);
  useEffect(() => {
    doneRef.current = onDone;
  });

  useEffect(() => {
    if (!streaming) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      doneRef.current?.();
      return;
    }
    let i = 0;
    const id = window.setInterval(() => {
      i += 1;
      setShown(i);
      if (i >= total) {
        window.clearInterval(id);
        // Tunggu animasi kata terakhir (0,4s) sebelum induk menukar pesan
        // ke mode statis; isi dan tata letaknya identik jadi tak terlihat.
        window.setTimeout(() => doneRef.current?.(), 600);
      }
    }, 70);
    return () => window.clearInterval(id);
  }, [streaming, total]);

  if (!streaming || shown >= total) {
    return <StaticReply blocks={blocks} />;
  }

  let index = 0;
  const renderWords = (pieces: Piece[], keyBase: string) =>
    pieces.map((piece, pieceIndex) =>
      piece.text
        .split(/\s+/)
        .filter(Boolean)
        .map((word) => {
          const current = index++;
          return (
            <span
              key={`${keyBase}-${pieceIndex}-${current}`}
              className={current < shown ? "stream-word" : "stream-word-pending"}
            >
              {word}{" "}
            </span>
          );
        }),
    );

  return (
    <div className="space-y-2">
      {blocks.map((block, blockIndex) => {
        if (block.kind === "p") {
          return (
            <p key={blockIndex} className="leading-6">
              {renderWords(block.pieces, `p${blockIndex}`)}
            </p>
          );
        }
        const Tag = block.kind === "ul" ? "ul" : "ol";
        return (
          <Tag
            key={blockIndex}
            className={
              block.kind === "ul"
                ? "list-disc space-y-1 pl-5"
                : "list-decimal space-y-1 pl-5"
            }
          >
            {block.items.map((item, itemIndex) => (
              <li key={itemIndex} className="leading-6">
                {renderWords(item, `b${blockIndex}-${itemIndex}`)}
              </li>
            ))}
          </Tag>
        );
      })}
    </div>
  );
}

function StaticReply({ blocks }: { blocks: Block[] }) {
  const renderPieces = (pieces: Piece[], keyBase: string) =>
    pieces.map((piece, index) =>
      piece.bold ? (
        <strong key={`${keyBase}-${index}`} className="font-semibold">
          {piece.text}
        </strong>
      ) : (
        <span key={`${keyBase}-${index}`}>{piece.text}</span>
      ),
    );

  return (
    <div className="space-y-2">
      {blocks.map((block, blockIndex) => {
        if (block.kind === "p") {
          return (
            <p key={blockIndex} className="leading-6">
              {renderPieces(block.pieces, `p${blockIndex}`)}
            </p>
          );
        }
        const Tag = block.kind === "ul" ? "ul" : "ol";
        return (
          <Tag
            key={blockIndex}
            className={
              block.kind === "ul"
                ? "list-disc space-y-1 pl-5"
                : "list-decimal space-y-1 pl-5"
            }
          >
            {block.items.map((item, itemIndex) => (
              <li key={itemIndex} className="leading-6">
                {renderPieces(item, `b${blockIndex}-${itemIndex}`)}
              </li>
            ))}
          </Tag>
        );
      })}
    </div>
  );
}
