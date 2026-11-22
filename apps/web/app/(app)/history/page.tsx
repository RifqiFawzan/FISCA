import type { Metadata } from "next";

import { HistoryView } from "@/components/views/history-view";

export const metadata: Metadata = { title: "Riwayat" };

export default function HistoryPage() {
  return <HistoryView />;
}
