import { DesktopSidebar } from "@/components/layout/desktop-sidebar";
import { MobileBottomNavigation } from "@/components/layout/mobile-bottom-navigation";

/** Application shell for every non-assistant route (DESIGN.md §5, §7). */
export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-dvh bg-canvas">
      <DesktopSidebar />

      <div className="md:pl-[76px] lg:pl-[248px]">
        <main className="mx-auto w-full max-w-[1600px] px-4 pt-6 pb-28 md:px-6 md:pt-8 md:pb-12 lg:px-8">
          {children}
        </main>
      </div>

      <MobileBottomNavigation />
    </div>
  );
}
