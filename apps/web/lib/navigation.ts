import type { LucideIcon } from "lucide-react";
import {
  ChartNoAxesColumn,
  House,
  MessagesSquare,
  PiggyBank,
  ReceiptText,
  UserRound,
  Wallet,
} from "lucide-react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  /** Elevated center action (Smart Assistant). */
  primary?: boolean;
};

/** Mobile bottom navigation — exactly five items (DESIGN.md §7.1). */
export const mobileNavItems: NavItem[] = [
  { href: "/home", label: "Beranda", icon: House },
  { href: "/history", label: "Riwayat", icon: ReceiptText },
  { href: "/assistant", label: "Asisten", icon: MessagesSquare, primary: true },
  { href: "/reports", label: "Laporan", icon: ChartNoAxesColumn },
  { href: "/profile", label: "Profil", icon: UserRound },
];

/** Desktop sidebar items (DESIGN.md §7.2). Assistant is a separate CTA. */
export const desktopNavItems: NavItem[] = [
  { href: "/home", label: "Beranda", icon: House },
  { href: "/history", label: "Riwayat", icon: ReceiptText },
  { href: "/reports", label: "Laporan", icon: ChartNoAxesColumn },
  { href: "/budget", label: "Budget", icon: Wallet },
  { href: "/savings", label: "Tabungan", icon: PiggyBank },
  { href: "/profile", label: "Profil & Pengaturan", icon: UserRound },
];

export function isActivePath(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(`${href}/`);
}
