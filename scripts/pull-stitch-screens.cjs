// Downloads every screen (HTML + PNG) from the FISCA Stitch project into
// design-refs/. Uses the key from .env and never prints it.
//
// By default only the "Dark Carbon Glass" screens are downloaded: the API
// returns every screen in the project (including PRD documents and the other
// design directions), which mixed unrelated variants into one folder. The
// design system id sits in nested JSON strings (theme -> screenMetadata ->
// designSystem -> name), so matching the serialized screen is the simplest
// exact filter. Pass --all to download everything instead.
const fs = require("node:fs");
const path = require("node:path");

const PROJECT = "2264071121943649738";
const BASE = "https://stitch.googleapis.com/v1";
const OUT = "design-refs";

// Dark Carbon Glass (figaro_agent, DARK). Subtle Dark Gradient = assets/f7e7ab47…,
// Clean/terang = assets/2eb8302a….
const DARK_CARBON_GLASS = "assets/06cc08b08a0845e7bdd43dc7802142c9";
const pullAll = process.argv.includes("--all");

function readKey() {
  const text = fs.readFileSync(".env", "utf8");
  const line = text.split(/\r?\n/).find((l) => l.startsWith("STITCH_API_KEY="));
  if (!line) throw new Error("STITCH_API_KEY not found in .env");
  return line.slice("STITCH_API_KEY=".length).trim();
}
const key = readKey();
const headers = { "X-Goog-Api-Key": key, Accept: "application/json" };

// Slug tanpa nomor urut: nomor berbasis posisi dalam daftar API membuat nama
// folder berganti tiap kali urutan screen berubah, sehingga folder lama
// tertinggal menumpuk. Kalau dua judul menghasilkan slug sama, id screen
// dipakai sebagai pembeda.
function slugify(title) {
  return (
    title
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 60) || "screen"
  );
}

function planDirs(screens) {
  const taken = new Set();
  return screens.map((screen) => {
    const base = slugify(screen.title || screen.id);
    let slug = base;
    if (taken.has(slug)) slug = `${base}-${screen.id.slice(0, 8)}`;
    taken.add(slug);
    return { screen, dir: path.join(OUT, slug) };
  });
}

async function download(url, destination) {
  const res = await fetch(url, { headers: { "X-Goog-Api-Key": key } });
  if (!res.ok) throw new Error(`${res.status} for ${url}`);
  const buffer = Buffer.from(await res.arrayBuffer());
  fs.writeFileSync(destination, buffer);
  return buffer.length;
}

(async () => {
  const listRes = await fetch(`${BASE}/projects/${PROJECT}/screens`, { headers });
  if (!listRes.ok) throw new Error(`list screens: ${listRes.status}`);
  const list = await listRes.json();
  const all = list.screens || [];
  // PRD documents carry the same design system asset as the screens, so also
  // require an actual design (screenType DESIGN) with a real viewport.
  const screens = pullAll
    ? all
    : all.filter(
        (s) => s.screenType === "DESIGN" && JSON.stringify(s).includes(DARK_CARBON_GLASS),
      );
  console.log(`screens: ${screens.length} of ${all.length}${pullAll ? " (--all)" : " (Dark Carbon Glass)"}`);
  if (screens.length === 0) {
    console.error("tidak ada screen yang cocok — cek konstanta design system atau pakai --all");
    process.exit(1);
  }

  fs.mkdirSync(OUT, { recursive: true });
  const plan = planDirs(screens);

  // Buang folder basi dari tarikan sebelumnya (slug lama berbasis nomor urut),
  // supaya design-refs/ tidak menumpuk.
  const keep = new Set(plan.map((p) => path.basename(p.dir)));
  for (const name of fs.readdirSync(OUT)) {
    if (name === "manifest.json") continue;
    const full = path.join(OUT, name);
    if (fs.statSync(full).isDirectory() && !keep.has(name)) {
      fs.rmSync(full, { recursive: true, force: true });
      console.log(`hapus folder basi: ${name}`);
    }
  }

  const manifest = [];

  for (let i = 0; i < plan.length; i++) {
    const { screen, dir } = plan[i];
    fs.mkdirSync(dir, { recursive: true });

    const entry = {
      id: screen.id,
      title: screen.title || "",
      width: screen.width,
      height: screen.height,
      generatedBy: screen.generatedBy || "",
      prompt: screen.prompt || "",
      dir: path.relative(".", dir).replace(/\\/g, "/"),
    };

    if (screen.htmlCode && screen.htmlCode.downloadUrl) {
      try {
        entry.htmlBytes = await download(screen.htmlCode.downloadUrl, path.join(dir, "index.html"));
      } catch (err) {
        entry.htmlError = err.message;
      }
    }
    if (screen.screenshot && screen.screenshot.downloadUrl) {
      try {
        entry.pngBytes = await download(screen.screenshot.downloadUrl, path.join(dir, "screenshot.png"));
      } catch (err) {
        entry.pngError = err.message;
      }
    }

    manifest.push(entry);
    console.log(
      `[${i + 1}/${screens.length}] ${entry.title.slice(0, 60)} | html=${entry.htmlBytes ?? entry.htmlError} png=${entry.pngBytes ?? entry.pngError}`,
    );
  }

  fs.writeFileSync(
    path.join(OUT, "manifest.json"),
    JSON.stringify({ project: PROJECT, fetchedAt: new Date().toISOString(), screens: manifest }, null, 2),
    "utf8",
  );
  const okHtml = manifest.filter((m) => m.htmlBytes).length;
  const okPng = manifest.filter((m) => m.pngBytes).length;
  console.log(`done: ${okHtml} html, ${okPng} png of ${manifest.length}`);

  // API mengirim screenshot sebagai pembungkus JSON; unwrap otomatis supaya
  // file langsung berupa JPEG/PNG yang valid.
  const { spawnSync } = require("node:child_process");
  const unwrap = spawnSync("node", [path.join("scripts", "unwrap-stitch-screenshots.cjs")], {
    stdio: "inherit",
  });
  if (unwrap.status !== 0) {
    console.error("unwrap gagal — jalankan manual: node scripts/unwrap-stitch-screenshots.cjs");
    process.exit(1);
  }
})();
