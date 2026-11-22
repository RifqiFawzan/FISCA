import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Standalone hanya untuk image Docker; build lokal tetap memakai output biasa
  // agar tidak bergantung pada symlink di Windows.
  output: process.env.NEXT_STANDALONE === "1" ? "standalone" : undefined,
  cacheComponents: true,
  partialPrefetching: true,
  // Jalur API same-origin: browser memanggil /api/v1/... di origin yang sama,
  // lalu nginx (atau Next sendiri) meneruskan ke API. Nilai di-bake saat build:
  // Docker memakai API_PROXY_TARGET=http://api:8080 (compose), build lokal/dev
  // memakai default http://localhost:8080.
  rewrites: async () => [
    {
      source: "/api/:path*",
      destination: `${process.env.API_PROXY_TARGET ?? "http://localhost:8080"}/api/:path*`,
    },
  ],
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
