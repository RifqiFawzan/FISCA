# FISCA — Final UI/UX Design Specification

**Document:** `DESIGN.md`  
**Version:** 1.0  
**Date:** 8 November 2026  
**Status:** Consolidated design baseline for MVP implementation  
**Product:** FISCA — AI Personal Finance  
**Platform:** Mobile-first responsive web application  
**Language:** Indonesian for primary user-facing interface; currency default: IDR

---

## 1. Document Purpose

This document consolidates the approved FISCA visual direction, the product's UI/UX requirements, and the reusable design tokens supplied through Stitch.

It is the implementation reference for:
- UI designers creating wireframes and high-fidelity screens.
- Frontend developers implementing responsive interfaces.
- AI coding agents generating or modifying UI components.
- QA engineers validating visual behavior and interaction states.

This document defines **what the interface should look and behave like**. It does not replace the product PRD, backend/API contracts, database schema, or security architecture.

### 1.1 Source-of-truth hierarchy

When source documents differ, follow this priority:

1. **Functional behavior and product scope:** UI/UX Design PRD v0.1.
2. **Approved FISCA visual identity:** FISCA UI/UX PRD v0.3.
3. **Reusable visual tokens and component styling:** Stitch `DESIGN.md`, after the normalization rules in this document are applied.

Do not reintroduce superseded visual concepts or treat draft options as approved decisions.

### 1.2 Consolidation decisions

| Area | Final direction |
|---|---|
| Product name | FISCA |
| Product meaning | Financial Intelligence, Smart Carbon & Allocation (working brand expansion; not a UI feature claim) |
| Platform | Responsive web |
| Design approach | Mobile-first |
| Theme | Dark-first |
| Visual language | Frosted Velvet Glassmorphism, used selectively |
| Main accent | Emerald |
| Typography | Plus Jakarta Sans |
| Mobile navigation | Home, History, Smart Assistant, Reports, Profile |
| Assistant | Full-screen experience; bottom navigation hidden |
| Input modes | Voice and chat |
| Transaction capture | Hybrid recording |
| Budget model | Overall Budget + Category Budgets |
| Budget changes via AI | Proposal and explicit user approval |
| Over-budget behavior | Warning only; transactions are never blocked by the UI |
| MVP audience | Individual users; architecture/design should not assume shared household accounts |

### 1.3 Change log

Locked sections may only change together with an entry in this table (plan.md §1.2: `LOCKED` = do not change without a change log).

| Date | Section | Change | Reason |
|---|---|---|---|
| 2026-11-10 | §4.1 `text-muted` | `#64748B` → `#8493AB` | The original measured 3.0–4.1:1 on FISCA's dark surfaces and failed WCAG AA (4.5:1). `text-disabled` keeps `#64748B` deliberately: disabled text is exempt from WCAG contrast. |
| 2026-11-10 | §4.1 `primary`, `primary-hover`, `primary-deep` | `#10B981` → `#4EDEA3`, `#059669` → `#46C893`, `#047857` → `#3EB282` | Follows the Stitch "Dark Carbon Glass" palette (ISSUE-23). The previous hover/deep values actually failed WCAG (3.50 / 2.41) and were invisible to Lighthouse because it only audits the resting state; the new ones measure 6.25 / 4.96. Still the same green family, so "Main accent: Emerald" in §1.2 keeps referring to the same role. |
| 2026-11-10 | §4.1 `ai-accent`, `analytics-accent` | `#06B6D4` → `#4CD7F6`, `#6366F1` → `#C0C1FF` | Stitch `secondary` and `tertiary`; both measure 7.23 / 7.20 on the darkest surface. |
| 2026-11-10 | §4.1 `danger` | `#F43F5E` → `#F66981` | The original measured 4.16:1 on tinted badges; `#F66981` passes on every real background (worst 4.60). |
| 2026-11-10 | §4.6 primary glow | `rgba(16,185,129,0.20)` → `rgba(78,222,163,0.20)` | Follows `primary`. The `ai` and `danger` glows were left unchanged on purpose: DESIGN.md and the code still agree on them; they only differ slightly in hue from their updated accents. |
| 2026-11-21 | §4.5 glass, §4.6 glow | Blur dibatasi **2 permukaan** (Panel + bottom navigation; floating & modal jadi solid). Glow dibatasi **1 jenis** (aksi primer saja); token glow `ai`/`warning`/`danger` dihapus dari `globals.css`. | Permintaan user mengikuti aturan antislop: dosis diturunkan ke batas R-10 (maks. 2 permukaan) dan R-13 (maks. 1 jenis elemen). Sebelumnya blur ada di 6 permukaan dan glow di 5 titik, sehingga glass/glow menjadi *karakter* UI, bukan aksen — bertentangan dengan §4.5 sendiri yang berbunyi "Use glassmorphism selectively". Token yang dihapus memang tidak dipakai komponen mana pun. Rincian di `docs/issues/ISSUE-26.md`. |
| 2026-11-22 | §4.1 `text-primary`, `text-secondary`, `text-muted`, `danger` | `#F1F5F9`→`#DFE2EE`, `#94A3B8`→`#BBCABF`, `#8493AB`→`#86948A`, `#F66981`→`#FFB4AB` | Arah product owner: samakan dengan blok `colors:` di `Color_Design.md` (token Stitch "Luminescent Carbon Fintech"). Permukaan sudah sama; hanya teks & danger yang berbeda. Semua nilai lolos WCAG AA di setiap permukaan FISCA — terburuk: `text-secondary`7,20 (surface-highest), `text-muted`5,14, `danger`8,12 (di atas tint-nya sendiri), `text-primary`9,51. Audit deterministik: `tests/audit-color-yaml.mjs`. Deviasi warna sebelumnya (#64748B→#8493AB, #F43F5E→#F66981) tetap tercatat sebagai sejarah di baris sebelumnya. |

Full deviation details live in `plan.md` §17.2; the work is documented in `docs/issues/ISSUE-23.md`.

---

## 2. Product Experience Principles

### 2.1 Mobile-first, not desktop-shrink

Design the mobile experience first, using approximately 390px as the primary design reference. Adapt the layout for tablet and desktop rather than shrinking a desktop dashboard.

### 2.2 Voice-first, never voice-only

Smart Assistant is the primary action, and voice is visually prominent. Chat must remain available as a complete alternative whenever microphone permission is unavailable, denied, or inconvenient.

### 2.3 Financial information must be scannable

Prioritize:
1. Current balance.
2. Income and expense for the selected period.
3. Net cash flow.
4. Budget remaining and budget status.
5. Recent activity.

Do not make users interpret dense tables or multiple charts to understand their current financial position.

### 2.4 AI must be transparent

The interface must distinguish:
- What the user said.
- What the AI understood.
- What the system is asking the user to clarify.
- What the backend has successfully saved.
- What is only an AI recommendation or proposal.

Never display a successful transaction state before the backend confirms persistence.

### 2.5 User control

Users remain in control of financial data:
- Users can edit transactions.
- Undo is offered only while the action is eligible.
- AI budget changes require explicit approval.
- Rejecting a proposal must not change the active budget.
- AI-generated savings plans remain drafts until the user accepts them.

### 2.6 Progressive disclosure

Keep the Home dashboard focused. Put detailed filters, transaction metadata, chart breakdowns, and advanced settings behind relevant detail screens or expandable sections.

### 2.7 Accessible by default

Use readable text, sufficient contrast, visible focus states, clear labels, and touch targets suitable for mobile use. Color must never be the only way to communicate status.

---

## 3. Brand and Visual Direction

### 3.1 Visual identity

**FISCA uses a Dark-First Frosted Velvet Glassmorphism visual language.**

The intended impression:
- Calm, trustworthy, modern personal finance.
- Clear financial information rather than a trading terminal.
- Soft emerald accents for positive financial actions.
- Subtle translucent surfaces for hierarchy.
- Restrained motion and glow.
- Comfortable long-session readability.

The design must not look like a carbon-credit dashboard, crypto trading terminal, or monitoring console.

### 3.2 Eye ergonomics

Dark mode is the default visual direction, but dark colors do not automatically guarantee eye comfort.

Rules:
- Use off-white rather than pure white for primary text.
- Keep secondary text readable; do not make essential information excessively dim.
- Avoid large areas of saturated neon.
- Use glow sparingly and only to clarify focus or primary actions.
- Avoid unnecessary animation and respect `prefers-reduced-motion`.
- Ensure charts, badges, and inputs remain legible at low brightness and on ordinary non-OLED screens.
- Do not claim that the interface eliminates eye strain or guarantees zero glare.

### 3.3 Carbon terminology

The original Stitch design system contains carbon-offset, clean-energy, ESG, and ecological-credit terminology. These concepts are **not part of the FISCA personal finance MVP**.

Do not implement:
- Carbon footprint metrics.
- Carbon credits or offsets.
- Clean-grid status.
- Ecological yield or sustainability telemetry.
- Carbon surplus indicators.

Use the emerald color for positive financial states such as income, savings progress, and successful actions. Do not label these states as carbon or ecological achievements.

---

## 4. Design Tokens

The tokens below are the normalized implementation baseline. If the existing Stitch tokens conflict with this section, this section takes precedence.

### 4.1 Color tokens

#### Base surfaces

| Token | Value | Usage |
|---|---|---|
| `canvas` | `#0A0E16` | Lowest application background |
| `background` | `#0F131C` | Main page background |
| `surface-low` | `#181C24` | Subtle cards and secondary surfaces |
| `surface` | `#1C2028` | Standard card/container |
| `surface-high` | `#262A33` | Elevated controls and selected surfaces |
| `surface-highest` | `#31353E` | Highest neutral surface |
| `surface-glass` | `rgba(24, 28, 36, 0.70)` | Translucent panels |
| `input-surface` | `rgba(7, 9, 14, 0.70)` | Form controls |

#### Text

| Token | Value | Usage |
|---|---|---|
| `text-primary` | `#DFE2EE` | Headings, primary values |
| `text-secondary` | `#BBCABF` | Supporting text |
| `text-muted` | `#86948A` | Non-essential metadata only |
| `text-disabled` | `#64748B` | Disabled content, with disabled styling |
| `text-on-primary` | `#003824` | Text on bright emerald surfaces |

#### Semantic and brand accents

| Token | Value | Usage |
|---|---|---|
| `primary` | `#4EDEA3` | Main brand/action accent |
| `primary-hover` | `#46C893` | Hover/pressed emphasis |
| `primary-deep` | `#3EB282` | Gradient/deeper emerald |
| `income` | `#34D399` | Income and positive cash flow |
| `expense` | `#FB7185` | Expense and negative cash flow |
| `transfer` | `#38BDF8` | Transfer information |
| `ai-accent` | `#4CD7F6` | AI processing and assistant-specific information |
| `analytics-accent` | `#C0C1FF` | Optional analytical distinction; use sparingly |
| `warning` | `#FBBF24` | Budget warning |
| `danger` | `#FFB4AB` | Budget exceeded and destructive actions |
| `success` | `#34D399` | Confirmed success |
| `border-subtle` | `rgba(255,255,255,0.06)` | Standard panel border |
| `border-default` | `rgba(255,255,255,0.08)` | Inputs and component boundaries |
| `border-focus` | `rgba(6,182,212,0.80)` | Keyboard/input focus |

**Semantic color rules**
- Income is not the same as account balance growth; show labels and values.
- Expense is not a destructive action. Use the expense color for data visualization, not as an error state.
- Transfer must remain visually distinct and must not be counted as income or expense.
- Warning and exceeded states require text labels or icons in addition to color.
- AI accent cyan must not compete with emerald as the primary brand color.

### 4.2 Typography

**Font family:** Plus Jakarta Sans.

Use tabular numerals for balances, transaction amounts, percentages, and chart values.

| Style | Size | Weight | Line height | Usage |
|---|---:|---:|---:|---|
| Display XL | 56px | 800 | 64px | Rare marketing/empty-state hero |
| Display Mobile | 36px | 800 | 44px | Mobile hero value |
| Heading XL | 40px | 700 | 48px | Large page title |
| Heading LG | 28px | 700 | 36px | Section/page heading |
| Heading MD | 24px | 700 | 32px | Card heading |
| Heading SM | 22px | 600 | 30px | Subsection heading |
| Body LG | 18px | 400 | 28px | Introductory copy |
| Body MD | 16px | 400 | 24px | Main body |
| Body SM | 14px | 400 | 20px | Supporting copy |
| Label LG | 14px | 600 | 20px | Form labels and buttons |
| Label MD | 12px | 600 | 16px | Metadata |
| Label SM | 11px | 700 | 14px | Compact status labels |

Implementation:
- Use `font-variant-numeric: tabular-nums` for financial values.
- Avoid uppercase micro-labels for long Indonesian words.
- Do not use 11px text for essential financial information.
- Use consistent font weights; do not rely on color alone to create hierarchy.

### 4.3 Spacing

Use a 4px base scale.

| Token | Value |
|---|---:|
| `space-1` | 4px |
| `space-2` | 8px |
| `space-3` | 12px |
| `space-4` | 16px |
| `space-5` | 20px |
| `space-6` | 24px |
| `space-8` | 32px |
| `space-10` | 40px |
| `space-12` | 48px |
| `space-16` | 64px |

Default page gutters:
- Mobile: 16px.
- Tablet: 24px.
- Desktop: 32px.

### 4.4 Radius

| Token | Value | Usage |
|---|---:|---|
| `radius-sm` | 4px | Small tags and compact elements |
| `radius-md` | 8px | Inputs, buttons, list controls |
| `radius-lg` | 16px | Standard cards |
| `radius-xl` | 24px | Main panels and hero cards |
| `radius-full` | 9999px | Pills and circular controls |

### 4.5 Borders, glass, and elevation

Glassmorphism is an **accent**, not the character of the whole UI. Blur is capped
at **two surfaces** (R-10), and both have a functional reason:

1. **Panel** (cards) — the signature frosted surface.
2. **Bottom navigation** — content scrolls beneath it, so the blur is doing work.

Everything else is solid: dropdowns, popovers, dialogs, sheets, sticky headers,
and input bars use opaque surfaces. A translucent surface *without* blur lets
content bleed through and hurts readability.

| Level | Surface | Border | Usage |
|---|---|---|---|
| Floor | `#0A0E16` | None | Application canvas |
| Panel | `rgba(24,28,36,0.70)` + `blur(16px)` | `1px solid rgba(255,255,255,0.06)` | Cards and panels — **blurred** |
| Floating | `#1C2028` (opaque) | `1px solid rgba(255,255,255,0.10)` | Dropdowns, popovers, hover surfaces |
| Modal | `rgba(15,19,28,0.94)` (opaque) | `1px solid rgba(255,255,255,0.14)` | Dialogs, sheets, command surfaces |
| Bottom nav | `rgba(15,19,28,0.95)` + `blur(24px)` | `1px solid rgba(255,255,255,0.06)` | Fixed navigation — **blurred** |

Do not depend on `backdrop-filter` for readability. Provide a sufficiently opaque fallback surface for browsers that do not support it.

### 4.6 Glow and shadows

Glow marks **one thing only: the primary action** (R-13) — buttons, the FAB, and
the sidebar CTA. It must not appear on cards, badges, icons, borders, and
backgrounds at the same time; when everything glows, nothing reads as important.

- Primary action: `0 0 16px rgba(78,222,163,0.20)`.

Glow tokens for AI, warning, and danger were **removed**: no component applied
them, and keeping unused tokens in `globals.css` would contradict this table.
Reintroduce one only with a written reason (R-31).

Avoid persistent pulsing glows. Motion should communicate state, not decorate the screen.

### 4.7 Motion

- Standard transition: 150–200ms.
- Dialog/sheet transition: 180–240ms.
- Use ease-out for entering and ease-in for exiting.
- Respect `prefers-reduced-motion: reduce`.
- Do not animate financial values in a way that could make the displayed amount ambiguous.
- Waveform animation is allowed only while actual recording is active.

---

## 5. Responsive Layout

Breakpoints are initial implementation targets and should be validated through responsive testing.

| Viewport | Layout |
|---|---|
| `< 768px` | Mobile single-column; bottom navigation |
| `768–1023px` | Tablet adaptive layout; collapsible navigation |
| `>= 1024px` | Desktop sidebar; multi-column content |

### 5.1 Mobile

- Main content uses a single-column flow.
- Bottom Navigation has exactly five items.
- Respect device safe areas.
- The Assistant occupies the full screen and hides Bottom Navigation.
- Primary actions must be reachable by thumb where practical.
- Horizontal scrolling is allowed only for intentional carousels or filter rows.
- Do not allow accidental horizontal page overflow.
- Keyboard must not cover chat input, forms, or primary actions.
- Use bottom sheets for compact contextual actions when appropriate.

### 5.2 Tablet

- Use adaptive two-column layouts where they improve scanning.
- Navigation may collapse into a drawer or compact rail.
- Avoid desktop density when the available width is insufficient.

### 5.3 Desktop

- Use a persistent left sidebar.
- Use a centered content area with a reasonable maximum width.
- Dashboard cards may use a multi-column grid.
- History may use a table when it improves comparison and scanning.
- Do not add a permanent 320px telemetry dock; it is not required by FISCA's product experience.
- Assistant must remain a dedicated, distraction-free workspace. Its desktop presentation should use the available content area rather than a monitoring dashboard.

---

## 6. Information Architecture

```text
FISCA
├── Authentication
│   ├── Sign In
│   ├── Sign Up
│   └── Password Recovery
│
├── Home
│   ├── Financial Summary
│   ├── Cash Flow Overview
│   ├── Spending by Category
│   ├── Recent Transactions
│   └── Budget Shortcut
│
├── History
│   ├── Transaction List
│   ├── Search and Filters
│   └── Transaction Detail / Edit
│
├── Smart Assistant (Full Screen)
│   ├── Conversation
│   ├── Voice Interaction
│   ├── Chat Interaction
│   ├── Transaction Result
│   ├── Clarification
│   ├── Financial Analysis
│   └── Budget Proposal / Approval
│
├── Reports
│   ├── Income vs Expense
│   ├── Net Cash Flow
│   ├── Spending by Category
│   └── Monthly Spending Trend
│
├── Budget
│   ├── Overall Budget
│   ├── Category Budgets
│   ├── Unallocated Budget
│   └── AI Budget Insight
│
└── Profile
    ├── Personal Settings
    ├── Accounts / Wallets
    ├── Categories
    ├── Currency and Preferences
    ├── Notifications
    └── Privacy and Security
```

Budget can be reached from Home and Reports. Savings Goals are not a Must Have feature for the initial MVP.

---

## 7. Navigation

### 7.1 Mobile Bottom Navigation

Exactly five items, in this order:

1. Home
2. History
3. Smart Assistant
4. Reports
5. Profile

Rules:
- Smart Assistant is centered and visually elevated.
- It is the most prominent navigation action, but must not obscure adjacent controls.
- All items have accessible labels.
- Active state must use more than color alone (icon treatment, label, or indicator).
- Navigation respects safe-area insets.
- When Assistant opens, Bottom Navigation is hidden.
- Back from Assistant returns to the previous page/context.

### 7.2 Desktop Sidebar

- Home
- History
- Reports
- Budget
- Profile / Settings

Provide a clearly visible Smart Assistant CTA in the sidebar or primary application shell. Do not force the mobile bottom-navigation pattern onto desktop.

---

## 8. Screen Specifications

## 8.1 Authentication

### Sign In

Required:
- FISCA logo/wordmark.
- Email input.
- Password input with visibility toggle.
- Remember-me option if supported by the authentication implementation.
- Password recovery link.
- Primary sign-in action.
- Loading, validation, success, and error feedback.
- Short, accurate privacy explanation.

### Sign Up

Required:
- Full name.
- Email.
- Password.
- Confirm password.
- Password-strength guidance.
- Terms/privacy consent.
- Sign-up action.
- Link to Sign In.

### Authentication claims

Do not display claims such as:
- "End-to-End Encrypted".
- "Zero-Knowledge Privacy".
- "Bank-Grade 256-Bit AES".
- "100% Secure".
- "Biometric authentication available".

unless the actual implemented architecture and supported platform capabilities have been verified.

Google/Apple SSO, passkeys, WebAuthn, Face ID, and fingerprint authentication are implementation-dependent. They are not assumed to exist merely because the UI design includes a place for them.

## 8.2 Home — Financial Overview

Purpose: help users understand their current financial position quickly.

Recommended mobile content order:
1. Header: greeting, date, profile, notifications.
2. Total Balance card.
3. Income and Expense summary.
4. Net Cash Flow.
5. Cash Flow chart.
6. Spending by Category.
7. Recent Transactions.
8. Budget shortcut.

Interactions:
- Period selector updates all period-dependent values.
- Balance visibility toggle hides sensitive amounts.
- Summary cards open relevant details.
- Recent transaction items open Transaction Detail.
- Dashboard refreshes after a confirmed transaction mutation.

Data presentation:
- Always show the selected period.
- Always label currency.
- Do not imply that a balance is a bank-synced balance if it is calculated from manually recorded data.
- If the app calculates balance from transactions, explain the calculation basis in relevant detail/help UI.
- Carbon status indicators from the original design are removed.

States:
- Loading skeleton.
- First-use empty state.
- Error with retry.
- Data state with period and last-updated context where useful.

## 8.3 History — Transaction Management

Purpose: search, review, and manage recorded transactions.

Required controls:
- Search by description/merchant, category, or tag where supported.
- Date filter: This Month, Last 7 Days, Custom.
- Type filter: All, Income, Expense, Transfer.
- Category filter.
- Account filter only if account data is supported.
- Filtered summary.
- Date-grouped transaction list.

Each transaction item should show:
- Description.
- Category.
- Date/time.
- Amount.
- Transaction type.
- Account, if known.
- Optional source label: Voice, Chat, or Manual.

Transaction detail:
- Full transaction information.
- Edit action.
- Undo/cancel only while eligible.
- Change history only if supported by the backend.

Rules:
- Transfers must not be counted as income or expense.
- Unsaved drafts must not appear in the active transaction list.
- Cancelled transactions must not contribute to active summaries.
- Do not display unknown account/payment details as confirmed facts.

## 8.4 Smart Assistant — Full-Screen Experience

Purpose: one place to capture transactions, ask financial questions, search transaction history, and manage budget proposals.

Layout:
- Full-screen view.
- Bottom Navigation hidden.
- Header with Back and Assistant identity.
- Conversation area as the main content region.
- Voice control as the most prominent action.
- Chat input remains available.
- Quick prompts may appear in the empty conversation state.
- Input must remain usable when the virtual keyboard is open.

Initial interaction:
1. User opens Smart Assistant.
2. Assistant opens in Ready state.
3. The user explicitly starts voice capture or chooses chat.
4. If microphone permission is missing, explain why permission is needed and offer chat as an alternative.
5. Display the response in the conversation.
6. Back returns to the previous page.

Do not start recording unexpectedly merely because the Assistant screen was opened. Browser permission and recording must follow an explicit user action.

### Assistant response cards

| Response type | Required UI content |
|---|---|
| Transaction Result | Amount, description, category, account if known, date/time, saved status, Undo if eligible |
| Clarification | One clear question, missing/uncertain fields, quick replies |
| Financial Summary | Income, expense, net cash flow, period |
| Historical Search | Matching transactions, dates, amounts, applied filters |
| Largest Transaction | Ranked transactions, selected period, transaction type/scope |
| Unusual Spending | Flagged transaction, comparison baseline, reason, data limitation |
| Budget Proposal | Before/after values, difference, affected categories, impact, Approve/Reject/Edit |
| Savings Plan | Target, contribution, duration, assumptions; draft until accepted |
| Recommendation | Insight, supporting data, suggested action, assumptions/limitations |

### Assistant conversation states

- Empty / Welcome.
- Ready.
- Permission Required.
- Listening.
- Transcribing.
- Processing.
- Clarification Required.
- Result.
- Error.
- Cancelled.
- No Result.

Conversation history management can be expanded after the core assistant flow is stable.

## 8.5 Voice Interaction

| State | UI behavior |
|---|---|
| Ready | Microphone action available; chat available |
| Permission Required | Explain microphone permission; provide chat fallback |
| Listening | Show recording indicator/waveform and Stop/Cancel controls |
| Transcribing | Show that speech is being converted to text |
| Processing | Show that the request is being interpreted |
| Clarification | Ask for missing/ambiguous information |
| Saved | Show only after backend confirmation |
| Analysis Result | Show result and the data period used |
| Error | Explain failure and offer retry/chat |
| Cancelled | Confirm that cancelled recording/action was not saved |
| No Result | Explain that no matching data was found |

Rules:
- Never show Listening before the browser has actually started capturing audio.
- Provide a visible recording indicator while audio is being captured.
- Users can stop or cancel recording.
- Microphone failure must not disable chat.
- Prevent duplicate submissions.
- If the user attempts to leave during recording, handle cancellation safely.
- Do not imply that audio is stored unless audio storage is actually implemented and disclosed.

## 8.6 Hybrid Recording

Product behavior:
- Clear and sufficiently complete transaction input may be auto-saved after backend validation.
- Ambiguous or incomplete input must request clarification.
- The UI must not invent missing account, date, category, or payment details.

Flow:
1. Receive voice transcription or chat message.
2. Display understood details when useful.
3. Extract transaction type, amount, description, category, date, account, and payment method when explicitly available.
4. Validate the structured data through the backend.
5. If valid, save and show Transaction Result.
6. If important details are missing or ambiguous, show Clarification.
7. Validate again after the user responds.
8. Save only after successful validation.

Clarification UI:
- Ask one focused question at a time where possible.
- Offer quick reply options when the ambiguity has a small known set of choices.
- Allow free-text or voice response.
- Preserve the original user context during clarification.

Undo:
- Show Undo only when the backend confirms that the action is reversible.
- The Undo window and rules must be defined by product/backend implementation.
- After Undo expires, direct users to edit or cancel from History if supported.

## 8.7 Reports — Financial Analytics

MVP reports:
- Income vs Expense.
- Net Cash Flow.
- Spending by Category.
- Monthly Spending Trend.
- Period Selector.

Chart requirements:
- Responsive on mobile and desktop.
- Clear labels, legends, and tooltips.
- Show period and currency.
- Distinguish Income, Expense, and Transfer using labels and visual treatment, not color alone.
- Do not plot misleading zero/empty data as if it were a real trend.
- Provide an empty state when there is insufficient data.
- Keep chart labels readable; do not force dense axis labels onto narrow screens.

## 8.8 Budget Management — Combined Budget

Model:
- Overall Budget.
- Category Budgets.
- Unallocated Budget.

Budget Overview content:
1. Period selector.
2. Overall Budget limit.
3. Total spent.
4. Remaining amount.
5. Overall progress.
6. Category Budget list.
7. Unallocated amount.
8. AI Budget Insight.
9. CTA to manage budget or open Assistant.

Each category item:
- Category name.
- Budget limit.
- Actual spending.
- Percentage used.
- Remaining or exceeded amount.
- Status label.

Thresholds currently proposed:
- Normal: below 80%.
- Warning: 80% to below 100%.
- Exceeded: 100% or above.

The 80% warning threshold remains configurable until explicitly finalized.

Rules:
- Budget over-limit is Warning Only.
- The UI must never block a transaction solely because a budget has been exceeded.
- Spending in a category without a category budget still contributes to Overall Budget.
- Show the exceeded amount, not just a red state.
- AI must not update active budgets without user approval.
- A multi-category change is presented as one coherent proposal.
- The sum of category allocations must be validated against the Overall Budget according to the product's allocation rules.

## 8.9 Budget Proposal and Approval

Proposal card must display:
- Current value.
- Proposed value.
- Difference.
- Affected category/categories.
- Budget period.
- Reason/context.
- Effect on Overall Budget and unallocated amount.
- Pending Approval status.
- Approve, Reject, and Edit actions.

Rules:
- No active budget changes before approval succeeds.
- Approval must refer to a specific proposal/version.
- If the underlying budget has changed since proposal creation, require revalidation.
- On successful approval, refresh and display the confirmed budget values.
- On failure, retain the previous active budget and show an error.
- Rejecting a proposal leaves the active budget unchanged.

## 8.10 Profile and Settings

Sections:
- Personal profile.
- Accounts / Wallets.
- Categories.
- Currency and number format.
- Notification preferences.
- AI preferences.
- Privacy and security.
- Sign out.

The account/wallet model is not fully decided. Keep the UI flexible for Cash, bank accounts, and e-wallets, but do not imply automatic bank synchronization.

## 8.11 Savings Goals

**Priority: Should Have / later MVP iteration.**

Potential UI:
- Goal name.
- Target amount.
- Current allocated amount.
- Remaining amount.
- Target date.
- Progress.
- Suggested recurring contribution.

AI-generated savings plans remain drafts until the user accepts them. Any action that changes persisted financial data requires user approval.

---

## 9. Reusable Component Inventory

Suggested component structure:

```text
components/
├── app-shell/
│   ├── AppLayout
│   ├── MobileBottomNavigation
│   ├── DesktopSidebar
│   └── PageHeader
│
├── authentication/
│   ├── AuthLayout
│   ├── SignInForm
│   ├── SignUpForm
│   ├── PasswordField
│   └── PasswordStrengthIndicator
│
├── assistant/
│   ├── AssistantLayout
│   ├── AssistantHeader
│   ├── ConversationList
│   ├── MessageBubble
│   ├── QuickPromptList
│   ├── VoiceRecorder
│   ├── AudioWaveform
│   ├── ChatInput
│   ├── TransactionResultCard
│   ├── ClarificationCard
│   ├── FinancialInsightCard
│   ├── HistoricalTransactionCard
│   └── BudgetProposalCard
│
├── dashboard/
│   ├── BalanceCard
│   ├── BalanceVisibilityToggle
│   ├── IncomeExpenseSummary
│   ├── NetCashFlowCard
│   ├── CashFlowChart
│   ├── CategoryBreakdown
│   └── RecentTransactions
│
├── transactions/
│   ├── TransactionList
│   ├── TransactionListItem
│   ├── TransactionSearch
│   ├── TransactionFilters
│   ├── TransactionDetail
│   └── TransactionForm
│
├── reports/
│   ├── PeriodSelector
│   ├── IncomeExpenseChart
│   ├── CashFlowTrendChart
│   └── SpendingCategoryChart
│
├── budget/
│   ├── OverallBudgetCard
│   ├── CategoryBudgetItem
│   ├── BudgetProgress
│   ├── UnallocatedBudget
│   ├── BudgetInsightCard
│   └── BudgetApprovalSheet
│
└── shared/
    ├── Button
    ├── Input
    ├── Select
    ├── Badge
    ├── Card
    ├── Dialog
    ├── BottomSheet
    ├── Toast
    ├── EmptyState
    ├── LoadingState
    ├── ErrorState
    └── ConfirmationDialog
```

Component rules:
- Components receive data and callbacks through explicit props.
- Reusable visual components must not contain financial calculation/business rules.
- Keep formatting helpers centralized.
- Use consistent loading, disabled, focus, success, warning, and error states.
- Avoid creating one-off components when an existing shared component can be extended safely.

---

## 10. Interaction and Feedback Standards

Every user action must have appropriate feedback.

| Situation | Required behavior |
|---|---|
| Request in progress | Loading/disabled state |
| Successful save | Confirmed success state |
| Failed request | Human-readable error and recovery action |
| Empty data | Explain why empty and what user can do |
| Destructive action | Confirmation where appropriate |
| Undo | Available only when supported and still valid |
| Validation error | Display near the relevant field |
| Duplicate submission | Prevent or safely deduplicate |
| Budget proposal | Explicit pending/approved/rejected state |
| Microphone unavailable | Chat fallback |

Use toast notifications for brief feedback. Keep important transaction and budget outcomes visible in the relevant card or detail screen.

---

## 11. Accessibility and Usability

- Minimum target touch area: 44 × 44 CSS pixels.
- All icon-only buttons require accessible names.
- All form controls require labels.
- Keyboard navigation must work.
- Focus states must be visible.
- Maintain adequate text and control contrast.
- Status must not rely on color alone.
- Respect `prefers-reduced-motion`.
- Waveform and loading animations cannot be the only status indicator.
- Core tasks must remain possible through chat without voice.
- Use natural, consistent Indonesian for primary UI copy.
- Ensure text remains readable when browser zoom is increased.
- Do not use tiny metadata text for amounts, dates, or important transaction details.

---

## 12. Financial Data Presentation and Trust

- Every financial amount must have an explicit currency context.
- Use consistent Indonesian Rupiah formatting, e.g. `Rp1.250.000`.
- Show the relevant date range on reports and summaries.
- Keep balance, income, expense, transfer, and budget remaining conceptually separate.
- Transfers must not inflate income or expense totals.
- Clearly distinguish actual records from AI suggestions.
- Show the period and scope used by financial analysis.
- For Unusual Spending, show the comparison baseline or explain why the result is flagged.
- If history is insufficient, state that limitation.
- Label prototype/sample data as Demo.
- Do not present manually calculated transaction balance as a verified bank balance.

---

## 13. Privacy and Security UX

- Financial information is visible only within the authenticated user's session.
- Do not expose another user's records in search, reports, or assistant responses.
- Explain microphone permission and show when recording is active.
- Provide stop and cancel recording controls.
- Do not claim that audio is stored or deleted unless the implementation supports that statement.
- Provide access to privacy and data-management settings.
- Avoid unsupported security marketing claims.
- UI privacy indicators must reflect actual system state.

The UI specification does not define encryption algorithms, key management, authorization, tenant isolation, or backend security controls. Those belong in the technical/security design and must be verified separately.

---

## 14. UI Copy Guidelines

Use clear Indonesian, short labels, and direct action verbs.

Preferred examples:
- "Catat pengeluaran"
- "Pemasukan"
- "Pengeluaran"
- "Transfer"
- "Sisa anggaran"
- "Anggaran hampir habis"
- "Anggaran terlampaui"
- "Simpan transaksi"
- "Batalkan"
- "Setujui perubahan"
- "Tolak usulan"
- "Coba lagi"
- "Gunakan chat"

Avoid:
- Technical AI terms in ordinary user-facing messages.
- Claims such as "100% akurat" or "100% aman".
- Vague success messages when the actual operation is still pending.
- Unexplained English status labels when a natural Indonesian label is clearer.

Technical names may remain in code and internal documentation.

---

## 15. MVP Scope and Priority

### Must Have

1. Authentication screens and states.
2. Mobile Bottom Navigation and Desktop Sidebar.
3. Home Financial Overview.
4. Transaction History, search, filters, and detail.
5. Full-screen Smart Assistant.
6. Voice and Chat interaction.
7. Voice permission, recording, transcription, processing, cancellation, and error states.
8. Hybrid Recording.
9. Transaction Result and Clarification cards.
10. Basic Reports.
11. Combined Budget overview.
12. Budget Proposal and Approval UI.
13. Profile and basic settings.
14. Loading, empty, error, success, and warning states.
15. Responsive behavior and accessibility baseline.

### Should Have

1. Savings Goals.
2. Advanced transaction filters.
3. Period comparison.
4. Richer AI financial insight cards.
5. More complete conversation history management.
6. Additional account/wallet management capabilities.

### Future

1. Realtime voice conversation.
2. Recurring transactions.
3. Advanced cash-flow forecasting.
4. Bank integrations.
5. Receipt scanning.
6. Advanced financial planning.

Do not silently move Should Have or Future features into the first MVP implementation.

---

## 16. Open Product Decisions

These items remain unresolved by the supplied source documents and must not be guessed by the design or implementation team.

| Decision | Current status |
|---|---|
| Final logo and brand mark | Not supplied |
| Exact desktop Assistant presentation | Dedicated page/workspace preferred; detailed layout requires prototype |
| Account/wallet model | Single vs multiple vs hybrid not finalized |
| Multi-currency behavior | IDR is initial default; broader support undecided |
| Budget warning threshold | 80% proposed; configurable/not final |
| Undo time window | Must be defined by product/backend |
| Savings Goals in initial MVP | Should Have; not Must Have |
| Google/Apple SSO | Depends on authentication implementation |
| Passkeys/biometrics | Depends on platform and WebAuthn implementation |
| Audio retention/storage | Not specified; must be decided before making UI claims |
| Notification behavior | Preferences and delivery scope need product definition |
| Exact charting library | Not specified by source documents |
| Exact icon library | Not specified by source documents |

---

## 17. Acceptance Criteria

### Navigation
- [ ] Mobile has exactly five Bottom Navigation items.
- [ ] Smart Assistant is centered and visually prominent.
- [ ] Opening Assistant hides Bottom Navigation.
- [ ] Assistant has a clear Back action.
- [ ] Desktop uses a suitable sidebar/navigation layout.

### Home
- [ ] Balance, income, expense, and net cash flow are clearly distinguished.
- [ ] Period selection updates relevant values.
- [ ] Balance visibility toggle works.
- [ ] Recent transactions open their detail.
- [ ] Empty, loading, and error states are designed.

### History
- [ ] Search and basic filters are available.
- [ ] Transactions are grouped by date.
- [ ] Income, Expense, and Transfer are distinguishable.
- [ ] Transfer is excluded from income/expense totals.
- [ ] Unsaved drafts and cancelled transactions are excluded from active history/summaries.
- [ ] Transaction detail and supported edit/cancel actions are available.

### Assistant
- [ ] Assistant supports Voice and Chat.
- [ ] Assistant opens full-screen on mobile.
- [ ] Chat remains usable without microphone permission.
- [ ] Recording has Ready, Permission, Listening, Transcribing, Processing, Cancelled, and Error states.
- [ ] User can stop/cancel recording.
- [ ] Duplicate submissions are prevented or safely handled.
- [ ] Transaction success is displayed only after backend confirmation.
- [ ] Clarification is shown for incomplete or ambiguous input.
- [ ] AI analysis shows relevant period/scope and limitations.

### Budget
- [ ] Overall Budget and Category Budgets are displayed.
- [ ] Remaining and unallocated budget are clear.
- [ ] Warning and Exceeded states include labels and amounts.
- [ ] Transactions remain recordable when budget is exceeded.
- [ ] AI proposals show before/after values and impact.
- [ ] Active budget changes only after successful approval.
- [ ] Rejecting a proposal leaves active budget unchanged.
- [ ] Stale proposals are revalidated.

### Reports
- [ ] Charts work on mobile and desktop.
- [ ] Period and currency are visible.
- [ ] Empty/insufficient-data states are available.
- [ ] Charts do not cause unintended horizontal overflow.
- [ ] Data categories are not distinguished by color alone.

### Responsive and Accessibility
- [ ] No unintended horizontal overflow.
- [ ] Keyboard does not obscure critical inputs/actions.
- [ ] Bottom Navigation respects safe areas.
- [ ] Important touch targets are at least 44 × 44 CSS pixels.
- [ ] Keyboard focus is visible.
- [ ] Icon-only controls have accessible names.
- [ ] Reduced-motion preference is respected.

### Trust and Privacy
- [ ] UI never claims a transaction is saved before server confirmation.
- [ ] UI does not present unknown details as confirmed.
- [ ] Financial records are scoped to the authenticated user.
- [ ] Microphone state is visible and controllable.
- [ ] Security and encryption claims match implemented capabilities.

---

## 18. Design Delivery and Definition of Done

Design is ready for frontend implementation when:

- [ ] All Must Have screens have mobile and desktop designs.
- [ ] Primary user flows are documented.
- [ ] Voice and chat recording flows are prototyped.
- [ ] Hybrid Recording and Clarification flows are defined.
- [ ] Combined Budget and Approval flows are testable.
- [ ] Loading, empty, error, success, warning, and disabled states are covered.
- [ ] Design tokens are implemented consistently.
- [ ] Reusable component inventory is documented.
- [ ] Responsive behavior is specified.
- [ ] Open decisions affecting implementation are resolved or explicitly tracked.
- [ ] Developers can implement the screens without guessing core interaction behavior.

Recommended design sequence:
1. Sitemap and user flows.
2. Mobile wireframes.
3. Assistant interaction prototype.
4. Desktop adaptation.
5. Design token and component library.
6. High-fidelity MVP screens.
7. Responsive and accessibility review.
8. Developer handoff.

---

## 19. Implementation Notes for Designers and AI Coding Agents

1. Treat this file as the visual and UI behavior baseline for FISCA.
2. Do not invent features outside the defined MVP scope.
3. Do not introduce carbon, ESG, clean-energy, or carbon-credit UI.
4. Do not add a permanent telemetry dock or unrelated monitoring widgets.
5. Use the defined semantic colors consistently.
6. Use Plus Jakarta Sans and tabular numerals for financial values.
7. Reuse shared components and tokens rather than hardcoding one-off styles.
8. Keep business calculations out of presentational components.
9. Keep financial data formatting consistent across Home, History, Reports, Budget, and Assistant.
10. Distinguish pending, confirmed, failed, cancelled, and draft states.
11. Do not fabricate transaction data or AI analysis.
12. Do not claim security capabilities that are not implemented.
13. When a product decision is unresolved, record it as an open decision instead of silently choosing a behavior.
14. Keep mobile as the primary experience while ensuring desktop remains fully usable.

---

**End of document — FISCA Final UI/UX Design Specification v1.0**
