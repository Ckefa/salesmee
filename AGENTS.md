# AGENTS.md — SalesMee

## Project Overview

SalesMee connects businesses with their clients. Businesses manage products, services, orders, bookings, payments, and client conversations through a dashboard + chat interface.

## Tech Stack

- **Backend:** Go 1.x, Gin web framework, GORM (PostgreSQL)
- **Frontend:** Tailwind CSS v3, HTMX, vanilla JS (no framework)
- **Templating:** Go `html/template` with `{{define}}`/`{{template}}` blocks
- **CSS:** Custom CSS vars in `web/static/css/styles.css` → built to `web/static/css/dist/styles.css` via Tailwind
- **Auth:** JWT tokens (cookie + bearer), OTP for clients, OAuth (Google, Facebook)
- **Payments:** Stripe, Paddle (subscriptions); cash/card/bank/mobile (in-person)
- **AI:** Groq API (Llama 3.1 8B) for salesmee Assist chatbot

## Build & Run

```sh
npm run css:build
go run ./cmd/web
go build ./...
go vet ./...
```

## Template Functions (defined in `cmd/server/main.go`)

| Function | Signature | Description |
|----------|-----------|-------------|
| `currencySymbol` | `func(string) string` | Returns symbol for currency code (e.g. `"USD"` → `"$"`) |
| `add` | `func(a, b float64) float64` | Addition |
| `sub` | `func(a, b float64) float64` | Subtraction |
| `mul` | `func(a, b float64) float64` | Multiplication |
| `div` | `func(a, b float64) float64` | Division |
| `float` | `func(i int) float64` | Int to float64 |
| `title` | `func(string) string` | `strings.Title` (capitalizes each word) |
| `default` | `func(def, val interface{})` | Returns `def` if `val` is nil/empty |
| `hasPrefix` | `strings.HasPrefix` | String prefix check |
| `dict` | `func(...interface{}) map[string]interface{}` | Builds a map from key-value pairs |
| `json` | `func(interface{}) template.JS` | JSON marshals to raw JS (returns `template.JS` so `html/template` injects as raw JS in `<script>` blocks, not wrapped in JS string quotes) |
| `formatDate` | `func(time.Time) string` | Formats as `"Jan 2, 2006"` |
| `formatTime` | `func(time.Time) string` | Formats as `"3:04 PM"` |
| `fbLogin` | `func() bool` | Returns true if FB_LOGIN env var is set |
| `seq` | `func(start, end int) []int` | Generates a range of ints (e.g. `seq 1 3` → `[1,2,3]`) |
| `percent` | `func(current, total int) int` | Returns percentage (0–100) |
| `printf` | `func(string, ...interface{}) string` | `fmt.Sprintf` — formatted string output |
| `substr` | `func(s string, start, length int) string` | Unicode-safe substring (handles multi-byte chars) |

## 4-Step Color System (Orders & Bookings)

Every order/booking status maps to 4 colored steps used across all surfaces (chat cards, dashboard tables, stat icons, step buttons):

| Step | Status(es) | Color | CSS Var | Badge Classes |
|------|-----------|-------|---------|---------------|
| 1 — Pending | `pending` | Yellow | `--color-warning` | `bg-[var(--color-warning-light)] text-[var(--color-warning)]` |
| 2 — Confirmed | `client_confirmed`, `confirmed` | Blue | `--color-info` | `bg-[var(--color-info-light)] text-[var(--color-info)]` |
| 3 — Paid | n/a (paid_amount check) | Purple | `--color-secondary` | `bg-[var(--color-secondary-light)] text-[var(--color-secondary)]` |
| 4 — Completed | `fulfilled`, `completed` | Green | `--color-success` | `bg-[var(--color-success-light)] text-[var(--color-success)]` |
| Cancelled | `cancelled` | Red | `--color-error` | `bg-[var(--color-error-light)] text-[var(--color-error)]` |
| Draft (orders only) | `draft` | Neutral | `--color-surface-tertiary` | `bg-[var(--color-surface-tertiary)] text-[var(--color-text-secondary)]` |

**Active button:** full bg + white text (e.g. `bg-[var(--color-warning)] text-white`).
**Dimmed:** light bg + colored text + `opacity-40 cursor-default`.
**Always show all 4 step buttons** — only the next logical step is clickable.

### Order lifecycle
`draft` → `pending` (step 1) → `client_confirmed`/`confirmed` (step 2) → paid (step 3) → `fulfilled`/`completed` (step 4)

### Booking lifecycle
`pending` (step 1) → `client_confirmed` (step 2) → paid (step 3) → `completed` (step 4)

### Display labels
- `fulfilled` and `completed` both display as **"Completed"** — DB stays `fulfilled` for orders
- `$isFulfilled` checks: `{{$isFulfilled := or (eq $status "fulfilled") (eq $status "completed")}}`

## CSS Variables

Defined in `:root` (light) and `.dark` (dark) in `styles.css`:

| Variable | Light | Dark | Usage |
|----------|-------|------|-------|
| `--color-warning` | `#f59e0b` | `#fbbf24` | Pending (step 1) |
| `--color-info` | `#0ea5e9` | `#38bdf8` | Confirmed (step 2) |
| `--color-secondary` | `#8b5cf6` | `#a78bfa` | Paid (step 3) |
| `--color-success` | `#10b981` | `#34d399` | Completed (step 4) |
| `--color-error` | `#f43f5e` | `#fb7185` | Cancelled |
| `--color-primary` | `#0d9488` | `#2dd4bf` | Brand teal (not status) |
| `--color-primary-light` | `#ccfbf1` | `#134e4a` | Brand light |
| `--color-primary-dark` | `#0f766e` | `#5eead4` | Brand dark |
| `--color-surface` | `#ffffff` | `#1e293b` | Card/panel bg |
| `--color-surface-secondary` | `#f1f5f9` | `#334155` | Hover/active bg |
| `--color-surface-tertiary` | `#e2e8f0` | `#475569` | Dimmed state |
| `--color-text` | `#0f172a` | `#f1f5f9` | Primary text |
| `--color-text-secondary` | `#64748b` | `#94a3b8` | Muted text |
| `--color-text-muted` | `#94a3b8` | `#64748b` | Very muted |
| `--color-border` | `#e2e8f0` | `#334155` | Borders |

Always use `var(--color-*)` via Tailwind arbitrary value syntax: `bg-[var(--color-warning-light)] text-[var(--color-warning)]`.

## Template Variable Naming

- `$isPending`, `$isConfirmed`, `$isCompleted`, `$isCancelled`, `$isFulfilled` — boolean status checks
- `$isFullyPaid := ge .PaidAmount .TotalAmount` — payment check gates step 3→4
- `$isDraft` — orders only
- `$isClientConfirmed` — orders only (distinct from business `$isConfirmed`)
- `$active := .ActivePage` — identifies which sidebar item is active

## Middleware Stack

All middleware is in `internal/middleware/`:

| Middleware | File | Description |
|-----------|------|-------------|
| `BizzMiddleware()` | `auth.go` | JWT auth for business owners and team members. Sets `business_id`, `auth_type` ("owner"/"team"), `role`, `team_member_id` in context. Redirects to login on failure. |
| `CSRFMiddleware()` | `csrf.go` | Generates CSRF token on GET, validates on POST/PUT/DELETE. Skips webhook endpoints. Provides `TemplateData()` helper to inject CSRFToken/AuthType/Role into gin.H. |
| `RateLimitGlobal()` | `ratelimit.go` | Token bucket rate limiter (5/s auth, 60/s API). Background cleanup every 10min. |
| `RequirePermission(perm)` | `roles.go` | Checks team member permissions against stored JSON. Owners get all permissions. |
| `RequireOwner()` | `roles.go` | Restricts to `auth_type == "owner"`. |
| `RequireFeature(feature)` | `subscription.go` | Gates features behind subscription plan. |
| `CheckResourceLimit(resource)` | `subscription.go` | Checks resource limits (clients, products, etc.) against plan. |

### Permission Constants

Defined in `roles.go`: `PermOrdersRead`/`Write`, `PermBookingsRead`/`Write`, `PermClientsRead`/`Write`, `PermProductsWrite`, `PermServicesWrite`, `PermPaymentsWrite`, `PermAnalyticsView`, `PermReportsView`, `PermLocationsView`.

## Onboarding Feature (5-Step Guided Setup)

Shown as a centered modal overlay only on `/business` page when business has `onboarding_step < 6`.

### 5-Step Flow
| Step | Label | Auto-advance condition |
|------|-------|------------------------|
| 1 | Welcome | Manual ("Let's Go!" button) |
| 2 | Products & Services | `hasProducts || hasServices` |
| 3 | Customize Profile | `hasLogo` |
| 4 | Share & Connect | `hasClients` |
| 5 | Place a Test Order | `hasOrdersOrBookings` |

### Detection Logic (`internal/services/onboarding/detector.go`)
- `DetectStep()` runs on every page load and key actions — queries counts, compares against `business.onboarding_step`, updates DB if changed.
- If current step's condition is met, advances to the next step (`detectedStep == N → N+1`).
- Completed when `detectedStep > 5` (step 6 = completed).

### Routes (`internal/handlers/business/onboarding.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/onboarding/status` | `GetOnboardingStatus` | Returns current step + total_steps |
| `POST /business/onboarding/advance` | `AdvanceOnboarding` | Manual advance (e.g. "Let's Go!") |
| `POST /business/onboarding/progress` | `CheckOnboardingProgress` | Re-runs detection, advances if condition met |
| `POST /business/onboarding/skip` | `SkipOnboarding` | Marks as completed |

### JS (`web/static/js/modules/onboarding.js`)
- `onboardingAdvance()` — POST to `/advance`
- `onboardingCheck()` — POST to `/progress`
- `onboardingMinimize()` / `onboardingExpand()` — toggle panel state
- `onboardingClose()` — fade out and hide
- `onboardingSkip()` — POST to `/skip`
- `onboardingStartOrder()` — loads first client chat + opens product picker
- `pollOnboarding()` — polls `/status` every 5s, reloads if step changed

### Template (`web/templates/components/onboarding/onboarding_panel.html`)
- `{{template "onboarding_panel" .}}` — included only in `business.html`
- Uses `.Onboarding`, `.Business`, `.Clients` (first client for step 5 button)
- Requires `.Business.ID`, `.Business.Slug` accessible in template context

## WhatsApp-Style Sidebar Sorting

Both business client list and client business list use a 4-priority client-side sort (`sortClientList()` / `sortBusinessList()`) re-triggered after any WS event that changes card state:

| Priority | Criterion | Source |
|----------|-----------|--------|
| 1 | **Pinned** (starred) | `localStorage['pinned_clients']` or `pinned_businesses` |
| 2 | **Online** (`data-online="true"`) | `data-online` attribute (set by presence WS type 5) |
| 3 | **Unread count** (higher first) | `data-unread` attribute (set by unread WS type 8) |
| 4 | **Last message time** (most recent first) | `data-last-message-at` attribute (RFC3339) |

### Sort data attributes on cards

**Business sidebar** (`.wa-chat-item`): `data-client-id`, `data-conversation-id`, `data-client-name`, `data-last-message-at`, `data-unread`, `data-online`, plus `onclick="togglePinClient(...)"` star button.

**Client sidebar** (`.business-item`): `data-business-id`, `data-conversation-id`, `data-business-name`, `data-business-type`, `data-last-message-at`, `data-unread`, `data-online`, plus `onclick="togglePinBusiness(...)"` star button.

### Sort triggers (deferred 200ms)

| Event | WS Type | Sidebar |
|-------|---------|---------|
| New message (preview/time update) | 1 | Both (`deferredSort()` / `deferredClientSort()`) |
| Presence update (online dot) | 5 | Both |
| Unread count update | 8 | Both |
| Conversation added/removed | 13 | Both |
| Pin toggle | User click | Both |
| Page load | — | Both (`sortClientList()` / `sortBusinessList()`) |

### Functions

| Function | File | Purpose |
|----------|------|---------|
| `togglePinClient(id)` | `business.js` | Toggle pin in localStorage + re-sort |
| `sortClientList()` | `business.js` | Priority sort for business sidebar |
| `deferredSort()` | `business.js` | Debounced (200ms) wrapper for `sortClientList` |
| `sortBusinessList()` | `client_chat.js` | Priority sort for client sidebar |
| `deferredClientSort()` | `client_chat.js` | Debounced (200ms) wrapper for `sortBusinessList` |
| `togglePinBusiness(id)` | `client.html` (inline) | Toggle pin in localStorage + re-sort |

### localStorage keys

| Key | Sidebar | Values |
|-----|---------|--------|
| `pinned_clients` | Business (client list) | `[clientID, ...]` |
| `pinned_businesses` | Client (business list) | `[businessID, ...]` |

## Per-Client Unread Badge (/business page)

`GetBizHome` handler in `business.go`:
- Unread count = unread messages + non-completed orders (`NOT IN ('fulfilled','completed','cancelled')`) + non-completed bookings (`NOT IN ('completed','cancelled')`)
- Template: `{{template "unread-badge" .}}` in `client_list.html`
- Navbar: `{{.PendingOrderCount}}`, `{{.PendingBookingCount}}`, `{{.TotalPending}}`

## Payment Model (polymorphic)

```go
type Payment struct {
    OrderID   *uint     // nullable — belongs to Order or Booking
    BookingID *uint     // nullable
    Amount    float64
    Method    string    // cash, card, bank_transfer, mobile_money
    Status    string    // pending, completed, failed
    Reference string
}
```

Payments link to either `Order` or `Booking` via nullable foreign keys. Quick mark-as-paid creates a `"cash"` + `"completed"` payment. Walk-in counter creates with `Reference: "Walk-in counter payment"`. Clients can submit payment claims (pending status) which business confirms or rejects.

## Dashboard Sidebar Navigation

12 items, active page detection via `.ActivePage`:

| # | Label | Icon | Link | Active Key |
|---|-------|------|------|-----------|
| 1 | Dashboard Overview | `fa-chart-line` | `/business/dashboard` | `dashboard` |
| 2 | Products | `fa-box` | `/business/products` | `products` |
| 3 | Services | `fa-concierge-bell` | `/business/services` | `services` |
| 4 | Orders | `fa-shopping-cart` | `/business/orders` | `orders` |
| 5 | Bookings | `fa-calendar-check` | `/business/bookings` | `bookings` |
| 6 | Payments | `fa-credit-card` | `/business/payments` | `payments` |
| 7 | Analytics | `fa-chart-bar` | `/business/analytics` | `analytics` |
| 8 | Notifications | `fa-bell` | `/business/notifications` | `notifications` |
| 9 | Hours | `fa-clock` | `/business/hours` | `hours` |
| 10 | Share | `fa-share-alt` | `/business/share` | `share` |
| 11 | Subscription | `fa-crown` | `/business/subscription` | `subscription` |
| 12 | Customers | `fa-users` | `/business` | `customers` |

Items 2–5 have count badges (ProductCount/ServiceCount/PendingOrderCount/PendingBookingCount). Subscription badge is loaded via HTMX from `/business/subscription/badge-sidebar`.

## Business Hours & Availability

Implemented as a full-page form at `GET /business/hours` (`hours.html`). Backend in `business_hours.go`.

### DB Fields (on `Business` model)
| Field | Type | Default |
|-------|------|---------|
| `TimeZone` | `string` | `"UTC"` |
| `BufferTime` | `int` | `0` (minutes between bookings) |
| `MaxBookingsPerSlot` | `int` | `1` |
| `IsAcceptingBookings` | `bool` | `true` |
| `BusinessHours` | `string` (jsonb) | `"{}"` |
| `SpecialHours` | `string` (jsonb) | `"[]"` |

### Routes (`business_hours.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/hours` | `GetBusinessHours` | Hours management page |
| `PUT /business/hours` | `UpdateBusinessHours` | Save weekly hours + buffer/max/tz |
| `PUT /business/hours/special` | `UpdateSpecialHours` | Save special hours/closures |
| `POST /business/hours/toggle` | `ToggleAcceptingBookings` | Toggle accepting new bookings |

### Template Data
Passes parsed `BusinessHours` (map) and `SpecialHours` (slice) so `{{json .BusinessHours}}` produces a JS object literal, not a string. The `json` template function returns `template.JS` for this reason — Go's `html/template` would otherwise wrap a bare `string` in JS quotes and escape it.

### JS rendering
- `renderWeeklyHours()` — iterates `serverHours[day.key]` to build time-range inputs
- `renderSpecialHours()` — iterates `serverSpecial` to build special-date entries
- `collectWeeklyHours()` / `saveWeeklyHours()` — serializes DOM state, POSTs to `PUT /business/hours`
- `saveSpecialHours()` — same for special hours
- `toggleAccepting(checked)` — toggles via `POST /business/hours/toggle`

### Availability validation
In `CreateBooking` (`bookings.go`), `validateBookingSlot()` checks:
1. Business is accepting bookings
2. Day has defined hours
3. Booking time falls within an open range
4. No conflicting special closures outside-of-hours

## Dashboard Table Pages (Orders & Bookings)

### Search/Filter Bar
Both pages have a client-side search/filter bar with:
- **Text search** with "Search in" dropdown: All Fields, Order/Booking #, Customer Name, Email
- **Status filter** dropdown
- **Clear button** (appears when filters active)
- **Results counter**: "X of Y results"
- **No match message** when filtering hides all rows

JS filter functions (`filterOrders`/`filterBookings`) use `data-number`, `data-customer-name`, `data-customer-email` attributes on `<tr>`. The filter JS is at the bottom of each page file.

### Action Dropdown
Inline step buttons collapsed into a three-dot (`⋮`) dropdown menu. `toggleDropdown()` JS function with click-outside-to-close. Always shows all 4 steps (Pending, Confirmed, Paid, Completed) plus Cancel. Only the next logical step is clickable.

### Receipt Column
Added as the last column in both orders and bookings tables. Shows a receipt icon button (`<i class="fas fa-receipt">`) — visible **only** when status is completed/fulfilled. Button opens `window.open('/business/orders/{{.ID}}/receipt', '_blank')`. The receipt page is a standalone HTML template with `@media print` CSS, item table, totals, payment summary, and a Print/Save PDF button.

## Walk-in Counter Flow

### Walk-in Action Cards
Two cards in the dashboard overview (`dashboard_content.html`) between the stat grid and recent tables:
- **Counter Order** — warning/yellow themed, opens `showQuickOrderModal()`
- **Walk-in Booking** — secondary/purple themed, opens `showQuickBookingModal()`

### mark_completed Checkbox
Present in all Quick Order / Quick Booking modals (dashboard.html, orders.html, bookings.html). When checked, the backend:
- Sets status to `"fulfilled"` (orders) or `"completed"` (bookings)
- Sets `PaidAmount = TotalAmount`
- Creates a completed cash payment record with `Reference: "Walk-in counter payment"`

### Backend handlers
- `CreateOrder` (`orders.go:16`): accepts optional `customer_name`/`email`/`phone` (creates client if no `client_id`), plus `mark_completed` flag
- `CreateBooking` (`bookings.go:324`): accepts `mark_completed` flag

## 3-Step Quick Wizards

Both Quick Order and Quick Booking modals use a 3-step wizard with numbered circle indicators and connector lines:

### Order wizard (`ordStep1/2/3`, `ordGoToStep`, `ordNextStep`, `ordRenderSummary`)
| Step | Label | Content |
|------|-------|---------|
| 1 | Browse | Searchable product picker + quantity |
| 2 | Details | Customer name/email/phone, delivery address, notes |
| 3 | Confirm | Summary + `mark_completed` checkbox + Create Order |

### Booking wizard (`bkgStep1/2/3`, `bkgGoToStep`, `bkgNextStep`, `bkgRenderSummary`)
| Step | Label | Content |
|------|-------|---------|
| 1 | Choose | Searchable service picker |
| 2 | Schedule | Date/time + customer name/email/phone |
| 3 | Confirm | Summary + `mark_completed` checkbox + Create Booking |

Step indicator colors:
- Order steps use `--color-warning` (yellow)
- Booking steps use `--color-secondary` (purple)

### Searchable Picker (replaces `<select>`)

Step 1 in all wizards uses a search input + filtered results list instead of a native `<select>`:

```html
<div class="relative mt-1">
  <i class="fas fa-search absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] text-xs"></i>
  <input type="text" id="orderProductSearch" placeholder="Type to search..." class="input pl-10 text-sm">
</div>
<div id="orderProductResults" class="mt-2 max-h-40 overflow-y-auto border border-[var(--color-border)] rounded-lg p-1 space-y-0.5">
  <!-- Rendered via JS: renderOrderProducts(filter) -->
</div>
<div id="orderProductCount" class="mt-1 text-xs text-[var(--color-text-muted)]">X of Y products</div>
```

Products/services are loaded from `/business/products` or `/business/services` HTML endpoints, parsed into JS arrays (`orderProductsList`/`bookingServicesList`), and rendered with real-time filtering via `input` event listener.

### Product & Service Loading
All wizard modals use HTML parsing (not JSON) for consistency:
- Products: `fetch('/business/products')` → parse HTML → extract `[data-product-id]` cards
- Services: `fetch('/business/services')` → parse HTML → extract `[data-service-id]` cards

## Receipt Feature

### Routes & Handlers
- `GET /business/orders/:id/receipt` → `GetOrderReceipt` (`orders.go`)
- `GET /business/bookings/:id/receipt` → `GetBookingReceipt` (`bookings.go`)

Both handlers:
1. Preload `Client`, `Items.Product`/`Items.Service`, `Payments`, `Business`
2. Verify business ownership and completed/fulfilled status
3. Render a standalone print-optimized HTML template

### Templates
- `web/templates/pages/business/dashboard/receipt_order.html`
- `web/templates/pages/business/dashboard/receipt_booking.html`

Standalone `<html>` pages (no sidebar/nav) with `@media print` CSS. Show:
- Business logo + name + email
- Receipt # + date
- Customer info (name, email, phone)
- Item table (name, qty, unit price, total)
- Totals (subtotal, paid, balance)
- Payment summary (method, reference, date, status)
- "Print / Save PDF" and "Close" buttons (hidden in print)

## AI Assistant (salesmee Assist)

Floating AI assistant chatbot available on business pages and client portal. Powered by Groq API (Llama 3.1 8B).

### Gating
- Only shown when `GROQ_API_KEY` env var is set (`assist.IsEnabled()`)
- Template data passes `AssistEnabled bool` — business side: `DashboardData.AssistEnabled`, client side: `gin.H["AssistEnabled"]`
- `assist_panel.html` wraps content in `{{if .AssistEnabled}}`

### Backend (`internal/services/assist/assist.go`)
| Function | Description |
|----------|-------------|
| `IsEnabled() bool` | Returns true if `GROQ_API_KEY` is set |
| `ChatCompletion(systemPrompt, messages)` | Calls Groq API with Llama 3.1 8B, returns reply |
| `BuildSystemPrompt(bizName, products, services, conversations)` | Builds business-context system prompt |
| `BuildClientSystemPrompt(businessCount)` | Builds client-context system prompt |

### Business Endpoints (`internal/handlers/business/assist.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /business/assist/chat` | `AssistChat` | Accepts `{ message, history }`, builds system prompt with business stats, returns `{ reply }` |
| `GET /business/assist/suggestions` | `GetAssistSuggestions` | Returns 4 suggestion chips: Draft a reply, Suggest a product, Help with SalesMee, Customer service tip |

### Client Endpoints (`internal/handlers/client/client_assist.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /client/assist/chat` | `ClientAssistChat` | Same as business but with client-focused system prompt |
| `GET /client/assist/suggestions` | `ClientGetAssistSuggestions` | Returns 4 suggestion chips: Draft a message, How to place an order, Help with SalesMee, Booking tips |

### JS (`web/static/js/modules/assist.js`)
- `toggleAssist()` / `closeAssist()` — open/close panel
- `sendMessage(text)` — POSTs to `API_BASE + '/chat'`, renders reply
- `loadSuggestions()` — GETs `API_BASE + '/suggestions'`, renders chips
- `assistQuickAction(id)` — triggers suggestion chip prompt
- API base configured via `window.ASSIST_API_BASE` (defaults to `/business/assist`, set to `/client/assist` on client pages)
- Conversation history persisted in `localStorage` (max 10 turns)

### CSS
All assist styles in `@layer components` in `styles.css`:
- `.assist-panel` — fixed position, glass background, flex layout
- `.assist-msg-user` / `.assist-msg-bot` — message bubbles
- `.assist-chip` — suggestion button
- `.assist-avatar` — wand icon in bot messages
- `.assist-btn` — floating button with glow + sparkle animations

## Client Portal

Full client-facing dashboard at `/client/`. All client handlers in `internal/handlers/client/` (5 files).

### Auth
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /client/login` | `ShowClientLogin` | Login page (redirects if already authed) |
| `POST /client/send-otp` | `SendClientOTP` | Send OTP to email, create client if new |
| `POST /client/verify-otp` | `VerifyClientOTP` | Verify OTP, set `client_token` cookie |
| `GET /client/logout` | `ClientLogout` | Clear session, mark offline |
| `GET /client/auth/google` | `InitiateClientGoogleAuth` | Google OAuth login |
| `GET /client/auth/facebook` | `InitiateClientFacebookAuth` | Facebook OAuth login |

### Dashboard & Chat
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /client/` | `ClientDashboard` | Main client page with business list + unread badges |
| `GET /client/discover` | `ShowDiscover` | Discover new businesses to connect with |
| `POST /client/connect/:business_id` | `ConnectToBusiness` | Create conversation with a business |
| `GET /client/businesses/:id/messages` | `GetClientMessages` | Chat messages with a business |
| `POST /client/businesses/:id/messages` | `CreateClientMessage` | Send a message |
| `PUT /client/businesses/:id/read` | `MarkClientConversationAsRead` | Mark conversation read |

### Client Order/Booking Actions
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /client/orders` | `ClientCreateOrder` | Place an order |
| `POST /client/orders/:id/confirm` | `ClientConfirmOrder` | Confirm pending order |
| `POST /client/orders/:id/cancel` | `ClientCancelOrder` | Cancel own order |
| `POST /client/orders/:id/update` | `ClientUpdateOrder` | Update order items |
| `POST /client/orders/:id/payment` | `ClientSubmitOrderPayment` | Submit payment claim |
| `POST /client/bookings` | `ClientCreateBooking` | Book a service |
| `POST /client/bookings/:id/confirm` | `ClientConfirmBooking` | Confirm pending booking |
| `POST /client/bookings/:id/cancel` | `ClientCancelBooking` | Cancel own booking |
| `POST /client/bookings/:id/update` | `ClientUpdateBooking` | Update booking details |
| `POST /client/bookings/:id/payment` | `ClientSubmitBookingPayment` | Submit payment claim |
| `POST /client/reviews` | `SubmitReview` | Leave a review |

### Presence
- `POST /client/heartbeat` — 30s interval, updates `is_online` + `last_seen_at`
- `POST /client/disconnect/:business_id` — remove connection

### Auth Middleware (`ClientMiddleware`)
- Validates JWT from `client_token` cookie or `Authorization` header
- Sets `client_id`, `client_email` in context
- Redirects to `/client/login` on failure

## Team Management

Full team member system with role-based permissions, invite flow, and location assignment.

### Model (`TeamMember`)
| Field | Type | Description |
|-------|------|-------------|
| `Name` | string | Member name |
| `Email` | string | Login email |
| `Password` | string | BCrypt hashed |
| `Role` | string | `manager`, `staff` |
| `Phone` | string | Contact phone |
| `Photo` | string | Profile photo URL |
| `Permissions` | string (JSON) | `{"orders:rw": true, ...}` |
| `InviteToken` | string | Token for accept-invite flow |
| `IsActive` | bool | Soft disable |
| `Locations` | []Location | M2M via `team_member_locations` |

### Routes (owner-only)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/team` | `GetTeam` | Team management page |
| `POST /business/team` | `InviteTeamMember` | Create + send invite |
| `PUT /business/team/:id` | `UpdateTeamMember` | Update details/permissions |
| `DELETE /business/team/:id` | `DeleteTeamMember` | Remove member |
| `GET /business/team/login` | `ShowTeamLogin` | Team login page |
| `POST /business/team/login` | `TeamLogin` | Authenticate team member |
| `GET /business/team/logout` | `TeamLogout` | Clear `team_token` |
| `GET /business/team/accept` | `ShowAcceptInvite` | Invite acceptance page |
| `POST /business/team/accept` | `AcceptInvite` | Set password, activate |

### Commission Tracking
`OrderItem` and `BookingItem` have `StaffID`, `CommissionType` (percentage/fixed), `CommissionValue`, `CommissionEarned` fields for tracking staff commissions.

## Multi-Location Management

Businesses can manage multiple physical locations with per-location filtering.

### Model (`Location`)
| Field | Type | Description |
|-------|------|-------------|
| `Name` | string | Location name |
| `Address` | string | Full address |
| `Phone` | string | Contact phone |
| `Email` | string | Contact email |
| `TimeZone` | string | IANA timezone |
| `Lat` / `Lng` | float64 | Map coordinates |
| `IsActive` | bool | Soft toggle |
| `SortOrder` | int | Display order |

### Routes (owner-only)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/locations` | `GetLocations` | Locations management page |
| `POST /business/locations` | `CreateLocation` | Create location |
| `PUT /business/locations/:id` | `UpdateLocation` | Update location |
| `DELETE /business/locations/:id` | `DeleteLocation` | Delete location |

### Per-Location Scoping
- Products, services, orders, and bookings have `LocationID` foreign key
- Dashboard overview supports `?location_id=` query param filtering
- Analytics can be filtered by location

## Reports & CSV Exports

Reporting dashboard with date-range filtering and CSV downloads.

### Routes (require `PermReportsView`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/reports` | `GetReportsPage` | Reports dashboard |
| `GET /business/reports/revenue` | `GetRevenueReport` | Daily revenue (orders + bookings) |
| `GET /business/reports/sales` | `GetSalesReport` | Product/service sales breakdown |
| `GET /business/reports/clients` | `GetClientReport` | New clients per day |
| `GET /business/reports/tax` | `GetTaxReport` | Monthly revenue for tax |
| `GET /business/reports/export/orders.csv` | `ExportOrdersCSV` | Download all orders as CSV |
| `GET /business/reports/export/bookings.csv` | `ExportBookingsCSV` | Download all bookings as CSV |
| `GET /business/reports/export/payments.csv` | `ExportPaymentsCSV` | Download all payments as CSV |
| `GET /business/reports/export/clients.csv` | `ExportClientsCSV` | Download all clients as CSV |

### HTMX Tabs
Tab navigation (`#reportTabs`) with `htmx-get` swapping `#reportContent`. Four tabs:
- Revenue — daily revenue chart data
- Sales — product/service ranking by revenue
- Clients — new client growth
- Tax — monthly aggregation for tax filing

## Notification System

Two types: in-app notifications (bell icon) and email notifications.

### Models
| Model | Description |
|-------|-------------|
| `BusinessNotifPrefs` | Per-business notification toggles (booking reminders, order/booking status, payment due, abandoned cart, re-engagement, sound) |
| `InAppNotification` | `Title`, `Body`, `Icon`, `Link`, `IsRead`, linked to `BusinessID` |
| `NotificationLog` | Email send log with `Type`, `Status`, `SentAt` |

### Routes
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/notifications` | `GetNotifications` | HTMX fragment — notification list |
| `GET /business/notifications/count` | `GetNotificationCount` | JSON `{"count": N}` |
| `POST /business/notifications/:id/read` | `MarkNotificationRead` | Mark single as read |
| `POST /business/notifications/read-all` | `MarkAllNotificationsRead` | Mark all as read |
| `GET /business/notification-settings` | `GetNotificationSettings` | Settings page |
| `PUT /business/notification-settings` | `UpdateNotificationSettings` | Save preferences |

### Scheduler (`internal/services/notifier/scheduler.go`)
Background goroutine started in `main.go` that runs notification checks on intervals (booking reminders, etc.).

## Conversation Insights & Progress

Tracks each conversation through a 7-stage sales funnel with scoring.

### Models
| Model | Fields | Description |
|-------|--------|-------------|
| `CustomerInsight` | `Tier`, `TierScore`, `ActivityScore`, `EngagementScore`, `TotalSpent`, `TotalOrders`, `TotalBookings`, `LastActiveAt` | Per-customer analytics |
| `ConversationProgress` | `CurrentStage`, `StageHistory` (JSON `[]StageTransition`), `ProgressScore`, `NextAction`, `ExpectedClose`, `ActualClose`, `Value` | Stage tracking |

### Conversation Stages
`initial` → `qualification` → `negotiation` → `confirmation` → `in_progress` → `completed` → `follow_up`

### Routes
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/conversations/:id/progress` | `GetConversationProgress` | Progress panel |
| `PUT /business/conversations/:id/stage` | `UpdateConversationStage` | Advance stage |
| `GET /business/conversations/:id/insights-badge` | `GetConversationInsightsBadge` | Tier badge |
| `GET /business/conversations/:id/insights-panel` | `GetConversationInsightsPanel` | Full insights panel |
| `POST /business/conversations/:id/insights/refresh` | `RefreshConversationInsights` | Recalculate scores |

## Social Auth

Both business and client support Google and Facebook OAuth.

### Business Social Auth
| Route | Handler |
|-------|---------|
| `GET /business/auth/google` | `InitiateBusinessGoogleAuth` |
| `GET /business/auth/google/callback` | `HandleBusinessGoogleCallback` |
| `GET /business/auth/facebook` | `InitiateBusinessFacebookAuth` |
| `GET /business/auth/facebook/callback` | `HandleBusinessFacebookCallback` |
| `GET /business/register/google` | `ShowRegisterGoogle` |
| `POST /business/register/google/complete` | `CompleteRegisterGoogle` |

### Client Social Auth
| Route | Handler |
|-------|---------|
| `GET /client/auth/google` | `InitiateClientGoogleAuth` |
| `GET /client/auth/google/callback` | `HandleClientGoogleCallback` |
| `GET /client/auth/facebook` | `InitiateClientFacebookAuth` |
| `GET /client/auth/facebook/callback` | `HandleClientFacebookCallback` |

## Subscription & Billing

### Plans
3 plans seeded in `main.go`:
| Plan | Price | Key Feature |
|------|-------|-------------|
| Silver | Free | Limited clients/products/services |
| Gold | $8/mo | Increased limits, analytics |
| Diamond | $15/mo | Unlimited, all features |

### Payment Providers
- **Stripe**: checkout sessions, subscriptions, billing portal, webhooks
- **Paddle**: checkout, subscriptions, transaction webhooks

Provider selection logic in `internal/services/payment/`: `provider.go` defines interface, `stripe.go` and `paddle.go` implement, `factory.go` creates provider based on env/config.

### Routes
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/subscription` | `GetSubscriptionPage` | Current plan + usage |
| `GET /business/subscription/plans` | `GetPlansPage` | All plans comparison |
| `GET /business/subscription/checkout` | `GetCheckoutPage` | Checkout with Paddle client token |
| `POST /business/subscription/checkout` | `CreateCheckout` | Create Stripe session or Paddle checkout |
| `POST /business/subscription/change` | `ChangePlan` | Switch plans |
| `POST /business/subscription/cancel` | `CancelSubscription` | Cancel at provider level |
| `GET /business/subscription/portal` | `BillingPortal` | Stripe/Paddle billing portal |
| `GET /business/subscription/badge` | `GetPlanBadge` | Navbar badge |
| `GET /business/subscription/badge-sidebar` | `GetPlanBadgeSidebar` | Sidebar badge |

### Webhooks
- `POST /stripe/webhook` — handles `checkout.completed`, `subscription.updated/deleted`, `invoice.paid/failed`
- `POST /paddle/webhook` — handles `subscription.created/updated/cancelled/past_due`, `transaction.completed/failed`

Both update the `BusinessSubscription` model and send notifications on failures.

## Chat & Conversation

### Chat Components (`web/templates/components/chat/`)
| Template | Description |
|----------|-------------|
| `text_message.html` | Business text message bubble |
| `media_message.html` | Business media (image/file) message |
| `order_card.html` | Business-side inline order card |
| `booking_card.html` | Business-side inline booking card |
| `client_text_message.html` | Client text message bubble |
| `client_media_message.html` | Client media message |
| `client_order_card.html` | Client-side order card |
| `client_booking_card.html` | Client-side booking card |
| `message_input.html` | Business chat input with media upload |
| `client_message_input.html` | Client chat input |
| `chat_header.html` | Business chat header with client info |
| `client_chat_header.html` | Client chat header with business info |
| `progress_controls.html` | Conversation stage progress |
| `insights_panel.html` | Customer insights panel |
| `insights_badge.html` | Tier badge |
| `empty_state.html` | Empty chat state |
| `sms.html` | Business SMS-style message |
| `booking_sms.html` | Booking SMS notification |

### JS Modules
| File | Description |
|------|-------------|
| `business_chat.js` | Business chat: send messages, media upload, inline actions (quick order/booking/payment/goal), order/booking lifecycle actions |
| `client_chat.js` | Client chat: send messages, view order/booking cards, confirm/cancel actions |
| `shared.js` | Toast notifications (`showToast()`), confirm/prompt modals, cookie helpers (`getCookie/setCookie`), CSRF setup, modal helpers |
| `theme.js` | Theme toggle (light/dark), localStorage persistence, system preference detection |
| `onboarding.js` | Onboarding panel lifecycle |
| `product_picker.js` | Business quick-order product picker with HTML parsing |
| `service_picker.js` | Business quick-booking service picker with HTML parsing |
| `client.js` | Client dashboard: sidebar toggles, tab switching, heartbeat, time-ago updates, discover navigation |
| `business.js` | Business chat page: sidebar toggles, client search, notification polling, conversation management |
| `app.js` | Core app setup: HTMX CSRF headers, click-outside dropdowns, image fallbacks |

## File Layout

```
internal/
  handlers/
    business/              — Business dashboard handlers (20 source files + 3 test)
      business.go           — BusinessHandler struct, GetBizHome, GetDashboard, profile CRUD
      orders.go             — Order CRUD, lifecycle (Send, Confirm, Reject, Fulfill, MarkPaid, Receipt)
      bookings.go           — Booking CRUD, lifecycle (MarkPaid, Receipt, Update, validateBookingSlot)
      products.go           — Product CRUD, images, client-facing product pages
      services.go           — Service CRUD
      payments.go           — Payments ledger, confirm/reject, payment methods CRUD, client payment claims
      analytics.go          — Analytics dashboard with date-range filtering
      subscription.go       — Subscription/billing, Stripe & Paddle webhooks
      public.go             — Public profile, client OTP connect
      business_widgets.go   — Quick booking/order/goal widgets from chat
      onboarding.go         — Onboarding progress, advance, skip
      business_hours.go     — Business hours & availability management
      profile_change_store.go — In-memory OTP-verified profile changes with 15-min TTL
      reports.go            — Reporting dashboard + CSV exports (orders, bookings, payments, clients)
      locations.go          — Multi-location CRUD
      reviews.go            — Reviews & ratings management
      team.go               — Team member CRUD, invite flow, location assignment
      team_auth.go          — Team login/logout
      assist.go             — AI assistant chat + suggestions
      notification_settings.go — Notification preferences
    client/                — Client handlers (5 files)
      client_auth.go        — Client auth (OTP, JWT), dashboard, chat, order/booking actions, middleware
      client_assist.go      — Client AI assistant chat + suggestions
      client.go             — Client discover, search, connect, business-side client CRUD
      reviews.go            — Client review submission
      social_auth.go        — Google & Facebook OAuth for clients
    admin/                 — Admin panel
    guide.go               — Public /guide page handler
    seo.go                 — Sitemap, robots.txt
    routes/
      routes.go             — Top-level routes (/, /guide, /b/:slug, legal pages, etc.)
      business_routes.go    — All /business/* route registrations (public + protected)
      client_routes.go      — /client/* route registrations
      admin_routes.go       — Admin routes
    models/
      business.go           — Business model
      client.go             — Client model
      order.go              — Order, Booking, OrderItem, BookingItem, Payment models
      product.go            — Product, ProductImage, InventoryLog models
      service.go            — Service model
      conversation.go       — Conversation model
      message.go            — Message model
      action.go             — Action model + ActionType constants
      subscription.go       — SubscriptionPlan, BusinessSubscription models
      payment_method.go     — PaymentMethod model
      client_auth.go        — ClientAuth model
      customer_insight.go   — CustomerInsight model (tier, scoring)
      conversation_progress.go — ConversationProgress, StageTransition models
      team_member.go        — TeamMember model
      location.go           — Location model
      review.go             — Review model
      notification.go       — BusinessNotifPrefs, NotificationLog, InAppNotification
      admin.go              — Admin model
      password_reset.go     — PasswordResetToken model
    middleware/
      auth.go               — BizzMiddleware (JWT for owners + team)
      csrf.go               — CSRF protection + TemplateData helper
      roles.go              — Permission constants + RequirePermission/RequireOwner
      ratelimit.go          — Token bucket rate limiter (5/s auth, 60/s API)
      subscription.go       — RequireFeature + CheckResourceLimit
    services/
      auth.go               — JWT generation/validation
      customer_auth.go      — Client token generation
      email.go              — Email sending (OTP, notifications)
      social.go             — Google/Facebook OAuth helpers
      assist/
        assist.go           — Groq API client, system prompt builders
      notifier/
        scheduler.go         — Background notification scheduling
        log.go               — Notification logging
      onboarding/
        detector.go          — Onboarding step detection
      payment/
        provider.go          — Payment provider interface
        stripe.go            — Stripe integration (checkout, subscriptions, portal, webhooks)
        paddle.go            — Paddle integration (checkout, subscriptions, portal, webhooks)
        factory.go           — Provider factory
      subscription/
        strategy.go          — Plan change transition strategies
        limits.go            — Resource limit checking
    data/
      currencies.go         — 171 currencies with symbols
      countries.go          — 200 countries
      businesstypes.go      — 11 business types with icons/colors
    db/
      db.go                 — Database connection (PostgreSQL/SQLite)
web/
  templates/
    layouts/
      base.html             — Base layout (doctype, meta, CSS, JS, blocks)
    pages/
      business/
        business.html        — Business home (client list + chat)
        dashboard.html       — Dashboard layout (sidebar + modal wizards)
        business_share.html  — Share page with QR code
        subscription.html    — Subscription management
        subscription_plans.html — Plans comparison
        checkout.html        — Checkout page
        notification_settings.html — Notification preferences
        login/register/...  — Auth pages
        team_login.html      — Team member login
        team_accept.html     — Accept invite
      business/dashboard/
        orders.html          — Orders table with search/filter, action dropdown, 3-step wizard
        bookings.html        — Bookings table with search/filter, action dropdown, 3-step wizard
        products.html        — Products grid CRUD
        services.html        — Services grid CRUD
        analytics.html       — Analytics dashboard
        payments.html        — Payments ledger
        hours.html           — Business hours & availability
        reports.html         — Reports dashboard with tabs
        reports_content.html — Report content partials
        reviews.html         — Reviews management
        locations.html       — Location management
        team.html            — Team management
        receipt_order.html   — Print receipt (standalone)
        receipt_booking.html — Print receipt (standalone)
        order_confirmation.html — Post-creation confirmation
      client/
        client.html          — Client dashboard (business list + chat)
        client_chat.html     — Chat messages partial
        client_login.html    — Login/register
        client_otp.html      — OTP verification
        client_connect.html  — Connect to business form
        client_connect_otp.html — Connect OTP
        client_discover.html — Discover businesses
        client_discover_content.html — Discover partial
        client_products.html — Products page
        client_verify.html   — Email verification
      admin/                 — Admin panel pages
      legal/                 — Privacy, terms, cookies, refund policy, user deletion
      guide.html            — Public guide page
    components/
      chat/                 — 20 templates (message bubbles, cards, input, progress, insights)
      ui/                   — Sidebar, bottom nav, dashboard sidebar, dashboard navbar, card, alert
      modals/               — Payment method, profile, product/service pickers, client order/booking, review, quick order/booking, new client
      cards/                — Client list with unread badges
      badges/               — Unread badge component, notifications list
      assist/               — AI assistant panel
      onboarding/           — Onboarding panel
      client/               — Business list sidebar
      dashboard_content.html — HTMX dashboard overview panel
  static/
    css/
      styles.css            — Source CSS (variables + Tailwind directives)
      dist/styles.css       — Built output (minified)
    js/
      modules/
        assist.js           — AI assistant panel logic
        business_chat.js    — Business chat with order/booking action functions
        client_chat.js      — Client chat with order/booking action functions
        shared.js           — Toast, confirm/prompt modals, cookie helpers
        theme.js            — Theme toggle (light/dark)
        onboarding.js       — Onboarding panel lifecycle
        product_picker.js   — Quick-order product picker
        service_picker.js   — Quick-booking service picker
        client.js           — Client dashboard (sidebar, heartbeat, time-ago)
        business.js         — Business chat page (sidebar, notifications, search)
      core/
        app.js              — HTMX CSRF, click-outside dropdowns, image fallbacks
        theme.js            — Theme system
      uploads/
        logos/              — Business logo uploads
        products/           — Product image uploads
cmd/
  server/
    main.go                — Entry point (config, migrations, seed, routes, server start)
```

## Key Go Handlers

### Business Handler (`internal/handlers/business/business.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/` | `GetBizHome` | Client list + unread counts + pending badges |
| `GET /business/dashboard` | `GetDashboard` | Dashboard overview (stats, recent items, low stock) |
| `GET /business/dashboard/stats` | `GetDashboardStats` | HTMX stats fragment |
| `GET /business/share` | `GetSharePage` | Share page with QR code |
| `PUT /business/profile` | `UpdateBusinessProfile` | Update business name, email, currency, etc. |
| `POST /business/logo` | `UploadBusinessLogo` | Upload business logo image |
| `POST /business/regenerate-slug` | `RegenerateSlug` | Generate new public profile slug |
| `POST /business/profile/initiate` | `InitiateProfileChange` | Start OTP-verified profile change |
| `POST /business/profile/confirm` | `ConfirmProfileChange` | Confirm profile change with OTP |
| `POST /business/profile/resend-otp` | `ResendProfileOTP` | Resend OTP code |
| `GET /business/notifications` | `GetNotifications` | HTMX notification list |
| `GET /business/notifications/count` | `GetNotificationCount` | Unread count JSON |
| `POST /business/notifications/:id/read` | `MarkNotificationRead` | Mark notification read |
| `POST /business/notifications/read-all` | `MarkAllNotificationsRead` | Mark all read |

### Orders (`internal/handlers/business/orders.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/orders` | `GetOrders` | Orders management page |
| `GET /business/orders/stats` | `GetOrdersStats` | HTMX orders fragment |
| `GET /business/orders/stats-grid` | `GetOrdersStatsGrid` | HTMX stats grid |
| `POST /business/orders` | `CreateOrder` | Create order (supports walk-in completed) |
| `PUT /business/orders/:id` | `UpdateOrder` | Update order details |
| `PUT /business/orders/:id/status` | `UpdateOrderStatus` | Update order status |
| `POST /business/orders/:id/send` | `SendOrderToClient` | Send draft → pending |
| `POST /business/orders/:id/confirm` | `ConfirmOrderBusiness` | Business confirms → confirmed |
| `POST /business/orders/:id/reject` | `RejectOrder` | Reject → cancelled |
| `POST /business/orders/:id/fulfill` | `FulfillOrder` | Fulfill → fulfilled |
| `PUT /business/orders/:id/paid` | `MarkOrderAsPaid` | Quick mark as fully paid |
| `GET /business/orders/:id/receipt` | `GetOrderReceipt` | Print receipt page |
| `GET /business/conversations/:id/products` | `GetConversationProducts` | Products for chat picker |
| `POST /business/conversations/:id/order-draft` | `CreateOrderDraft` | Draft order from chat |

### Bookings (`internal/handlers/business/bookings.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/bookings` | `GetBookings` | Bookings management page |
| `GET /business/bookings/stats` | `GetBookingsStats` | HTMX bookings fragment |
| `GET /business/bookings/stats-grid` | `GetBookingsStatsGrid` | HTMX stats grid |
| `GET /business/bookings/:id` | `GetBooking` | Get single booking |
| `POST /business/bookings` | `CreateBooking` | Create booking (supports walk-in completed) |
| `PUT /business/bookings/:id` | `UpdateBooking` | Update booking details |
| `PUT /business/bookings/:id/status` | `UpdateBookingStatus` | Update booking status |
| `PUT /business/bookings/:id/paid` | `MarkBookingAsPaid` | Quick mark as fully paid |
| `GET /business/bookings/:id/receipt` | `GetBookingReceipt` | Print receipt page |
| `GET /business/conversations/:id/services` | `GetConversationServices` | Services for chat picker |

### Products (`internal/handlers/business/products.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/products` | `GetProducts` | Products management page |
| `GET /business/products/:id` | `GetProduct` | Get single product |
| `POST /business/products` | `CreateProduct` | Create product |
| `PUT /business/products/:id` | `UpdateProduct` | Update product |
| `DELETE /business/products/:id` | `DeleteProduct` | Delete product |
| `POST /business/products/:id/image` | `UploadProductImage` | Upload product image |
| `GET /business/products/:id/images` | `GetProductImages` | List product images |
| `DELETE /business/products/:id/images/:image_id` | `DeleteProductImage` | Delete product image |

### Services (`internal/handlers/business/services.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/services` | `GetServices` | Services management page |
| `GET /business/services/:id` | `GetService` | Get single service |
| `POST /business/services` | `CreateService` | Create service |
| `PUT /business/services/:id` | `UpdateService` | Update service |
| `DELETE /business/services/:id` | `DeleteService` | Delete service |

### Payments (`internal/handlers/business/payments.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/payments` | `GetPayments` | Payments ledger page |
| `GET /business/payments/stats` | `GetPaymentsStats` | HTMX payments fragment |
| `GET /business/payments/stats-grid` | `GetPaymentsStatsGrid` | HTMX stats grid |
| `POST /business/payment-instructions` | `UpdatePaymentInstructions` | Set payment instructions text |
| `POST /business/orders/:id/payments/confirm-all` | `ConfirmAllOrderPayments` | Confirm all pending order payments |
| `POST /business/orders/:id/payments/:payment_id/confirm` | `ConfirmOrderPayment` | Confirm single order payment |
| `POST /business/orders/:id/payments/:payment_id/reject` | `RejectOrderPayment` | Reject single order payment |
| `POST /business/bookings/:id/payments/confirm-all` | `ConfirmAllBookingPayments` | Confirm all pending booking payments |
| `POST /business/bookings/:id/payments/:payment_id/confirm` | `ConfirmBookingPayment` | Confirm single booking payment |
| `POST /business/bookings/:id/payments/:payment_id/reject` | `RejectBookingPayment` | Reject single booking payment |
| `GET /business/orders/:id/payments` | `GetOrderPayments` | Get order payments |
| `GET /business/bookings/:id/payments` | `GetBookingPayments` | Get booking payments |
| `GET /business/payment-methods` | `GetPaymentMethods` | List payment methods |
| `POST /business/payment-methods` | `CreatePaymentMethod` | Create payment method |
| `PUT /business/payment-methods/:id` | `UpdatePaymentMethod` | Update payment method |
| `DELETE /business/payment-methods/:id` | `DeletePaymentMethod` | Delete payment method |

### Analytics (`internal/handlers/business/analytics.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/analytics` | `GetAnalytics` | Analytics dashboard |
| `GET /business/analytics/stats` | `GetAnalyticsStats` | HTMX analytics fragment |
| `GET /business/analytics/stats-grid` | `GetAnalyticsStatsGrid` | HTMX stats grid |

### Subscription (`internal/handlers/business/subscription.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/subscription` | `GetSubscriptionPage` | Subscription management |
| `GET /business/subscription/plans` | `GetPlansPage` | Plans listing |
| `GET /business/subscription/checkout` | `GetCheckoutPage` | Checkout page |
| `POST /business/subscription/checkout` | `CreateCheckout` | Create checkout session |
| `POST /business/subscription/change` | `ChangePlan` | Change subscription plan |
| `POST /business/subscription/cancel` | `CancelSubscription` | Cancel subscription |
| `GET /business/subscription/portal` | `BillingPortal` | Stripe/Paddle billing portal |
| `GET /business/subscription/badge` | `GetPlanBadge` | Plan badge for navbar |
| `GET /business/subscription/badge-sidebar` | `GetPlanBadgeSidebar` | Plan badge for sidebar |
| `POST /stripe/webhook` | `StripeWebhook` | Stripe webhook handler |
| `POST /paddle/webhook` | `PaddleWebhook` | Paddle webhook handler |

### Business Hours (`internal/handlers/business/business_hours.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/hours` | `GetBusinessHours` | Hours management page |
| `PUT /business/hours` | `UpdateBusinessHours` | Save weekly hours + buffer/max/tz |
| `PUT /business/hours/special` | `UpdateSpecialHours` | Save special hours/closures |
| `POST /business/hours/toggle` | `ToggleAcceptingBookings` | Toggle accepting new bookings |

### Onboarding (`internal/handlers/business/onboarding.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/onboarding/status` | `GetOnboardingStatus` | Returns current step + total_steps |
| `POST /business/onboarding/advance` | `AdvanceOnboarding` | Manual advance (e.g. "Let's Go!") |
| `POST /business/onboarding/progress` | `CheckOnboardingProgress` | Re-runs detection, advances if condition met |
| `POST /business/onboarding/skip` | `SkipOnboarding` | Marks as completed |

### Client Widgets (`internal/handlers/business/business_widgets.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /business/clients/:id/quick-booking` | `QuickBooking` | Quick booking from chat |
| `POST /business/clients/:id/quick-order` | `QuickOrder` | Quick order from chat |
| `POST /business/clients/:id/request-payment` | `RequestPayment` | Request payment from client |
| `POST /business/clients/:id/set-goal` | `SetGoal` | Set conversation goal |

### AI Assistant (`internal/handlers/business/assist.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /business/assist/chat` | `AssistChat` | AI chat for business owners |
| `GET /business/assist/suggestions` | `GetAssistSuggestions` | Suggestion chips |

### Reports (`internal/handlers/business/reports.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/reports` | `GetReportsPage` | Reports dashboard |
| `GET /business/reports/revenue` | `GetRevenueReport` | Revenue report |
| `GET /business/reports/sales` | `GetSalesReport` | Sales breakdown |
| `GET /business/reports/clients` | `GetClientReport` | Client growth report |
| `GET /business/reports/tax` | `GetTaxReport` | Tax report |
| `GET /business/reports/export/orders.csv` | `ExportOrdersCSV` | CSV export |
| `GET /business/reports/export/bookings.csv` | `ExportBookingsCSV` | CSV export |
| `GET /business/reports/export/payments.csv` | `ExportPaymentsCSV` | CSV export |
| `GET /business/reports/export/clients.csv` | `ExportClientsCSV` | CSV export |

### Locations (`internal/handlers/business/locations.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/locations` | `GetLocations` | Locations management |
| `POST /business/locations` | `CreateLocation` | Create location |
| `PUT /business/locations/:id` | `UpdateLocation` | Update location |
| `DELETE /business/locations/:id` | `DeleteLocation` | Delete location |

### Team (`internal/handlers/business/team.go` + `team_auth.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/team` | `GetTeam` | Team management page |
| `POST /business/team` | `InviteTeamMember` | Create + invite member |
| `PUT /business/team/:id` | `UpdateTeamMember` | Update member |
| `DELETE /business/team/:id` | `DeleteTeamMember` | Delete member |
| `GET /business/team/login` | `ShowTeamLogin` | Team login page |
| `POST /business/team/login` | `TeamLogin` | Authenticate team member |
| `GET /business/team/logout` | `TeamLogout` | Team logout |
| `GET /business/team/accept` | `ShowAcceptInvite` | Accept invite page |
| `POST /business/team/accept` | `AcceptInvite` | Set password |

### Reviews (`internal/handlers/business/reviews.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/reviews` | `GetReviews` | Reviews management |
| `POST /business/reviews/:id/reply` | `ReplyToReview` | Reply to review |

### Notification Settings (`internal/handlers/business/notification_settings.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/notification-settings` | `GetNotificationSettings` | Settings page |
| `PUT /business/notification-settings` | `UpdateNotificationSettings` | Save preferences |

### Public Business Routes (standalone, no BusinessHandler)
| Route | Handler | File |
|-------|---------|------|
| `GET /b/:slug` | `GetPublicProfile` | `public.go` |
| `GET /api/connect/:slug` | `ShowConnect` | `public.go` |
| `POST /api/connect/:slug` | `SendConnectOTP` | `public.go` |
| `POST /api/connect/:slug/verify` | `VerifyConnectOTP` | `public.go` |

### Public General Routes
| Route | Handler | File |
|-------|---------|------|
| `GET /` | `HomePage` | `handlers/home.go` |
| `GET /guide` | `ShowGuide` | `guide.go` |
| `GET /privacy` | `ShowPrivacy` | `handlers/legal.go` |
| `GET /terms` | `ShowTerms` | `handlers/legal.go` |
| `GET /cookies` | `ShowCookies` | `handlers/legal.go` |
| `GET /refund-policy` | `ShowRefund` | `handlers/legal.go` |
| `GET /user-deletion` | `ShowUserDeletion` | `handlers/legal.go` |
| `POST /user-deletion` | `SubmitUserDeletion` | `handlers/legal.go` |
| `GET /sitemap.xml` | `ServeSitemap` | `seo.go` |
| `GET /robots.txt` | `ServeRobots` | `seo.go` |

## WebSocket Protocol (Protobuf)

Event types defined in `internal/chatpb/chatpb.pb.go`:

| Type | Name | Direction | Purpose |
|------|------|-----------|---------|
| 1 | `NEW_MESSAGE` | Both | New chat message |
| 2 | `READ_RECEIPT` | Both | Message read ack |
| 3 | `TYPING_START` | Both | User started typing |
| 4 | `TYPING_STOP` | Both | User stopped typing |
| 5 | `PRESENCE_UPDATE` | Both | Online/offline (client ↔ biz via heartbeat) |
| 6 | `ORDER_UPDATE` | Biz → Client | Order status change |
| 7 | `BOOKING_UPDATE` | Biz → Client | Booking status change |
| 8 | `UNREAD_COUNT` | Both | Per-conversation unread badge update |
| 9 | `PING` | Server → Client | Keepalive |
| 10 | `PONG` | Client → Server | Keepalive response |
| 11 | `DELIVERED_ACK` | Client → Server | Client received message |
| 12 | `DELIVERED_RECEIPT` | Server → Sender | Server confirms delivery to recipient |
| 13 | `CONVERSATION_UPDATE` | Both | Sidebar card insert/update/remove (see `BroadcastConversationUpdate`) |
| 14 | `PENDING_COUNT` | Biz → Navbar | Order/booking pending count badges |

### jsonConversationUpdate (`internal/ws/client.go:364`)

Used for WS type 13 frames — sent to both `biz:<id>` and `client:<id>` rooms when a conversation is created or a card needs updating:

```go
type jsonConversationUpdate struct {
    ConversationID string `json:"conversation_id"`
    BizCardHTML    string `json:"biz_card_html"`
    ClientCardHTML string `json:"client_card_html"`
    ClientID       string `json:"client_id"`       // set from frame.SenderId
    Removed        bool   `json:"removed"`          // auto-detected when both HTML empty
}
```

`Removed=true` when both `BizCardHTML` and `ClientCardHTML` are empty — JS handlers delete the card instead of inserting/updating.

### Broadcast functions (`internal/ws/broadcast.go`)

| Function | Rooms | SenderId | Usage |
|----------|-------|----------|-------|
| `BroadcastConversationUpdate(hub, convID, bizHTML, clientHTML, bizID, clientID)` | `biz:<bizID>`, `client:<clientID>` | `clientID` | New/updated card for all 6+ creation paths |
| `BroadcastConversationRemovedToBiz(hub, convID, bizID, clientID)` | `biz:<bizID>` only | `clientID` | Business deletes client → card removed from biz sidebar only |
| `BroadcastConversationRemovedToClient(hub, convID, bizID, clientID)` | `client:<clientID>` only | `bizID` | Client disconnects → card removed from client sidebar only |
| `BroadcastPresenceUpdate(hub, clientID, isOnline, lastSeen, bizID)` | `biz:<bizID>` | — | Client online/offline, sets `data-online` attribute |
| `BroadcastBusinessPresenceUpdate(hub, bizID, isOnline, clientIDs)` | `client:<clientID>` per client | `bizID` | Business online/offline, sets `data-online` attribute |
| `BroadcastUnreadCount(hub, convID, count, roomID, roomPrefix)` | configurable | — | Updates `data-unread` + badge element |

### 6+2 conversation update paths

Every path that creates or destroys a conversation calls one of the broadcast functions:

**Creation (type 13 with card HTML):**
1. `CreateMessage` — first message in a conversation triggers card render + broadcast
2. `ConnectToBusiness` — client discovers + connects
3. `CreateClient` — business adds client manually
4. `ShowConnect` — public OTP connect
5. `getOrCreateConversation` — client auth auto-creates (used by `GetClientMessages`, `CreateClientMessage`, `ClientDashboard`)
6. `CreateOrder` / `CreateBooking` — creating an order/booking for a client creates conversation if needed

**Removal (type 13 with `removed: true`):**
1. `DeleteClient` — business deletes → `BroadcastConversationRemovedToBiz`
2. `DisconnectFromBusiness` — client disconnects → `BroadcastConversationRemovedToClient`

### JS globals & HTMX swap

`business_chat.js` and `chat_common.js` use `window.*` globals (not bare `clientId`/`conversationId`) because `loadClient()` was changed from `htmx.ajax()` to `fetch() + innerHTML + htmx.process()` to avoid HTMX 1.9.10's `insertBefore on null` bug:

```javascript
// business.js — loadClient()
window.clientId = clientId;
window.conversationId = el.getAttribute('data-conversation-id');
window.businessId = window.BUSINESS_ID;
window.sender = 'business';
```

HTMX View Transitions disabled via `<meta name="htmx-config" content='{"useViewTransition": false}'>` in `dashboard_head.html` and `base.html`.

### Script loading order (business.html)

`shared.js` → `ws.js` → `business.js` → `chat_common.js` → `business_chat.js`

Inline `<script>` blocks in `business_chat.html` are NOT executed when loaded via `fetch() + innerHTML` — all cross-file references use `window.*` globals.
