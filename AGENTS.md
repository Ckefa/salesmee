# AGENTS.md — SalesMee

## Project Overview

SalesMee connects businesses with their clients. Businesses manage products, services, orders, bookings, payments, and client conversations through a dashboard + chat interface.

## Tech Stack

- **Backend:** Go 1.x, Gin web framework, GORM (PostgreSQL)
- **Frontend:** Tailwind CSS v3, HTMX, vanilla JS (no framework)
- **Templating:** Go `html/template` with `{{define}}`/`{{template}}` blocks
- **CSS:** Custom CSS vars in `web/static/css/styles.css` → built to `web/static/css/dist/styles.css` via Tailwind

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
| `sub` | `func(a, b float64) float64` | Subtraction |
| `mul` | `func(a, b float64) float64` | Multiplication |
| `div` | `func(a, b float64) float64` | Division |
| `float` | `func(i int) float64` | Int to float64 |
| `title` | `func(string) string` | `strings.Title` (capitalizes each word) |
| `default` | `func(def, val interface{})` | Returns `def` if `val` is nil/empty |
| `hasPrefix` | `strings.HasPrefix` | String prefix check |
| `dict` | `func(...interface{}) map[string]interface{}` | Builds a map from key-value pairs |
| `json` | `func(interface{}) string` | JSON marshals to string |
| `formatDate` | `func(time.Time) string` | Formats as `"Jan 2, 2006"` |
| `formatTime` | `func(time.Time) string` | Formats as `"3:04 PM"` |
| `fbLogin` | `func() bool` | Returns true if FB_LOGIN env var is set |

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

Payments link to either `Order` or `Booking` via nullable foreign keys. Quick mark-as-paid creates a `"cash"` + `"completed"` payment. Walk-in counter creates with `Reference: "Walk-in counter payment"`.

## Dashboard Sidebar Navigation

10 items, active page detection via `.ActivePage`:

| # | Label | Icon | Link | Active Key |
|---|-------|------|------|-----------|
| 1 | Dashboard Overview | `fa-chart-line` | `/business/dashboard` | `dashboard` |
| 2 | Products | `fa-box` | `/business/products` | `products` |
| 3 | Services | `fa-concierge-bell` | `/business/services` | `services` |
| 4 | Orders | `fa-shopping-cart` | `/business/orders` | `orders` |
| 5 | Bookings | `fa-calendar-check` | `/business/bookings` | `bookings` |
| 6 | Payments | `fa-credit-card` | `/business/payments` | `payments` |
| 7 | Analytics | `fa-chart-bar` | `/business/analytics` | `analytics` |
| 8 | Share | `fa-share-alt` | `/business/share` | `share` |
| 9 | Subscription | `fa-crown` | `/business/subscription` | `subscription` |
| 10 | Customers | `fa-users` | `/business` | `customers` |

Items 2–5 have count badges (ProductCount/ServiceCount/PendingOrderCount/PendingBookingCount). Subscription badge is loaded via HTMX from `/business/subscription/badge-sidebar`.

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

## File Layout

```
internal/
  handlers/
    business/              — Business dashboard handlers (11 files)
      business.go           — BusinessHandler struct, GetBizHome, GetDashboard, profile
      orders.go             — Order CRUD, lifecycle (Send, Confirm, Reject, Fulfill, MarkPaid, Receipt)
      bookings.go           — Booking CRUD, lifecycle (MarkPaid, Receipt, Update)
      products.go           — Product CRUD, images
      services.go           — Service CRUD
      payments.go           — Payment management, confirm/reject, payment methods
      analytics.go          — Dashboard analytics
      subscription.go       — Subscription/billing, Stripe & PayPal webhooks
      public.go             — Public profile, client OTP connect
      business_widgets.go   — Quick booking/order/goal widgets
      profile_change_store.go — In-memory OTP-verified profile changes
    client/                — Client handlers (client_auth.go)
    guide.go               — Public /guide page handler
    seo.go                 — Sitemap, robots.txt
    routes/
      business_routes.go    — All /business/* route registrations (public + protected)
      routes.go             — Top-level routes (/, /guide, /b/:slug, etc.)
      client_routes.go      — Client /api/connect/* routes
    models/
      business.go           — Business model
      client.go             — Client model
      order.go              — Order, Booking, OrderItem, BookingItem, Payment models
      product.go            — Product model
      service.go            — Service model
      conversation.go       — Conversation, Message, Action models
      subscription.go       — SubscriptionPlan, BusinessSubscription models
    data/                  — Country/currency data
web/
  templates/
    pages/business/dashboard/  — Dashboard subpages
      orders.html            — Orders table (search/filter, action dropdown, 3-step wizard, receipt column)
      bookings.html          — Bookings table (search/filter, action dropdown, 3-step wizard, receipt column)
      products.html          — Products grid (CRUD, image gallery)
      services.html          — Services grid (CRUD)
      analytics.html         — Analytics dashboard
      payments.html          — Payments ledger
      receipt_order.html     — Print-receipt page for orders (standalone)
      receipt_booking.html   — Print-receipt page for bookings (standalone)
      order_confirmation.html — Post-creation confirmation page
    pages/business/          — Main business pages
      business.html          — Business home (client list + chat)
      dashboard.html         — Dashboard layout (sidebar + modal wizards)
      business_share.html    — Share page with QR code
      login/register/...     — Auth pages
    pages/guide.html         — Public guide page
    components/
      chat/                  — Chat cards (4-order-card, 4-booking-card, messages, etc.)
      ui/                    — Sidebar, bottom nav, dashboard sidebar
      modals/                — Payment method, profile, product/service pickers
      cards/                 — Client list with unread badges
      badges/                — Unread badge component
      dashboard_content.html — HTMX dashboard overview panel (stats + walk-in cards + recent items)
  static/
    css/
      styles.css             — Source CSS (variables + Tailwind directives)
      dist/styles.css        — Built output (minified)
    js/modules/
      client_chat.js         — Client chat rendering with dynamic order/booking status colors
      business_chat.js       — Business chat with order/booking action functions
      shared.js              — Toast notifications, confirm/prompt modals, cookie helpers
      theme.js               — Theme toggle (light/dark)
```

## Key Go Handlers

### Business Handler (`internal/handlers/business/business.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/` | `GetBizHome` | Client list + unread counts + pending badges |
| `GET /business/dashboard` | `GetDashboard` | Dashboard overview (stats, recent items, low stock) |
| `PUT /business/profile` | `UpdateBusinessProfile` | Update business name, email, currency, etc. |
| `POST /business/logo` | `UploadBusinessLogo` | Upload business logo image |
| `POST /business/regenerate-slug` | `RegenerateSlug` | Generate new public profile slug |
| `POST /business/profile/initiate` | `InitiateProfileChange` | Start OTP-verified profile change |
| `POST /business/profile/confirm` | `ConfirmProfileChange` | Confirm profile change with OTP |
| `POST /business/profile/resend-otp` | `ResendProfileOTP` | Resend OTP code |

### Orders (`internal/handlers/business/orders.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/orders` | `GetOrders` | Orders management page |
| `POST /business/orders` | `CreateOrder` | Create order (supports walk-in completed) |
| `PUT /business/orders/:id` | `UpdateOrder` | Update order details |
| `PUT /business/orders/:id/status` | `UpdateOrderStatus` | Update order status |
| `POST /business/orders/:id/send` | `SendOrderToClient` | Send draft → pending |
| `POST /business/orders/:id/confirm` | `ConfirmOrderBusiness` | Business confirms → confirmed |
| `POST /business/orders/:id/reject` | `RejectOrder` | Reject → cancelled |
| `POST /business/orders/:id/fulfill` | `FulfillOrder` | Fulfill → fulfilled |
| `PUT /business/orders/:id/paid` | `MarkOrderAsPaid` | Quick mark as fully paid |
| `GET /business/orders/:id/receipt` | `GetOrderReceipt` | Print receipt page |

### Bookings (`internal/handlers/business/bookings.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/bookings` | `GetBookings` | Bookings management page |
| `GET /business/bookings/:id` | `GetBooking` | Get single booking |
| `POST /business/bookings` | `CreateBooking` | Create booking (supports walk-in completed) |
| `PUT /business/bookings/:id` | `UpdateBooking` | Update booking details |
| `PUT /business/bookings/:id/status` | `UpdateBookingStatus` | Update booking status |
| `PUT /business/bookings/:id/paid` | `MarkBookingAsPaid` | Quick mark as fully paid |
| `GET /business/bookings/:id/receipt` | `GetBookingReceipt` | Print receipt page |

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

### Subscription (`internal/handlers/business/subscription.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `GET /business/subscription` | `GetSubscriptionPage` | Subscription management |
| `GET /business/subscription/plans` | `GetPlansPage` | Plans listing |
| `GET /business/subscription/checkout` | `GetCheckoutPage` | Checkout page |
| `POST /business/subscription/checkout` | `CreateCheckout` | Create checkout session |
| `POST /business/subscription/change` | `ChangePlan` | Change subscription plan |
| `POST /business/subscription/cancel` | `CancelSubscription` | Cancel subscription |
| `GET /business/subscription/portal` | `BillingPortal` | Stripe billing portal |
| `GET /business/subscription/badge` | `GetPlanBadge` | Plan badge for navbar |
| `GET /business/subscription/badge-sidebar` | `GetPlanBadgeSidebar` | Plan badge for sidebar |

### Client Widgets (`internal/handlers/business/business_widgets.go`)
| Route | Handler | Description |
|-------|---------|-------------|
| `POST /business/clients/:id/quick-booking` | `QuickBooking` | Quick booking from chat |
| `POST /business/clients/:id/quick-order` | `QuickOrder` | Quick order from chat |
| `POST /business/clients/:id/request-payment` | `RequestPayment` | Request payment from client |
| `POST /business/clients/:id/set-goal` | `SetGoal` | Set conversation goal |

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
| `GET /guide` | `ShowGuide` | `guide.go` |
| `GET /sitemap.xml` | `ServeSitemap` | `seo.go` |
| `GET /robots.txt` | `ServeRobots` | `seo.go` |
