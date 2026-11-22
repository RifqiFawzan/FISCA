// Renders each FISCA page at 360 and 1280 px and saves the hydrated HTML for
// the accessibility/responsive audit. Scripts are stripped: they carry no
// accessibility information and dominate the file size.
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const chrome = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe";
const port = 9445;
const outDir = "audit-html";
const pages = ["home", "history", "reports", "budget", "savings", "profile", "assistant", "login"];
const widths = [
  { name: "360", width: 360, height: 900, mobile: true },
  { name: "1280", width: 1280, height: 900, mobile: false },
];

const profile = path.join(os.tmpdir(), "fisca-capture");
fs.rmSync(profile, { recursive: true, force: true });
fs.mkdirSync(outDir, { recursive: true });

const child = spawn(
  chrome,
  [
    "--headless=new", "--disable-gpu", "--no-sandbox", "--no-first-run",
    `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`,
    "--window-size=1280,900", "about:blank",
  ],
  { stdio: "ignore" },
);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function stripScripts(html) {
  return html
    .replace(/<script[\s\S]*?<\/script>/gi, "")
    .replace(/\s{2,}/g, " ");
}

(async () => {
  let targets = null;
  for (let i = 0; i < 60; i++) {
    try {
      targets = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
      if (targets.length) break;
    } catch {}
    await sleep(250);
  }
  const target = targets.find((t) => t.type === "page");
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });

  let id = 0;
  const pending = new Map();
  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.id && pending.has(msg.id)) {
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
    }
  };
  const send = (method, params = {}) =>
    new Promise((resolve, reject) => {
      const next = ++id;
      pending.set(next, { resolve, reject });
      ws.send(JSON.stringify({ id: next, method, params }));
    });

  await send("Page.enable");
  await send("Runtime.enable");

  for (const size of widths) {
    await send("Emulation.setDeviceMetricsOverride", {
      width: size.width, height: size.height, deviceScaleFactor: 1, mobile: size.mobile,
    });
    for (const page of pages) {
      await send("Page.navigate", { url: `http://localhost:3000/${page}` });
      await sleep(3500);
      const result = await send("Runtime.evaluate", {
        expression: "document.documentElement.outerHTML",
        returnByValue: true,
      });
      const html = stripScripts(String(result.result.value || ""));
      const file = path.join(outDir, `${page}-${size.name}.html`);
      fs.writeFileSync(file, html, "utf8");
      console.log(`${page}-${size.name}: ${html.length} bytes`);
    }
  }

  ws.close();
  child.kill();
  console.log("captured", pages.length * widths.length, "files");
})().catch((err) => {
  console.error(err);
  child.kill();
  process.exit(1);
});
