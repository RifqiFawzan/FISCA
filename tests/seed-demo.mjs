// Seeds a demo account with real activity through the API, then prints the
// session token so the browser can be opened already signed in.
const API = "http://localhost:8080/api/v1";
const PASSWORD = "kata-sandi-e2e-123";

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
  const json = text ? JSON.parse(text) : null;
  if (res.status >= 400) throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 200)}`);
  return json;
}

function day(offset) {
  const d = new Date();
  d.setDate(d.getDate() - offset);
  return d.toISOString();
}
function prevMonthDay(offset) {
  const d = new Date();
  d.setMonth(d.getMonth() - 1);
  d.setDate(Math.min(d.getDate(), 28) - offset);
  return d.toISOString();
}

async function main() {
  const stamp = Date.now();
  const email = `demo-${stamp}@fisca.test`;
  const reg = await api(null, "POST", "/auth/register", { email, password: PASSWORD });
  const token = reg.data.token;

  const cats = (await api(token, "GET", "/categories")).data;
  const cat = (name) => cats.find((c) => c.name.toLowerCase().includes(name))?.id;
  const incomeCat = cats.find((c) => c.kind === "income")?.id;

  // Second account so transfers and the wallet list have real content.
  const bca = (await api(token, "GET", "/accounts")).data[0];
  const gopay = await api(token, "POST", "/accounts", { name: "GoPay", type: "ewallet", openingBalance: 1500000 });

  const tx = (i, body) => api(token, "POST", "/transactions", body, `demo-${stamp}-${i}`);

  const seed = [
    // This month
    [0,  { type: "expense", amount: 42000,  description: "Kopi Kenangan Senopati", categoryId: cat("makanan"), accountId: bca.id, source: "voice", occurredAt: day(0) }],
    [1,  { type: "expense", amount: 185000, description: "Makan siang klien", categoryId: cat("makanan"), accountId: gopay.data.id, source: "manual", occurredAt: day(1) }],
    [2,  { type: "expense", amount: 65000,  description: "Gojek ke kantor", categoryId: cat("transport"), accountId: gopay.data.id, source: "voice", occurredAt: day(2) }],
    [3,  { type: "income",  amount: 18500000, description: "Gaji bulanan", categoryId: incomeCat, accountId: bca.id, source: "manual", occurredAt: day(4) }],
    [4,  { type: "expense", amount: 850000, description: "Langganan alat kerja", categoryId: cat("langganan") ?? cat("tagihan"), accountId: bca.id, source: "chat", occurredAt: day(4) }],
    [5,  { type: "expense", amount: 320000, description: "Belanja kebutuhan mingguan", categoryId: cat("belanja"), accountId: bca.id, source: "manual", occurredAt: day(5) }],
    [6,  { type: "expense", amount: 240000, description: "Nonton bioskop", categoryId: cat("hiburan"), accountId: gopay.data.id, source: "manual", occurredAt: day(6) }],
    [7,  { type: "expense", amount: 850000, description: "Tagihan listrik PLN", categoryId: cat("tagihan"), accountId: bca.id, source: "manual", occurredAt: day(8) }],
    [8,  { type: "income",  amount: 2500000, description: "Proyek freelance", categoryId: incomeCat, accountId: bca.id, source: "manual", occurredAt: day(9) }],
    [9,  { type: "expense", amount: 1200000, description: "Bensin & tol", categoryId: cat("transport"), accountId: bca.id, source: "manual", occurredAt: day(10) }],
    [10, { type: "transfer", amount: 2000000, description: "Top up GoPay", fromAccountId: bca.id, toAccountId: gopay.data.id, occurredAt: day(11) }],
    // Last month (period switch has data)
    [11, { type: "expense", amount: 45000,  description: "Kopi bulan lalu", categoryId: cat("makanan"), accountId: bca.id, source: "voice", occurredAt: prevMonthDay(3) }],
    [12, { type: "expense", amount: 900000, description: "Servis kendaraan", categoryId: cat("transport"), accountId: bca.id, source: "manual", occurredAt: prevMonthDay(7) }],
  ];
  for (const [i, body] of seed) await tx(i, body);

  await api(token, "POST", "/budgets", { period: "this_month", overallLimit: 12000000 });

  const deadline = new Date();
  deadline.setDate(deadline.getDate() + 210);
  await api(token, "POST", "/savings-goals", {
    name: "Dana Darurat6 Bulan",
    targetAmount: 60000000,
    deadline: deadline.toISOString().slice(0, 10),
    accountId: bca.id,
  });

  // One assistant conversation so the workspace has history too.
  await api(token, "POST", "/assistant/messages", { text: "berapa pengeluaran bulan ini" });
  await api(token, "POST", "/assistant/messages", { text: "catat makan siang35rb pakai gopay" });

  const dash = await api(token, "GET", "/analytics/dashboard?period=this_month");
  console.log(JSON.stringify({
    email,
    token,
    income: dash.data.summary.income,
    expense: dash.data.summary.expense,
    txCount: dash.data.recentTransactions.length,
  }));
}

main().catch((e) => { console.error(e); process.exit(1); });
