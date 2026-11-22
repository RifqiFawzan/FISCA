// Stitch's download endpoint returns a JSON wrapper ({imageMetadata, dataString})
// instead of raw image bytes, even though the file is named .png. This unwraps
// every screenshot into a real image file the viewer can display.
const fs = require("node:fs");
const path = require("node:path");

const root = "design-refs";
let fixed = 0;
let skipped = 0;

for (const dir of fs.readdirSync(root, { withFileTypes: true })) {
  if (!dir.isDirectory()) continue;
  const file = path.join(root, dir.name, "screenshot.png");
  if (!fs.existsSync(file)) continue;

  const head = fs.readFileSync(file).subarray(0, 1).toString("utf8");
  if (head !== "{") {
    // Already a real image.
    skipped++;
    continue;
  }

  const wrapper = JSON.parse(fs.readFileSync(file, "utf8"));
  const mime = wrapper.imageMetadata?.mimeType || "image/jpeg";
  const ext = mime === "image/png" ? "png" : "jpg";
  const bytes = Buffer.from(wrapper.dataString, "base64");

  const out = path.join(root, dir.name, `screenshot.${ext}`);
  // Hapus wrapper lama sebelum menulis. Bila ekstensi hasil sama (png -> png),
  // menulis dulu lalu menghapus akan menghapus gambar yang baru ditulis.
  if (path.resolve(out) !== path.resolve(file)) fs.rmSync(file, { force: true });
  fs.writeFileSync(out, bytes);
  fixed++;
  console.log(`${dir.name}: ${wrapper.imageMetadata.width}x${wrapper.imageMetadata.height} ${mime} -> screenshot.${ext} (${bytes.length} bytes)`);
}

console.log(`unwrapped=${fixed} already_ok=${skipped}`);
