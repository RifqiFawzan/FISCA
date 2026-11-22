export default function AssistantLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Assistant is a dedicated full-screen workspace: no AppShell, no bottom nav
  // (DESIGN.md §5.1, §7.1, §8.4).
  return <div className="min-h-dvh bg-background">{children}</div>;
}
