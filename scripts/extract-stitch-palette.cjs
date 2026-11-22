// Extracts the exact color palette from the Dark Carbon Glass screens so the
// FISCA token change is based on measured values, not eyeballing.
const fs = require("node:fs");
const path = require("node:path");

const root = "design-refs";
const manifest = JSON.parse(fs.readFileSync(path.join(root, "manifest.json"), "utf8"));

function decode(file) {
  const raw = fs.readFileSync(file, "utf8");
  if (!raw.trimStart().startsWith("{")) return raw;
  try {
    const w = JSON.parse(raw);
    return w.dataString ?? w.html ?? raw;
  } catch {
    return raw;
  }
}

// tailwind.config = { theme: { extend: { colors: { "name": "#hex", ... } } } }
function tailwindColors(html) {
  const out = {};
  const cfg = html.match(/tailwind\.config\s*=\s*(\{[\s\S]*?\});?\s*<\/script>/);
  const src = cfg ? cfg[1] : html;
  const colorsBlock = src.match(/"colors"\s*:\s*\{([\s\S]*?)\}\s*,\s*"(?:fontFamily|borderRadius|boxShadow)/);
  const block = colorsBlock ? colorsBlock[1] : src;
  for (const m of block.matchAll(/"([^"]+)"\s*:\s*"(#[0-9a-fA-F]{3,8})"/g)) {
    out[m[1]] = m[2].toUpperCase();
  }
  return out;
}

function hexCounts(html) {
  const c = new Map();
  for (const m of html.matchAll(/#[0-9a-fA-F]{6}\b/g)) {
    const h = m[0].toUpperCase();
    c.set(h, (c.get(h) || 0) + 1);
  }
  return [...c.entries()].sort((a, b) => b[1] - a[1]);
}

const merged = {};
const usage = new Map();

for (const s of manifest.screens) {
  const file = path.join(s.dir, "index.html");
  if (!fs.existsSync(file)) continue;
  const html = decode(file);
  const colors = tailwindColors(html);
  for (const [k, v] of Object.entries(colors)) merged[k] = v;
  for (const [hex, n] of hexCounts(html)) usage.set(hex, (usage.get(hex) || 0) + n);
}

console.log("=== TAILWIND COLORS (gabungan 10 screen) ===");
const names = Object.keys(merged).sort();
for (const n of names) console.log(`  ${n.padEnd(34)} ${merged[n]}`);

console.log("\n=== HEX PALING SERING MUNCUL DI HTML ===");
const ranked = [...usage.entries()].sort((a, b) => b[1] - a[1]);
for (const [hex, n] of ranked.slice(0, 25)) console.log(`  ${hex}  x${n}`);

// Focused view on the accent pair that matters for the FISCA token change.
console.log("\n=== AKSEN KUNCI ===");
for (const key of Object.keys(merged).sort()) {
  if (/primary|secondary|tertiary|surface-tint|on-primary$|error|surface$|background/i.test(key)) {
    console.log(`  ${key.padEnd(34)} ${merged[key]}`);
  }
}
