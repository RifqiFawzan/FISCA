import type { Metadata } from "next";

import { AssistantView } from "@/components/views/assistant-view";

export const metadata: Metadata = { title: "Asisten" };

export default function AssistantPage() {
  return <AssistantView />;
}
