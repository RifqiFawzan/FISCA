// Mobile layout audit (antislop R-03/R-35 + DESIGN.md §11/§17).
// Per page at phone widths: horizontal overflow + offenders, under-sized tap
// targets, clipped text, fixed bottom-nav overlap at scroll end, console
// errors. Screenshots land in tests/.artifacts/.
// CRITICAL findings fail the run; WARN findings are printed for review.
import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";

const WEB = process.env.WEB_URL || "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const WIDTHS = [
  { w: 390, h: 844, tag: "390" },
  { w: 360, h: 740, tag: "360" },
  { w: 414, h: 896, tag: "414" }, // iPhone 11 CSS viewport
];
mkdirSync("tests/.artifacts", { recursive: true });

const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
if (!login.ok) throw new Error("login failed");
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const critical = [];
const warns = [];

async function auditPage(page, path, tag) {
  const errors = [];
  const onConsole = (m) => { if (m.type() === "error") errors.push(m.text().slice(0, 160)); };
  page.on("console", onConsole);
  page.on("pageerror", (e) => errors.push(e.message.slice(0, 160)));

  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(900);

  const findings = await page.evaluate(() => {
    const vw = window.innerWidth;
    const out = { overflow: 0, offenders: [], small: [], clipped: [], navOverlap: null, viewportMeta: null };

    const doc = document.documentElement;
    out.overflow = Math.max(0, doc.scrollWidth - vw);

    const inScroller = (el) => {
      let p = el.parentElement;
      while (p && p !== document.body) {
        const ox = getComputedStyle(p).overflowX;
        if (ox === "auto" || ox === "scroll") return true;
        p = p.parentElement;
      }
      return false;
    };

    if (out.overflow > 1) {
      for (const el of document.querySelectorAll("body *")) {
        const r = el.getBoundingClientRect();
        if (r.width < 2 || r.height < 2) continue;
        if (inScroller(el)) continue;
        if (r.right > vw + 1 || r.left < -1) {
          out.offenders.push({
            tag: el.tagName,
            cls: String(el.className || "").slice(0, 80),
            left: Math.round(r.left),
            right: Math.round(r.right),
          });
          if (out.offenders.length >= 6) break;
        }
      }
    }

    for (const el of document.querySelectorAll("a[href], button, input, select, textarea, [role=button]")) {
      const r = el.getBoundingClientRect();
      if (r.width < 4 || r.height < 4) continue;
      const cs = getComputedStyle(el);
      if (cs.visibility === "hidden" || cs.display === "none") continue;
      // Skip hidden states (opacity-0/pointer-events-none): e.g. the inactive
      // jump-to-end button is scaled down while invisible; measure visible only.
      if (cs.opacity === "0" || cs.pointerEvents === "none") continue;
      if (r.width < 24 || r.height < 24) {
        out.small.push({ crit: true, tag: el.tagName, txt: (el.getAttribute("aria-label") || el.textContent || "").trim().slice(0, 40), w: Math.round(r.width), h: Math.round(r.height) });
      } else if (r.width < 44 || r.height < 44) {
        out.small.push({ crit: false, tag: el.tagName, txt: (el.getAttribute("aria-label") || el.textContent || "").trim().slice(0, 40), w: Math.round(r.width), h: Math.round(r.height) });
      }
    }

    for (const el of document.querySelectorAll("p, span, h1, h2, h3, a, button")) {
      if (el.children.length > 0) continue;
      if (!el.textContent || !el.textContent.trim()) continue;
      if (el.className && String(el.className).includes("truncate")) continue;
      if (el.className && String(el.className).includes("sr-only")) continue;
      const cs = getComputedStyle(el);
      if ((cs.overflowX === "hidden" || cs.textOverflow === "ellipsis") && el.scrollWidth > el.clientWidth + 2) {
        out.clipped.push({ tag: el.tagName, txt: el.textContent.trim().slice(0, 50), scrollW: el.scrollWidth, clientW: el.clientWidth });
      }
    }

    const nav = document.querySelector('[class*="fixed bottom-0"], nav[aria-label*="utama"], [data-mobile-nav]');
    const main = document.querySelector("main");
    if (nav && main) {
      window.scrollTo(0, document.body.scrollHeight);
      const nr = nav.getBoundingClientRect();
      const last = main.querySelector(":scope > *:last-child");
      const lr = last ? last.getBoundingClientRect() : null;
      out.navOverlap = lr ? { lastBottom: Math.round(lr.bottom), navTop: Math.round(nr.top), overlap: lr.bottom > nr.top + 2 } : null;
    }

    const meta = document.querySelector('meta[name="viewport"]');
    out.viewportMeta = meta ? meta.content : null;
    return out;
  });

  const label = `${path}@${tag}`;
  for (const e of [...new Set(errors)]) critical.push(`${label} console: ${e}`);
  if (findings.overflow > 1) critical.push(`${label} HORIZONTAL OVERFLOW ${findings.overflow}px: ${JSON.stringify(findings.offenders.slice(0, 3))}`);
  for (const s of findings.small) (s.crit ? critical : warns).push(`${label} target ${s.w}x${s.h} ${s.tag} "${s.txt}"`);
  for (const c of findings.clipped) warns.push(`${label} clipped <${c.tag}> ${c.txt} (${c.scrollW}>${c.clientW})`);
  if (findings.navOverlap?.overlap) critical.push(`${label} bottom nav covers content end: ${JSON.stringify(findings.navOverlap)}`);
  if (findings.viewportMeta && /user-scalable=no|maximum-scale=1/.test(findings.viewportMeta)) critical.push(`${label} zoom blocked: ${findings.viewportMeta}`);

  await page.screenshot({ path: `tests/.artifacts/mobile-${path.replace(/\//g, "_") || "_root"}-${tag}.png`, fullPage: true });
  page.off("console", onConsole);
}

for (const { w, h, tag } of WIDTHS) {
  // logged-out login page
  let page = await browser.newPage({ viewport: { width: w, height: h } });
  await auditPage(page, "/login", tag);
  await page.close();

  // logged-in app pages
  page = await browser.newPage({ viewport: { width: w, height: h } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  for (const path of ["/home", "/history", "/reports", "/budget", "/savings", "/profile", "/assistant"]) {
    await auditPage(page, path, tag);
  }
  await page.close();
}
await browser.close();

console.log("CRITICAL:", critical.length);
for (const c of critical) console.log("  X " + c);
console.log("WARN (<44px target / clipped):", warns.length);
for (const w2 of warns.slice(0, 40)) console.log("  ! " + w2);
const pass = critical.length === 0;
console.log(pass ? "MOBILE AUDIT PASS" : "MOBILE AUDIT FAIL");
process.exit(pass ? 0 : 1);
