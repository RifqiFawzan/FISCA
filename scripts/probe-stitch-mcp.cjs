// Menyelidiki kenapa tool MCP sp_screens gagal sementara jalur REST sukses.
// Tidak pernah mencetak API key.
const fs = require("node:fs");

const PROJECT = "2264071121943649738";
const BASE = "https://stitch.googleapis.com/v1";
const TARGET_SCREEN = "90c5033b0eba4a04bc5b80d63dbe2244"; // screen yang disebut galat MCP

function readKey() {
  const line = fs.readFileSync(".env", "utf8").split(/\r?\n/).find((l) => l.startsWith("STITCH_API_KEY="));
  if (!line) throw new Error("STITCH_API_KEY tidak ada di .env");
  return line.slice("STITCH_API_KEY=".length).trim();
}
const key = readKey();

async function head(label, url) {
  let res;
  try {
    res = await fetch(url, { headers: { "X-Goog-Api-Key": key } });
  } catch (err) {
    console.log(`  ${label}: FETCH ERROR ${err.message}`);
    return null;
  }
  const text = await res.text();
  console.log(`  ${label}: ${res.status} (${text.length} byte) ${res.ok ? "OK" : text.slice(0, 160).replace(/\s+/g, " ")}`);
  return res;
}

(async () => {
  const listRes = await fetch(`${BASE}/projects/${PROJECT}/screens`, {
    headers: { "X-Goog-Api-Key": key },
  });
  const list = await listRes.json();
  const screens = list.screens || [];
  console.log(`screens dari REST: ${screens.length}`);

  const target = screens.find((s) => s.id === TARGET_SCREEN);
  if (!target) {
    console.log("screen target tidak ditemukan");
    process.exit(1);
  }
  console.log(`\nscreen target: ${target.title}`);
  console.log(`  id                 : ${target.id}`);
  console.log(`  screenshot.download : ${target.screenshot?.downloadUrl ?? "(tidak ada)"}`);
  console.log(`  htmlCode.download   : ${target.htmlCode?.downloadUrl ?? "(tidak ada)"}`);

  const wrong = `${BASE}/projects/${PROJECT}/files/${target.id}:download`;
  const right = target.screenshot?.downloadUrl;

  console.log("\n=== uji dua URL ===");
  await head("URL yang DITOLAK MCP (pakai screen id)", wrong);
  if (right) await head("URL yang dikirim REST (downloadUrl)", right);

  console.log("\n=== apakah beda? ===");
  console.log(`  URL MCP : ${wrong}`);
  console.log(`  URL REST: ${right}`);
  console.log(`  sama?   : ${wrong === right ? "SAMA" : "BEDA — konstruksi URL MCP salah"}`);
})();
