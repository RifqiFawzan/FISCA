// WCAG AA audit for the Color_Design.md YAML token adoption.
// Compares current vs proposed values on every real FISCA surface/tint.
const hex = (h) => {
  h = h.replace("#", "");
  if (h.length === 3) h = [...h].map((c) => c + c).join("");
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
};
const lum = ([r, g, b]) => {
  const f = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
};
const ratio = (fg, bg) => {
  const [l1, l2] = [lum(hex(fg)), lum(hex(bg))].sort((a, b) => b - a);
  return (l1 + 0.05) / (l2 + 0.05);
};
const mix = (over, under, alpha) => {
  const o = hex(over), u = hex(under);
  return "#" + o.map((c, i) => Math.round(c * alpha + u[i] * (1 - alpha)).toString(16).padStart(2, "0")).join("");
};

const surfaces = {
  canvas: "#0a0e16", background: "#0f131c", surfaceLow: "#181c24",
  surface: "#1c2028", surfaceHigh: "#262a33", surfaceHighest: "#31353e",
};

const checks = [
  // [label, fg current, fg proposed, bg-key or bg fn]
  ["foreground / surface", "#f1f5f9", "#dfe2ee", () => surfaces.surface],
  ["foreground / surfaceHighest", "#f1f5f9", "#dfe2ee", () => surfaces.surfaceHighest],
  ["muted / background", "#94a3b8", "#bbcabf", () => surfaces.background],
  ["muted / surfaceLow", "#94a3b8", "#bbcabf", () => surfaces.surfaceLow],
  ["muted / surface", "#94a3b8", "#bbcabf", () => surfaces.surface],
  ["muted / surfaceHigh", "#94a3b8", "#bbcabf", () => surfaces.surfaceHigh],
  ["muted / surfaceHighest", "#94a3b8", "#bbcabf", () => surfaces.surfaceHighest],
  ["muted / canvas", "#94a3b8", "#bbcabf", () => surfaces.canvas],
  ["muted-dim / surfaceLow", "#8493ab", "#86948a", () => surfaces.surfaceLow],
  ["muted-dim / surface", "#8493ab", "#86948a", () => surfaces.surface],
  ["muted-dim / background", "#8493ab", "#86948a", () => surfaces.background],
  ["muted-dim / canvas", "#8493ab", "#86948a", () => surfaces.canvas],
  ["danger / surfaceLow (badge base)", "#f66981", "#ffb4ab", () => surfaces.surfaceLow],
  ["danger / danger-15 over surfaceLow", "#f66981", "#ffb4ab", () => mix("#f66981", surfaces.surfaceLow, 0.15)],
  ["danger / danger-10 over surface", "#f66981", "#ffb4ab", () => mix("#f66981", surfaces.surface, 0.1)],
  ["danger / surfaceHigh", "#f66981", "#ffb4ab", () => surfaces.surfaceHigh],
];

let fail = 0;
console.log("label".padEnd(42), "old".padStart(6), "new".padStart(6), " verdict");
for (const [label, oldFg, newFg, bgFn] of checks) {
  const bg = bgFn();
  const o = ratio(oldFg, bg);
  const n = ratio(newFg, bg);
  const verdict = n >= 4.5 ? "AA PASS" : n >= 3 ? "LARGE-ONLY" : "FAIL";
  if (n < 4.5) fail++;
  console.log(label.padEnd(42), o.toFixed(2).padStart(6), n.toFixed(2).padStart(6), " " + verdict + (n < o ? " (drops)" : ""));
}
console.log(fail === 0 ? "\nALL PROPOSED VALUES PASS AA (>=4.5) ON EVERY SURFACE" : `\n${fail} CHECK(S) BELOW 4.5 — NEEDS REVIEW`);
process.exit(fail === 0 ? 0 : 1);
