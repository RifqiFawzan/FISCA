import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Standalone hanya untuk image Docker; build lokal tetap memakai output biasa
  // agar tidak bergantung pada symlink di Windows.
  output: process.env.NEXT_STANDALONE === "1" ? "standalone" : undefined,
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    // Repo punya dua lockfile (root + apps/web); pin workspace root agar
    // Turbopack tidak menebak dari direktori yang salah.
    root: process.cwd(),
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
