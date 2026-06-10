## 0. HTMX Dashboard Refactor — 🚧 IN PROGRESS (Quick Fix)

**Goal:** Refactor the entire dashboard so sidebar/navbar remain static and only content partials are swapped via HTMX. Each page is modular: layout → content partial → sub-templates. Handlers detect `HX-Request` header to return only the fragment.

### 0.1 Dashboard Base Layout — ❌ NOT STARTED
- [ ] Create `web/templates/layouts/dashboard_base.html` — shared `<html>`, `<head>`, CSS, JS, navbar, sidebar, `<main>`, modals, assist panel, footer scripts
- [ ] Add JS sidebar active-state updater via `htmx:pushedUrl` (no longer depends on server-side `.ActivePage`)
- [ ] Remove 30s auto-refresh from layout (incompatible with HTMX nav)
- [ ] Keep step wizard JS, quick order/booking modal JS loaded once in layout

### 0.2 Refactor Core Pages — ❌ NOT STARTED
- [ ] `dashboard.html`: inline `{{template "dashboard_base" .}}`, keep `dashboard_content` define
- [ ] `orders.html`: remove HTML shell, inline `dashboard_base`, keep `orders_content` define
- [ ] `bookings.html`: same pattern
- [ ] `products.html`: same pattern + add `products_content` partial
- [ ] `services.html`: same pattern + add `services_content` partial
- [ ] `payments.html`: same pattern + add `payments_content` partial
- [ ] `analytics.html`: same pattern + add `analytics_content` partial

### 0.3 Refactor Secondary Pages — ❌ NOT STARTED
- [ ] `reports.html`: inline `dashboard_base`, keep report tab content as partials
- [ ] `reviews.html`: same pattern
- [ ] `hours.html`: same pattern
- [ ] `locations.html`: same pattern
- [ ] `team.html`: same pattern
- [ ] `business_share.html`: same pattern
- [ ] `subscription.html`: same pattern
- [ ] `notification_settings.html`: same pattern
- [ ] `business.html`: same pattern (customers/chat page)

### 0.4 Sidebar HTMX Navigation — ❌ NOT STARTED
- [ ] Add `hx-get`, `hx-target="#main-content"`, `hx-push-url="true"` to all sidebar links in `dashboard_sidebar.html`
- [ ] Keep `href` as fallback for non-JS access

### 0.5 Handler HX-Request Detection — ❌ NOT STARTED
- [ ] `GetOrders`: return `orders_content` on HX-Request
- [ ] `GetBookings`: return `bookings_content` on HX-Request
- [ ] `GetProducts`: return `products_content` on HX-Request
- [ ] `GetServices`: return `services_content` on HX-Request
- [ ] `GetPayments`: return `payments_content` on HX-Request
- [ ] `GetAnalytics`: return `analytics_content` on HX-Request
- [ ] `GetBizHome` (customers): return `business_content` on HX-Request
- [ ] `GetReportsPage`, `GetReviews`, `GetBusinessHours`, `GetLocations`, `GetTeam`, `GetSharePage`, `GetSubscriptionPage`, `GetNotificationSettings`: same pattern

### 0.6 Fix Pagination + Actions for HTMX — ❌ NOT STARTED
- [ ] Change arrow `hx-get="?page=N"` → use stats endpoint URLs (target content div, not full page)
- [ ] Replace `fetch()` + `location.reload()` action handlers with `hx-post`/`hx-put` + content swap
- [ ] Verify all lifecycle actions (confirm, fulfill, mark-paid, cancel) work with HTMX

### 0.7 Add Missing Content Partials — ❌ NOT STARTED
- [ ] `products_content` + `products_stats_grid` partials for products page
- [ ] `services_content` + `services_stats_grid` partials for services page
- [ ] `payments_content` + `payments_stats_grid` partials for payments page
- [ ] `analytics_content` + `analytics_stats_grid` partials for analytics page

### 0.8 Server-Side Filtering — ❌ NOT STARTED
- [ ] Replace client-side `filterOrders`/`filterBookings` JS with HTMX query param + server-side filtering
- [ ] Add search/filter query params to stats/content endpoint handlers
- [ ] Replace search input `onkeyup` → `hx-get` with debounce

### 0.9 JS Consolidation — ❌ NOT STARTED
- [ ] Move step wizard modal functions from inline `<script>` to `shared.js` module
- [ ] Move product/service picker loaders from inline to shared module
- [ ] Remove duplicated JS from orders.html and bookings.html

---

## 1. General UX / System Improvements — ✅ DONE (Phase 1)

### 1.1 UI/UX — ✅ DONE
- Standardize UI components (buttons, cards, charts, modals):
  - Added `.btn-danger`, `.btn-xs` CSS classes
  - Converted all modal buttons (8 modals, ~18 buttons) to use `.btn`, `.btn-sm`, `.btn-primary`, `.btn-ghost` component classes
  - Consolidated page-level `<style>` blocks: `.badge-status`, `.table-row-hover`, `.sidebar-item`, `.card-hover`, `.stat-card` moved to `styles.css`; removed 5 duplicate `<style>` blocks from orders.html, payments.html, products.html, services.html, analytics.html
  - Eliminated 25 inline `style` attributes from business.html stat cards via `.biz-stat` CSS variable theming (`.biz-stat-primary/warning/info/success`)
  - Moved assist panel inline text colors (7 inline styles) into `.assist-header-content`, `.assist-header-icon`, `.assist-header-title`, `.assist-header-subtitle`, `.assist-empty-icon/text/hint` CSS classes
- Improve visual hierarchy (colors, spacing, typography)
- Empty states for all list views (clients, orders, bookings, products, services)

### 1.2 Performance — ✅ DONE
- Optimize analytics queries (GROUP BY, SQL date_trunc, combined review query)
- Add DB indexing for frequently queried fields (composite indexes on orders, bookings, payments, messages, products, services, clients)
- Implement pagination for large datasets:
  - Orders: `GetOrders` + `GetOrdersStats` (HTMX) — page size 20, GROUP BY counts, SUM revenue
  - Bookings: `GetBookings` + `GetBookingsStats` (HTMX) — same pattern
  - Products: `GetProducts` — page size 20, preserves location filter
  - Services: `GetServices` — page size 20, preserves location filter
  - Payments: `GetPayments` — page size 20, preserves location filter
  - Stats grids (`GetOrdersStatsGrid`, `GetBookingsStatsGrid`) — GROUP BY optimization (was loading all rows)
  - Pagination UI with Previous/Next links, "Page X of Y" counter, range filter resets to page 1

### 1.3 UX Enhancements
- Command palette (Cmd+K) for quick navigation and actions
- Page transitions (CSS view transitions)
- Animated micro-interactions (button scale, card hover)

### 1.4 Responsiveness
- Ensure full mobile responsiveness across all modules
- Breakpoints: desktop (1280+), tablet (768–1279), mobile (<768)  and more, as adaptive as possible

---

## 2. Architecture Refactor: Hexagonal Architecture (Ports & Adapters) — ❌ NOT STARTED

### Target
Refactor into clean hexagonal architecture: Domain → Application → Infrastructure → Delivery.

### Migration Strategy
1. Extract services from handlers
2. Introduce interfaces for repositories
3. Gradually move logic into use-case layer
4. Keep system running during refactor (incremental)

### Outcome
- Highly scalable SaaS architecture
- Easy testing (mockable layers)
- Clean separation of concerns
- Enterprise-ready

---

## 3. Feature Roadmap

### Tier 1 — High Value, Launch-adjacent

#### 3.1 Promotions, Discounts & Coupons
- Discount codes (%, fixed amount, BOGO, bundle deals)
- Auto-apply promos (first-time client, seasonal, holiday)
- Limited-time offers with expiry
- Coupon redemption tracking

### Tier 2 — Operations & Growth

#### 3.2 Broadcast / Announcements
- One-to-many messages to all clients or filtered segments
- Promotional announcements, service updates
- Opt-out per client

### Tier 3 — Monetization & Differentiation

#### 3.3 Customer Loyalty / Rewards Program
- Points-per-spend or per-visit
- Redeem points for discounts or free services
- Tie into existing CustomerInsight tier system
- Birthday/anniversary automated rewards
- Points balance visible in client dashboard

#### 3.4 Gift Cards & Vouchers
- Sell gift cards (fixed amount or specific service)
- Redeem at checkout
- Balance tracking
- Send via email

#### 3.5 Invoice / Billing System
- Formal invoices with invoice #, tax info, business details
- PDF download + email to client
- Payment terms (due on receipt, net 15/30)
- Deposit / partial payment plans

#### 3.6 Tax Configuration
- Tax rates per country/location
- Auto-apply tax to orders/bookings
- Tax-inclusive or exclusive pricing
- VAT/GST support
- Tax report for filing

### Tier 4 — Scale & Integrations

#### 3.7 Product / Service Categories
- Categorize products and services
- Category-based filtering on public page and dashboards

#### 3.8 Waitlist
- Join waitlist for fully booked slots
- Auto-notify when slot opens
- Priority based on client tier

#### 3.9 Integrations
- Calendar sync (Google Calendar, iCal)
- Accounting sync (QuickBooks, Xero)
- SMS notifications (Twilio)
- Webhooks / Zapier / Make
- Public REST API

---

## 4. Creative Additions (Backlog)

- **Achievement badges** — "First Connect", "10 Orders", "5-Star Service"
- **Sound design** — optional notification sounds
- **"salesmee Streak"** — daily active usage counter
- **Conversation themes** — per-business chat color themes
