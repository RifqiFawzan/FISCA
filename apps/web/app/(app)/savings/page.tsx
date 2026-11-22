import type { Metadata } from "next";

import { SavingsView } from "@/components/views/savings-view";

export const metadata: Metadata = { title: "Tabungan" };

export default function SavingsPage() {
  return <SavingsView />;
}
