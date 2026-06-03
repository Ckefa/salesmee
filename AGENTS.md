# AGENTS.md — SalesMee

## Project Overview

SalesMee is a platform connecting businesses with their clients. Businesses can manage products, services, orders, bookings, and client conversations through a dashboard + chat interface.

## Tech Stack

- **Backend:** Go 1.x, Gin web framework, GORM (PostgreSQL)
- **Frontend:** Tailwind CSS v3, HTMX, vanilla JS (no framework)
- **Templating:** Go `html/template` with `{{define}}`/`{{template}}` blocks
- **CSS:** Custom CSS vars in `web/static/css/styles.css` → built to `web/static/css/dist/styles.css` via Tailwind

## Build & Run

```sh
# Build CSS (required after any Tailwind class change)
npm run css:build

# Run Go app
go run ./cmd/web

# Build Go binary
go build ./...
go vet ./...
```

## Unified 4-Step Color System (Orders & Bookings)

Every order/booking status maps to one of 4 colored steps. The same colors are used across all surfaces (chat cards, dashboard tables, stat icons, step buttons):

| Step | Status(es) | Color | CSS Var | Badge Classes |
|------|-----------|-------|---------|---------------|
| 1 — Pending | `pending` | Yellow | `--color-warning` | `bg-[var(--color-warning-light)] text-[var(--color-warning)]` |
| 2 — Confirmed | `client_confirmed`, `confirmed` | Blue | `--color-info` | `bg-[var(--color-info-light)] text-[var(--color-info)]` |
| 3 — Paid | n/a (paid_amount check) | Purple | `--color-secondary` | `bg-[var(--color-secondary-light)] text-[var(--color-secondary)]` |
| 4 — Completed | `fulfilled`, `completed` | Green | `--color-success` | `bg-[var(--color-success-light)] text-[var(--color-success)]` |
| Cancelled | `cancelled` | Red | `--color-error` | `bg-[var(--color-error-light)] text-[var(--color-error)]` |
| Draft (orders only) | `draft` | Neutral | `--color-surface-tertiary` | `bg-[var(--color-surface-tertiary)] text-[var(--color-text-secondary)]` |

**Active button state:** full background + white text (e.g. `bg-[var(--color-warning)] text-white`).
**Dimmed/past state:** light background + colored text + `opacity-40 cursor-default`.
**Always show all 4 step buttons** — only the next logical step is clickable.

### Order lifecycle
`draft` → `pending` (step 1) → `client_confirmed` / `confirmed` (step 2) → paid (step 3) → `fulfilled`/`completed` (step 4)

### Booking lifecycle
`pending` (step 1) → `client_confirmed` (step 2) → paid (step 3) → `completed` (step 4)

### Important: Display labels
- `fulfilled` and `completed` statuses both display as **"Completed"** in the UI
- Internal DB status remains `fulfilled` for orders — only display labels change
- Template `$isFulfilled` checks both `"fulfilled"` and `"completed"`:
  `{{$isFulfilled := or (eq $status "fulfilled") (eq $status "completed")}}`

## Status Badge Patterns

### Dashboard table badge (orders.html, bookings.html)
```html
<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full
  {{if eq .Status `pending`}}bg-[var(--color-warning-light)] text-[var(--color-warning)]
  {{else if eq .Status `client_confirmed`}}bg-[var(--color-info-light)] text-[var(--color-info)]
  ...">
  {{if eq .Status `pending`}}Pending{{else if eq .Status `client_confirmed`}}Confirmed{{end}}
</span>
```

### Dashboard overview badge (dashboard_content.html)
```html
<span class="badge-status px-2 py-0.5 rounded-md font-medium text-[10px] sm:text-xs
  {{if eq .Status `pending`}}bg-[var(--color-warning-light)] text-[var(--color-warning)]...">
  {{if eq .Status `pending`}}Pending{{end}}
</span>
```

### Step action buttons (table actions column)
```html
<button onclick="{{if $isPending}}approveBooking({{.ID}}){{else}};{{end}}"
  class="px-1.5 py-1 rounded text-xs font-medium transition flex items-center gap-1
  {{if $isPending}}bg-[var(--color-info)] text-white cursor-pointer hover:brightness-110
  {{else}}bg-[var(--color-info-light)] text-[var(--color-info)] opacity-40 cursor-default{{end}}">
  <i class="fas fa-check-circle text-[10px]"></i>Confirmed
</button>
```

### Chat card badge
```html
<span class="text-xs font-medium px-2 py-0.5 rounded-full
  {{if $isPending}}bg-[var(--color-warning-light)] text-[var(--color-warning)]
  {{else if $isConfirmed}}bg-[var(--color-info-light)] text-[var(--color-info)]...">
  {{if $isPending}}Pending{{else if $isConfirmed}}Confirmed{{end}}
</span>
```

### Chat card progress bar (4 steps)
Progress bar uses 4 segments with dynamic classes per segment:
- `"completed"` segment: `bg-[var(--color-success)] text-white` (always green check)
- `"active"` segment: color matches the step (yellow/blue/purple/green)
- `"pending"` segment: `bg-[var(--color-surface-tertiary)]` (gray)

## Per-Client Unread Badge (/business page)

`internal/handlers/business/business.go` — `GetBizHome` handler:
- Unread count = unread messages + non-completed orders (`NOT IN ('fulfilled', 'completed', 'cancelled')`) + non-completed bookings (`NOT IN ('completed', 'cancelled')`)
- Template: `{{template "unread-badge" .}}` in `client_list.html`
- Navbar: `{{.PendingOrderCount}}`, `{{.PendingBookingCount}}`, `{{.TotalPending}}`

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

Always use `var(--color-*)` via Tailwind arbitrary value syntax: `bg-[var(--color-warning-light)] text-[var(--color-warning)]`.

## File Layout

```
internal/
  handlers/business/  — Business handlers (orders.go, bookings.go, business.go, etc.)
  handlers/client/    — Client handlers (client_auth.go)
  models/             — GORM models
  routes/             — Route registrations
web/
  templates/
    pages/business/dashboard/  — Dashboard subpages (orders.html, bookings.html, analytics.html)
    pages/business/            — Main business layout (business.html, dashboard.html)
    components/chat/           — Chat cards (order_card.html, booking_card.html, etc.)
    components/                — Shared components (dashboard_content.html, badges/)
  static/
    css/
      styles.css               — Source CSS (variables + Tailwind directives)
      dist/styles.css           — Built output (minified)
    js/modules/
      client_chat.js           — Client chat rendering with dynamic order/booking status colors
      business_chat.js         — Business chat with order/booking action functions
```

## Key Go Handlers

| Route | Handler | File |
|-------|---------|------|
| `GET /business/` | `GetBizHome` | `internal/handlers/business/business.go` |
| `GET /business/dashboard` | `GetDashboard` | `internal/handlers/business/business.go` |
| `GET /business/orders` | `GetOrdersPage` | `internal/handlers/business/orders.go` |
| `GET /business/bookings` | `GetBookingsPage` | `internal/handlers/business/bookings.go` |
| `PUT /business/bookings/:id/paid` | `MarkBookingAsPaid` | `internal/handlers/business/bookings.go` |
| `PUT /business/orders/:id/paid` | `MarkOrderAsPaid` | `internal/handlers/business/orders.go` |

## Template Variable Naming

- `$isPending`, `$isConfirmed`, `$isCompleted`, `$isCancelled`, `$isFulfilled` — boolean status checks
- `$isFullyPaid := ge .PaidAmount .TotalAmount` — payment check gates step 3→4
- `$isDraft` — orders only
- `$isClientConfirmed` — orders only (distinct from business `$isConfirmed`)
