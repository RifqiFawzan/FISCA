// Finds hover/deep companions for the new mint primary that keep on-primary
// text (#003824) at >= 4.5:1, and reports the current values for comparison.
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
const toHex = (c) => "#" + c.map((v) => v.toString(16).padStart(2, "0")).join("");

const ON_PRIMARY = "#003824";
const surfaces = ["#0A0E16", "#0F131C", "#181C24", "#1C2028", "#262A33", "#31353E"];

console.log("=== NILAI SEKARANG (on-primary di atas) ===");
for (const [name, hex] of Object.entries({
  primary: "#10B981",
  "primary-hover": "#059669",
  "primary-deep": "#047857",
})) {
  console.log(`  ${name.padEnd(15)} ${hex} = ${ratio(ON_PRIMARY, hex).toFixed(2)} ${ratio(ON_PRIMARY, hex) >= 4.5 ? "LULUS" : "GAGAL"}`);
}

// Gradient end (primary-deep) also carries the logo "F" at 14px extrabold, which
// is NOT WCAG-large text, so it needs 4.5 too.
console.log("\n=== KANDIDAT UNTUK MINT #4EDEA3 ===");
const mint = rgb("#4EDEA3");
const rows = [];
for (let step = 0; step <= 75; step++) {
  const t = step / 100;
  const c = mint.map((v) => Math.round(v * (1 - t)));
  const hex = toHex(c);
  const onBtn = ratio(ON_PRIMARY, hex);
  let worst = Infinity;
  for (const sh of surfaces) worst = Math.min(worst, ratio(hex, sh));
  rows.push({ step, hex, onBtn, worst });
}
const pass = rows.filter((r) => r.onBtn >= 4.5 && r.worst >= 4.5);
console.log(`  lulus semua: ${pass.length} kandidat`);
console.log(`  paling terang lulus : ${pass[0].hex} (tombol ${pass[0].onBtn.toFixed(2)}, worst ${pass[0].worst.toFixed(2)})`);
const mid = pass.find((r) => r.step >= 30) || pass[Math.floor(pass.length / 2)];
console.log(`  sedang (hover)      : ${mid.hex} (tombol ${mid.onBtn.toFixed(2)}, worst ${mid.worst.toFixed(2)})`);
const deep = [...pass].reverse()[0];
console.log(`  paling gelap lulus  : ${deep.hex} (tombol ${deep.onBtn.toFixed(2)}, worst ${deep.worst.toFixed(2)})`);

// Also verify the gradient endpoint against the lightest end (primary itself),
// since the logo sits on a primary -> primary-deep gradient.
console.log("\n=== LOGO 'F' (14px extrabold, bukan teks besar) di gradien ===");
for (const hex of ["#10B981", "#4EDEA3", mid.hex, deep.hex]) {
  const r = ratio(ON_PRIMARY, hex);
  console.log(`  ${hex} = ${r.toFixed(2)} ${r >= 4.5 ? "LULUS" : "GAGAL"}`);
}
