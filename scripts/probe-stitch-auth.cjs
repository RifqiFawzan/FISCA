// Menentukan gaya autentikasi mana yang diterima endpoint download Stitch.
// URL yang dipakai adalah persis yang diminta MCP (htmlCode.downloadUrl).
// Tidak pernah mencetak API key.
const fs = require("node:fs");

const PROJECT = "2264071121943649738";
const BASE = "https://stitch.googleapis.com/v1";
const SCREEN = "90c5033b0eba4a04bc5b80d63dbe2244";

function readKey() {
  const line = fs.readFileSync(".env", "utf8").split(/\r?\n/).find((l) => l.startsWith("STITCH_API_KEY="));
  return line.slice("STITCH_API_KEY=".length).trim();
}
const key = readKey();
const url = `${BASE}/projects/${PROJECT}/files/${SCREEN}:download`;

async function probe(label, headers, query) {
  const target = query ? `${url}?${query}` : url;
  let res;
  try {
    res = await fetch(target, { headers: headers || {} });
  } catch (e) {
    console.log(`  ${label.padEnd(34)} FETCH ERROR ${e.message}`);
    return;
  }
  const t = await res.text();
  const kind = t.trimStart().startsWith("{") ? "JSON" : /<\?xml|<!DOCTYPE|<html/i.test(t) ? "HTML/XML" : "biner";
  console.log(`  ${label.padEnd(34)} ${res.status}  ${t.length} byte  ${kind}  ${res.ok ? "" : t.slice(0, 110).replace(/\s+/g, " ")}`);
}

(async () => {
  console.log(`target: ${url}\n`);
  console.log("=== gaya autentikasi pada endpoint download ===");
  await probe("tanpa autentikasi", {});
  await probe("X-Goog-Api-Key (dipakai REST)", { "X-Goog-Api-Key": key });
  await probe("apikey header", { apikey: key });
  await probe("Authorization: Bearer", { Authorization: `Bearer ${key}` });
  await probe("query ?key=", {}, `key=${key}`);

  console.log("\n=== perbandingan: endpoint list (yang BERHASIL di MCP) ===");
  await probeList("tanpa autentikasi", {});
  await probeList("X-Goog-Api-Key", { "X-Goog-Api-Key": key });
  await probeList("apikey header", { apikey: key });
  await probeList("Authorization: Bearer", { Authorization: `Bearer ${key}` });
  await probeList("query ?key=", {}, `key=${key}`);
})();

async function probeList(label, headers, query) {
  const target = `${BASE}/projects/${PROJECT}/screens${query ? "?" + query : ""}`;
  let res;
  try {
    res = await fetch(target, { headers: headers || {} });
  } catch (e) {
    console.log(`  ${label.padEnd(34)} FETCH ERROR ${e.message}`);
    return;
  }
  const t = await res.text();
  let n = "?";
  try {
    n = (JSON.parse(t).screens || []).length;
  } catch {}
  console.log(`  ${label.padEnd(34)} ${res.status}  screens=${n}  ${res.ok ? "" : t.slice(0, 100).replace(/\s+/g, " ")}`);
}
