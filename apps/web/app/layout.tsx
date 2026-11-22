import type { Metadata, Viewport } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import "./globals.css";

const plusJakarta = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-plus-jakarta",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "FISCA · Keuangan pribadi yang lebih jernih",
    template: "%s · FISCA",
  },
  description:
    "Catat transaksi lewat suara atau chat, pahami arus kas, dan kelola budget bulanan dalam satu tempat.",
  applicationName: "FISCA",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: "#0a0e16",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="id" className={`${plusJakarta.variable} h-full`}>
      <body className="min-h-full bg-canvas text-foreground">
        {children}
      </body>
    </html>
  );
}
