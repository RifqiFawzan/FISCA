// WCAG contrast check for the Stitch Dark Carbon Glass palette against the
// surfaces FISCA actually uses. Run before adopting any new token.
function lum(hex) {
  const n = parseInt(hex.replace("#", ""), 16);
  const c = [(n >> 16) & 255, (n >> 8) & 255, n & 255].map((v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
}
const rgb = (h) => {
  const n = parseInt(h.replace("#", ""), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
};
const ratio = (a, b) => {
  const [hi, lo] = lum(a) > lum(b) ? [lum(a), lum(b)] : [lum(b), lum(a)];
  return (hi + 0.05) / (lo + 0.05);
};
const over = (fg, alpha, bg) => fg.map((v, i) => Math.round(alpha * v + (1 - alpha) * bg[i]));
const toHex = (c) => "#" + c.map((v) => v.toString(16).padStart(2, "0")).join("");

// Surfaces FISCA uses (identical to Stitch's surface-container scale).
const surfaces = {
  canvas: "#0A0E16",
  background: "#0F131C",
  surfaceLow: "#181C24",
  surface: "#1C2028",
  surfaceHigh: "#262A33",
  surfaceHighest: "#31353E",
};

// Candidate accent text colors from the Stitch palette.
const fg = {
  primary_NEW: "#4EDEA3",
  primary_OLD: "#10B981",
  secondary_NEW: "#4CD7F6",
  tertiary_NEW: "#C0C1FF",
  foreground: "#DFE2EE",
  muted: "#94A3B8",
  error_NEW: "#FFB4AB",
};

console.log("=== TEKS AKSEN DI PERMUKAAN (butuh >= 4.5) ===");
for (const [name, hex] of Object.entries(fg)) {
  let worst = Infinity, at = "";
  for (const [sn, sh] of Object.entries(surfaces)) {
    const r = ratio(hex, sh);
    if (r < worst) { worst = r; at = sn; }
  }
  const glass = toHex(over(rgb(hex), 0.15, rgb("#141820"))); // badge tint di atas panel kaca
  const rBadge = ratio(hex, glass);
  const flag = worst >= 4.5 && rBadge >= 4.5 ? "LULUS" : "GAGAL";
  console.log(
    `  ${name.padEnd(14)} ${hex}  worst=${worst.toFixed(2)} (${at})  badge15=${rBadge.toFixed(2)}  ${flag}`,
  );
}

// Button text: on-primary on the accent background.
console.log("\n=== TEKS TOMBOL (on-primary #003824 di atas aksen) ===");
for (const [name, hex] of Object.entries(fg)) {
  const r = ratio("#003824", hex);
  console.log(`  on-primary on ${name.padEnd(14)} ${hex} = ${r.toFixed(2)} ${r >= 4.5 ? "LULUS" : "GAGAL"}`);
}

// If mint is too light for button text, find a darker mint that still reads
// as the same accent and passes on both the button and as text on surfaces.
console.log("\n=== KANDIDAT MINT GELAP (teks tombol >= 4.5, tetap terbaca) ===");
const mint = rgb("#4EDEA3");
for (let step = 0; step <= 60; step += 3) {
  const t = step / 100;
  const c = mint.map((v) => Math.round(v * (1 - t)));
  const hex = toHex(c);
  const onBtn = ratio("#003824", hex);
  let worst = Infinity;
  for (const sh of Object.values(surfaces)) worst = Math.min(worst, ratio(hex, sh));
  const glass = toHex(over(c, 0.15, rgb("#141820")));
  const badge = ratio(hex, glass);
  if (onBtn >= 4.5 && worst >= 4.5 && badge >= 4.5) {
    console.log(`  ${hex}  tombol=${onBtn.toFixed(2)}  worst=${worst.toFixed(2)}  badge=${badge.toFixed(2)}`);
    if (step > 30) break;
  }
}
