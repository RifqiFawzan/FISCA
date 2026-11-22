"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { AssistantActionButton } from "@/components/layout/assistant-action-button";
import { cn } from "@/lib/cn";
import { isActivePath, mobileNavItems } from "@/lib/navigation";

/** Fixed bottom navigation, mobile only (DESIGN.md §7.1). */
export function MobileBottomNavigation() {
  const pathname = usePathname();

  return (
    <nav
      aria-label="Navigasi utama"
      className="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-background/95 backdrop-blur-xl md:hidden"
    >
      <div className="pb-safe mx-auto flex max-w-md items-end justify-between gap-1 px-3 pt-2">
        {mobileNavItems.map((item) => {
          const active = isActivePath(pathname, item.href);

          if (item.primary) {
            return (
              <AssistantActionButton key={item.href} active={active} />
            );
          }

          const Icon = item.icon;

          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex min-w-0 flex-1 flex-col items-center gap-1 rounded-md px-1 py-1.5 transition-colors",
                active
                  ? "text-primary"
                  : "text-muted hover:text-foreground",
              )}
            >
              <Icon
                className="size-5"
                strokeWidth={active ? 2.4 : 1.9}
                aria-hidden
              />
              <span
                className={cn(
                  "truncate text-[11px] leading-4",
                  active ? "font-semibold" : "font-medium",
                )}
              >
                {item.label}
              </span>
              <span
                aria-hidden
                className={cn(
                  "h-0.5 w-4 rounded-full",
                  active ? "bg-primary" : "bg-transparent",
                )}
              />
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
