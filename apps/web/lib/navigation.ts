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

/** Desktop sidebar, grouped for scanning (DESIGN.md §7.2 destinations). */
export type NavGroup = {
  label: string;
  items: NavItem[];
};

/**
 * Three clusters instead of one flat list: review surfaces, money plans, and
 * the account. Labels follow DESIGN.md §14: short, natural Indonesian, no
 * uppercase micro-labels.
 */
export const desktopNavGroups: NavGroup[] = [
  {
    label: "Tinjauan",
    items: [
      { href: "/home", label: "Beranda", icon: House },
      { href: "/history", label: "Riwayat", icon: ReceiptText },
      { href: "/reports", label: "Laporan", icon: ChartNoAxesColumn },
    ],
  },
  {
    label: "Kelola",
    items: [
      { href: "/budget", label: "Budget", icon: Wallet },
      { href: "/savings", label: "Tabungan", icon: PiggyBank },
    ],
  },
  {
    label: "Akun",
    items: [{ href: "/profile", label: "Profil & Pengaturan", icon: UserRound }],
  },
];

export function isActivePath(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(`${href}/`);
}
