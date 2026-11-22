# Smoke test end-to-end API FISCA.
# Jalankan: powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1
# Arahkan ke instance lain: $env:FISCA_BASE="http://localhost:8081/api/v1"
$ErrorActionPreference = "Stop"
$Base = if ($env:FISCA_BASE) { $env:FISCA_BASE } else { "http://localhost:8080/api/v1" }
# Auth lokal (ISSUE-15): tidak ada lagi token `dev:<uuid>`, jadi smoke mendaftar
# dua akun tetap lalu masuk. Run berikutnya memakai akun yang sama (409 -> login),
# sehingga tidak menumpuk akun baru setiap kali dijalankan.
$SmokePassword = "kata-sandi-smoke-123"
$Headers = @{ "Content-Type" = "application/json" }
function Call($Method, $Path, $Body, $ExtraHeaders) {
    $h = @{}
    $Headers.GetEnumerator() | ForEach-Object { $h[$_.Key] = $_.Value }
    if ($ExtraHeaders) { $ExtraHeaders.GetEnumerator() | ForEach-Object { $h[$_.Key] = $_.Value } }
    $args = @{ Method = $Method; Uri = "$Base$Path"; Headers = $h; UseBasicParsing = $true }
    if ($Body -ne $null) { $args.Body = ($Body | ConvertTo-Json -Depth 10 -Compress) }
    try {
        $r = Invoke-WebRequest @args
        return @{ Status = [int]$r.StatusCode; Body = ($r.Content | ConvertFrom-Json) }
    } catch {
        $resp = $_.Exception.Response
        if ($resp) {
            $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
            $text = $reader.ReadToEnd()
            return @{ Status = [int]$resp.StatusCode; Body = ($text | ConvertFrom-Json) }
        }
        throw
    }
}

# Mendapatkan token sesi: coba daftar dulu, dan bila akun sudah ada (409),
# masuk memakai kredensial yang sama.
function Get-SmokeToken($Email) {
    $body = @{ email = $Email; password = $SmokePassword }
    $register = Call POST "/auth/register" $body
    if ($register.Status -eq 201) { return $register.Body.data.token }
    $login = Call POST "/auth/login" $body
    if ($login.Status -eq 200) { return $login.Body.data.token }
    throw "gagal mendapatkan token untuk $Email (register=$($register.Status) login=$($login.Status))"
}

Write-Output "== auth lokal (ISSUE-15) =="
$TokenA = Get-SmokeToken "smoke-a@fisca.test"
$TokenB = Get-SmokeToken "smoke-b@fisca.test"
$Headers = @{ Authorization = "Bearer $TokenA"; "Content-Type" = "application/json" }
Write-Output "tokenA siap; tokenB disiapkan untuk uji isolasi"

Write-Output "== GET /me =="
$me = Call GET "/me"
Write-Output "status=$($me.Status) name=$($me.Body.data.displayName) tz=$($me.Body.data.timezone)"

Write-Output "== GET /accounts =="
$accounts = Call GET "/accounts"
$accounts.Body.data | ForEach-Object { Write-Output ("- {0} {1} balance={2}" -f $_.name, $_.type, $_.balance) }
$accountId = $accounts.Body.data[0].id

Write-Output "== GET /categories =="
$categories = Call GET "/categories"
$categories.Body.data | ForEach-Object { Write-Output ("- {0} [{1}]" -f $_.name, $_.kind) }
$expenseCat = ($categories.Body.data | Where-Object { $_.kind -eq "expense" } | Select-Object -First 1)
$incomeCat = ($categories.Body.data | Where-Object { $_.kind -eq "income" } | Select-Object -First 1)

Write-Output "== POST /transactions (idempotent) =="
$key = "smoke-" + [guid]::NewGuid().ToString()
$txBody = @{ type = "expense"; amount = 25000; description = "Smoke test kopi"; categoryId = $expenseCat.id; accountId = $accountId; source = "manual" }
$tx1 = Call POST "/transactions" $txBody @{ "Idempotency-Key" = $key }
$tx2 = Call POST "/transactions" $txBody @{ "Idempotency-Key" = $key }
Write-Output "first=$($tx1.Status) id=$($tx1.Body.data.transaction.id) warnings=$($tx1.Body.data.warnings.Count)"
Write-Output "replay=$($tx2.Status) sameId=$($tx2.Body.data.transaction.id -eq $tx1.Body.data.transaction.id)"
$txId = $tx1.Body.data.transaction.id

Write-Output "== POST /transactions same key different body =="
$tx3 = Call POST "/transactions" @{ type = "expense"; amount = 99999; description = "Beda body"; accountId = $accountId; source = "manual" } @{ "Idempotency-Key" = $key }
Write-Output "status=$($tx3.Status) code=$($tx3.Body.error.code)"

Write-Output "== GET /transactions =="
$list = Call GET "/transactions?limit=5"
Write-Output "count=$($list.Body.data.items.Count) next=$($list.Body.meta.nextCursor)"

Write-Output "== GET /transactions/{id} =="
$detail = Call GET "/transactions/$txId"
Write-Output "status=$($detail.Status) auditRows=$($detail.Body.data.audit.Count)"

Write-Output "== PATCH /transactions/{id} =="
$patch = Call PATCH "/transactions/$txId" @{ version = 1; amount = 30000; description = "Smoke test kopi (edit)" }
Write-Output "status=$($patch.Status) amount=$($patch.Body.data.amount) version=$($patch.Body.data.version)"

Write-Output "== GET /analytics/dashboard =="
$dash = Call GET "/analytics/dashboard?period=this_month"
Write-Output "period=$($dash.Body.data.period.label) income=$($dash.Body.data.summary.income) expense=$($dash.Body.data.summary.expense) topCats=$($dash.Body.data.topCategories.Count)"

Write-Output "== GET /transactions/search =="
$search = Call GET "/transactions/search?q=pengeluaran%20bulan%20ini%20di%20atas%2010rb"
Write-Output "interpreted=$($search.Body.data.filters.interpreted) count=$($search.Body.data.items.Count)"

Write-Output "== POST /budgets (reuse existing for this_month) =="
$budgetBody = @{ period = "this_month"; overallLimit = 1000000; allocations = @(@{ categoryId = $expenseCat.id; limitAmount = 500000 }) }
$budget = Call POST "/budgets" $budgetBody
if ($budget.Status -eq 409) {
    $existing = Call GET "/budgets"
    $budgetId = ($existing.Body.data | Where-Object { $_.status -eq "active" } | Select-Object -First 1).id
    Write-Output "reused existing budget id=$budgetId"
} else {
    $budgetId = $budget.Body.data.id
}
Write-Output "status=$($budget.Status) id=$budgetId unallocated=$($budget.Body.data.unallocated)"

Write-Output "== POST /transactions near budget threshold =="
$big = Call POST "/transactions" @{ type = "expense"; amount = 420000; description = "Smoke test besar"; categoryId = $expenseCat.id; accountId = $accountId; source = "manual" } @{ "Idempotency-Key" = "smoke-big-" + [guid]::NewGuid().ToString() }
Write-Output "status=$($big.Status) warnings=$($big.Body.data.warnings.Count) first=$($big.Body.data.warnings[0].type)"

Write-Output "== POST /budget-proposals + approve =="
$changes = @{ overallLimit = 1500000; allocations = @(@{ categoryId = $expenseCat.id; limitAmount = 600000 }) }
$proposal = Call POST "/budget-proposals" @{ budgetId = $budgetId; changes = $changes; rationale = "Smoke test" }
Write-Output "proposal=$($proposal.Status) id=$($proposal.Body.data.id) base=$($proposal.Body.data.baseVersion)"
$approve = Call POST "/budget-proposals/$($proposal.Body.data.id)/approve" @{ baseVersion = $proposal.Body.data.baseVersion } @{ "Idempotency-Key" = "smoke-approve-" + [guid]::NewGuid().ToString() }
Write-Output "approve=$($approve.Status) limit=$($approve.Body.data.overallLimit) version=$($approve.Body.data.version)"

Write-Output "== POST /assistant/messages (heuristic) =="
$assistant = Call POST "/assistant/messages" @{ text = "catat makan siang 25rb"; source = "text" }
Write-Output "status=$($assistant.Status) card=$($assistant.Body.data.card.type) reply=$($assistant.Body.data.reply)"
$convId = $assistant.Body.data.conversationId

Write-Output "== GET /assistant/conversations =="
$convs = Call GET "/assistant/conversations"
Write-Output "count=$($convs.Body.data.Count)"

Write-Output "== GET /assistant/conversations/{id} =="
$msgs = Call GET "/assistant/conversations/$convId"
Write-Output "messages=$($msgs.Body.data.Count) roles=$(($msgs.Body.data | ForEach-Object { $_.role }) -join ',')"

Write-Output "== POST /assistant/messages (summary) =="
$summary = Call POST "/assistant/messages" @{ text = "berapa pengeluaran bulan ini"; source = "text" }
Write-Output "card=$($summary.Body.data.card.type) summaryExpense=$($summary.Body.data.card.summary.expense)"

Write-Output "== POST /assistant/messages (search) =="
$asearch = Call POST "/assistant/messages" @{ text = "cari transaksi kopi"; source = "text" }
Write-Output "card=$($asearch.Body.data.card.type) items=$($asearch.Body.data.card.items.Count)"

Write-Output "== POST /speech/transcriptions without key =="
$speech = Call POST "/speech/transcriptions" $null
Write-Output "status=$($speech.Status) code=$($speech.Body.error.code)"

Write-Output "== cross-user isolation =="
$otherHeaders = @{ Authorization = "Bearer $TokenB"; "Content-Type" = "application/json" }
$other = Call GET "/transactions" $null $otherHeaders
Write-Output "otherStatus=$($other.Status) otherCount=$($other.Body.data.items.Count)"

Write-Output "== void transaction =="
$void = Call POST "/transactions/$($big.Body.data.transaction.id)/void" @{ reason = "smoke test" } @{ "Idempotency-Key" = "smoke-void-" + [guid]::NewGuid().ToString() }
Write-Output "status=$($void.Status) txStatus=$($void.Body.data.status)"

Write-Output "DONE"
