"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { MessagesSquare } from "lucide-react";

import { cn } from "@/lib/cn";
import { desktopNavItems, isActivePath } from "@/lib/navigation";

/**
 * Persistent desktop sidebar. Collapses to an icon rail on tablet (DESIGN.md
 * §5.2/§5.3) and expands at >= 1024px.
 */
export function DesktopSidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-[76px] flex-col border-r border-line bg-background md:flex lg:w-[248px]">
      <div className="flex h-16 shrink-0 items-center px-2 lg:px-5">
        <Link
          href="/home"
          aria-label="F FISCA · Beranda"
          className="flex items-center gap-3 rounded-md"
        >
          <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-primary to-primary-deep text-sm font-extrabold text-on-primary">
            F
          </span>{" "}
          <span className="hidden text-lg font-extrabold tracking-tight lg:block">
            FISCA
          </span>
        </Link>
      </div>

      <nav
        aria-label="Navigasi samping"
        className="flex flex-1 flex-col gap-1 overflow-y-auto px-2 pt-4 lg:px-3"
      >
        {desktopNavItems.map((item) => {
          const active = isActivePath(pathname, item.href);
          const Icon = item.icon;

          return (
            <Link
              key={item.href}
              href={item.href}
              title={item.label}
              aria-current={active ? "page" : undefined}
              className={cn(
                "group relative flex items-center justify-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors lg:justify-start",
                active
                  ? "bg-surface-high text-foreground"
                  : "text-muted hover:bg-surface-low hover:text-foreground",
              )}
            >
              {active ? (
                <span
                  aria-hidden
                  className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary"
                />
              ) : null}
              <Icon
                className="size-5 shrink-0"
                strokeWidth={active ? 2.4 : 1.9}
                aria-hidden
              />
              <span
                className={cn(
                  "hidden truncate lg:block",
                  active ? "font-semibold" : "font-medium",
                )}
              >
                {item.label}
              </span>
            </Link>
          );
        })}
      </nav>

      <div className="shrink-0 p-2 lg:p-4">
        <Link
          href="/assistant"
          title="Tanya Asisten"
          aria-current={
            pathname.startsWith("/assistant") ? "page" : undefined
          }
          className="flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-3 text-sm font-semibold text-on-primary shadow-glow-primary transition-colors hover:bg-primary-hover lg:justify-start"
        >
          <MessagesSquare className="size-5 shrink-0" aria-hidden />
          <span className="hidden lg:inline">Tanya Asisten</span>
        </Link>
      </div>
    </aside>
  );
}
