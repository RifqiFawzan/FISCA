"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { MessagesSquare } from "lucide-react";

import { cn } from "@/lib/cn";
import { desktopNavGroups, isActivePath, type NavItem } from "@/lib/navigation";
import { useIsDesktop } from "@/lib/use-desktop";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

/**
 * Persistent desktop sidebar. Icon rail at md, labeled at lg (DESIGN.md
 * §5.2/§5.3). Destinations sit in three groups under quiet labels so six
 * items scan as clusters instead of one flat list; in the icon-rail state
 * every row keeps its name through a shadcn tooltip that opens on hover and
 * on keyboard focus.
 */
export function DesktopSidebar() {
  const pathname = usePathname();
  // The label column appears at lg (1024px); below that rows are icon-only.
  const labelsVisible = useIsDesktop(1024);

  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-[76px] flex-col border-r border-line bg-background md:flex lg:w-[248px]">
      <div className="flex h-16 shrink-0 items-center px-2 lg:px-5">
        <Link
          href="/home"
          aria-label="F FISCA · Beranda"
          className="flex items-center gap-3 rounded-md"
        >
          <span
            aria-hidden
            className="grid size-9 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-primary to-primary-deep text-sm font-extrabold text-on-primary"
          >
            F
          </span>
          <span className="hidden text-lg font-extrabold tracking-tight lg:block">
            FISCA
          </span>
        </Link>
      </div>

      <TooltipProvider delay={200}>
        <nav
          aria-label="Navigasi samping"
          className="flex flex-1 flex-col overflow-y-auto px-2 pt-2 pb-4 scrollbar-thin lg:px-3"
        >
          {desktopNavGroups.map((group, index) => (
            <div
              key={group.label}
              className={cn(
                "flex flex-col gap-1",
                // Hairline between clusters. It also structures the icon rail,
                // where the group labels themselves are hidden for width.
                index > 0 && "mt-4 border-t border-line pt-4",
              )}
            >
              <p className="hidden px-3 pb-1 text-xs font-semibold text-muted lg:block">
                {group.label}
              </p>
              {group.items.map((item) => (
                <SidebarLink
                  key={item.href}
                  item={item}
                  active={isActivePath(pathname, item.href)}
                  labelsVisible={labelsVisible}
                />
              ))}
            </div>
          ))}
        </nav>

        <div className="shrink-0 border-t border-line p-2 lg:p-4">
          <AssistCta
            active={pathname.startsWith("/assistant")}
            labelsVisible={labelsVisible}
          />
        </div>
      </TooltipProvider>
    </aside>
  );
}

function SidebarLink({
  item,
  active,
  labelsVisible,
}: {
  item: NavItem;
  active: boolean;
  labelsVisible: boolean;
}) {
  const Icon = item.icon;
  const link = (
    <Link
      href={item.href}
      aria-label={item.label}
      aria-current={active ? "page" : undefined}
      className={cn(
        "relative flex items-center justify-center gap-3 rounded-lg px-3 py-3 text-sm transition-colors lg:justify-start",
        active
          ? "bg-surface-high font-semibold text-foreground"
          : "font-medium text-muted hover:bg-surface-low hover:text-foreground",
      )}
    >
      {active ? (
        // Active marker that survives the icon rail: a real state (current
        // page), so it earns its mark in addition to tint and icon weight.
        <span
          aria-hidden
          className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary"
        />
      ) : null}
      <Icon className="size-5 shrink-0" strokeWidth={active ? 2.4 : 1.9} aria-hidden />
      <span className="hidden truncate lg:block">{item.label}</span>
    </Link>
  );

  if (labelsVisible) return link;
  return (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent side="right">{item.label}</TooltipContent>
    </Tooltip>
  );
}

function AssistCta({
  active,
  labelsVisible,
}: {
  active: boolean;
  labelsVisible: boolean;
}) {
  const link = (
    <Link
      href="/assistant"
      aria-label="Tanya Asisten"
      aria-current={active ? "page" : undefined}
      // Glow stays on this one element only: DESIGN.md §4.6 names the
      // sidebar CTA as the primary-action glow.
      className="flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-3 text-sm font-semibold text-on-primary shadow-glow-primary transition-colors hover:bg-primary-hover lg:justify-start"
    >
      <MessagesSquare className="size-5 shrink-0" aria-hidden />
      <span className="hidden lg:inline">Tanya Asisten</span>
    </Link>
  );

  if (labelsVisible) return link;
  return (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent side="right">Tanya Asisten</TooltipContent>
    </Tooltip>
  );
}
