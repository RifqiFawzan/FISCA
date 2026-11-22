// Removes leftover design-refs folders from earlier pulls. manifest.json written
// by pull-stitch-screens.cjs is the source of truth for which folders are current.
const fs = require("node:fs");
const path = require("node:path");

const root = "design-refs";
const manifest = JSON.parse(fs.readFileSync(path.join(root, "manifest.json"), "utf8"));
const keep = new Set((manifest.screens || []).map((s) => s.dir.split("/").pop()));

let removed = 0;
for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue;
  if (keep.has(entry.name)) continue;
  fs.rmSync(path.join(root, entry.name), { recursive: true, force: true });
  console.log(`removed: ${entry.name}`);
  removed++;
}

console.log(`kept=${keep.size} removed=${removed}`);
