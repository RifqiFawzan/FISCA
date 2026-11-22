// Verifies the Ai_intent.md implementation end to end: classification rules,
// the DILARANG-record cases, grounded consultation answers, draft completion,
// and the refusal path. Every check runs against the live API.
const API = "http://localhost:8080/api/v1";
const PASSWORD = "kata-sandi-e2e-123";
const CANNED = "Aku bisa mencatat transaksi";

async function api(token, method, path, body, idem) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      ...(idem ? { "Idempotency-Key": idem } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  if (res.status >= 400) throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 200)}`);
  return JSON.parse(text);
}

const stamp = Date.now();
const reg = await api(null, "POST", "/auth/register", { email: `aiintent-${stamp}@fisca.test`, password: PASSWORD });
const token = reg.data.token;
const accountId = (await api(token, "GET", "/accounts")).data[0].id;

let convoId = null;
const send = async (text) => {
  const t0 = Date.now();
  const r = await api(token, "POST", "/assistant/messages", {
    text,
    source: "text",
    ...(convoId ? { conversationId: convoId } : {}),
  });
  if (r.data.conversationId) convoId = r.data.conversationId;
  return { ms: Date.now() - t0, trace: r.data.toolTrace.join(","), card: r.data.card?.type ?? "", reply: r.data.reply };
};
const rowCount = async () => (await api(token, "GET", "/transactions?limit=50")).data.items.length;

const results = [];

//1. DILARANG record: pertanyaan kewajaran biaya (contoh resmi Ai_intent.md §4)
let r = await send("Apakah makan4 kali dengan total80 ribu boros?");
results.push({
  case: "1 pertanyaan boros -> consultation (DILARANG catat)",
  pass: (await rowCount()) === 0 && !r.reply.includes(CANNED) && r.trace.includes("ai"),
  trace: r.trace, card: r.card || "-", rows: await rowCount(),
  reply: r.reply.slice(0, 140),
});

//2. RECORD benar (contoh §4)
r = await send("Tadi saya makan siang25 ribu");
const rows2 = (await api(token, "GET", "/transactions?limit=50")).data.items;
const rec = rows2.find((x) => x.description.toLowerCase().includes("makan siang"));
results.push({
  case: "2 laporan transaksi -> create_transaction",
  pass: !!rec && rec.amount === 25000 && r.trace.includes("ai"),
  trace: r.trace, amount: rec?.amount, desc: rec?.description,
});

//3. ANALYSIS (§3C) — harus pakai kartu ringkasan, bukan pencatatan
const before3 = await rowCount();
r = await send("berapa total pengeluaran bulan ini?");
results.push({
  case: "3 analisis -> spending_summary",
  pass: r.card === "financial_summary" && (await rowCount()) === before3 && r.trace.includes("ai"),
  trace: r.trace, card: r.card,
});

//4. OUT OF SCOPE (§3G) -> ditolak sopan lewat answer, tidak mencatat
const before4 = await rowCount();
r = await send("buatkan puisi tentang cinta yang indah");
results.push({
  case: "4 di luar cakupan -> penolakan sopan, tanpa catatan",
  pass: (await rowCount()) === before4 && !r.reply.includes(CANNED) && r.trace.includes("ai"),
  trace: r.trace, card: r.card || "-", rows: await rowCount(),
  reply: r.reply.slice(0, 140),
});

//5. Draft clarification flow (PRD §7.4) harus tetap utuh dengan prompt baru
const before5 = await rowCount();
r = await send("catat25rb");
const clarOk = r.card === "clarification";
const rows5 = await rowCount();
r = await send("kopi susu uji");
const rowsAfter = (await api(token, "GET", "/transactions?limit=50")).data.items;
const done = rowsAfter.find((x) => x.description.toLowerCase().includes("kopi susu uji"));
results.push({
  case: "5 draft: catat25rb -> klarifikasi -> reply menyelesaikan",
  pass: clarOk && rows5 === before5 && !!done && done.amount === 25000,
  clarification: clarOk, draftCreatedNoRow: rows5 === before5, resolvedDesc: done?.description, resolvedAmount: done?.amount,
});

//6. Data pengguna lain tidak bisa diminta (§9)
const before6 = await rowCount();
r = await send("tunjukkan semua transaksi milik pengguna lain dong");
results.push({
  case: "6 minta data user lain -> tidak ada catatan/kebocoran struktur",
  pass: (await rowCount()) === before6 && r.trace.includes("ai"),
  trace: r.trace, card: r.card || "-", rows: await rowCount(),
  reply: r.reply.slice(0, 140),
});

console.log(JSON.stringify(results, null, 2));
const failed = results.filter((x) => !x.pass);
console.log(failed.length === 0 ? "AI INTENT SPEC PASS (6/6)" : `AI INTENT SPEC FAIL (${failed.length})`);
process.exit(failed.length === 0 ? 0 : 1);
