// Stitch's htmlCode download is also a JSON wrapper ({... "dataString": "<html>"}).
// This decodes it and prints the design-relevant parts: tailwind config colors,
// gradients, backdrop-filter, and any ambient/mesh glow definitions.
const fs = require("node:fs");
const path = require("node:path");

const dir = process.argv[2];
if (!dir) {
  console.error("usage: node read-stitch-html.cjs <design-refs/NN-...>");
  process.exit(1);
}

const raw = fs.readFileSync(path.join(dir, "index.html"), "utf8");
let html = raw;
if (raw.trimStart().startsWith("{")) {
  const wrapper = JSON.parse(raw);
  html = wrapper.dataString ?? wrapper.html ?? raw;
}

const patterns = [
  ["tailwind colors", /"colors":\s*\{[^}]*\}/g],
  ["gradients", /(?:background(?:-image)?:\s*)?(?:radial|linear)-gradient\([^;"}]{0,240}/g],
  ["backdrop", /backdrop-filter[^;"}]{0,120}/g],
  ["mesh/glow words", /.{90}(?:mesh|glow|ambient).{90}/gi],
];

for (const [label, re] of patterns) {
  const found = [...new Set((html.match(re) || []).map((s) => s.replace(/\s+/g, " ").trim()))];
  console.log(`\n=== ${label} (${found.length}) ===`);
  for (const f of found.slice(0, 8)) console.log("  " + f.slice(0, 320));
}
