# Loading UX Improvement Plan (Complete)

## Goal

Eliminate all blank flashes, jarring content swaps, and loading feedback gaps across every business sub-page and chat container. Every async content load should show an instant visual placeholder and transition smoothly to real content.

## Principles

- **Skeleton-first**: Every HTMX-loaded region renders a skeleton as initial content, replaced on `hx-trigger="load"` or user action
- **Zero blank flashes**: No `hx-indicator` without visible skeleton content (the indicator IS the skeleton)
- **Morph on stats**: Use `hx-swap="morph"` + idiomorph for stat grids to preserve layout, avoid card flicker
- **Chat skeleton**: Show shimmer message bubbles while conversation loads, not a blank area

## Completed

### Phase 1 — Skeleton Templates (9 new in `skeletons.html`)

| Template | Purpose |
|----------|---------|
| `skeleton_stats_4` | 4 stat cards (Payments, Analytics) |
| `skeleton_stats_6` | 6 stat cards (Bookings) |
| `skeleton_orders_table` | 6 table rows with shimmer |
| `skeleton_bookings_table` | 6 table rows with shimmer |
| `skeleton_revenue_report` | 3 summary cards + chart bar shimmers |
| `skeleton_sales_report` | 5 rows with product icon placeholders |
| `skeleton_clients_report` | 2 stat cards + chart bar |
| `skeleton_tax_report` | 12 month-row shimmers |
| `skeleton_chat_messages` | Header bar + 4 message bubbles (2 in, 2 out) |
| `skeleton_table` | Existing: 5 rows (reused by orders/bookings stats pages) |
| `skeleton_stats` | Existing: 8 cards (reused by stats page) |

### Phase 2 — Reports Page

- [x] Static placeholder replaced with `{{template "skeleton_revenue_report" .}}` + `hx-trigger="load once"`
- [x] `hx-indicator="#reportContent"` added to all 4 tab buttons
- [x] Hidden `<div id="skeleton-*-report">` elements embedded for JS access
- [x] `loadRange()`, `applyCustomRange()`, `switchReportTab()` all call `showSkeletonReport(tab)` before `htmx.ajax()`
- [x] `skeletonReportContent()` function maps tab name to skeleton element ID

### Phase 3 — Orders Page

- [x] Eager `{{template "orders_stats_grid" .}}` replaced with skeleton + `hx-trigger="load once"`
- [x] `hx-ext="morph"` wrapped on `#orders-stats-container`
- [x] `hx-swap="morph"` on stats grid for smooth transitions
- [x] `loading-overlay` class on `#orders-page` (spinner + blur overlay on pagination/period-swap)
- [x] `hx-indicator="#orders-page"` on pagination prev/next links

### Phase 4 — Bookings Page

- [x] Same as Orders: skeleton stats + `hx-trigger="load"` + morph + `loading-overlay` + `hx-indicator`

### Phase 5 — Payments Page

- [x] Eager `{{template "payments_stats_grid" .}}` replaced with `{{template "skeleton_stats_4" .}}` + `hx-trigger="load"`
- [x] `hx-indicator="#payments-stats-container"` on all 6 pill buttons
- [x] `hx-swap="outerHTML"` → `hx-swap="morph"` for smooth stat transitions
- [x] `hx-ext="morph"` on stats container

### Phase 6 — Analytics Page

- [x] `{{template "analytics_stats_grid" .}}` replaced with `{{template "skeleton_stats_4" .}}` + `hx-trigger="load"`
- [x] `hx-swap="outerHTML"` → `hx-swap="morph"` on all 6 pill buttons
- [x] `hx-ext="morph"` on stats container (already had `hx-indicator`)

### Phase 7 — Chat Container Loading

- [x] Existing `#skeleton-area` with `{{template "skeleton_chat" .}}` wired into `loadClient()` JS
- [x] `loadClient()` now: hides `#content-area` → shows `#skeleton-area` → fetches → hides skeleton → fills `#content-area` → adds `content-fade-in` class
- [x] `content-fade-in` animation: 0.25s ease-out, opacity + translateY(4px→0)

### Phase 8 — Auto-Select (Not Done)

Deferred. Virtual scroll + sidebar sorting should stabilize first. Can revisit.

### Phase 9 — CSS Fixes & Polish

- [x] **Fixed missing `@keyframes shimmer`** — was referenced by `.skeleton::after` animation but never defined. Skeletons now actually shimmer.
- [x] Added `content-fade-in` keyframe for skeleton→content transitions
- [x] Added `loading-overlay` class: spinner + blur overlay on `htmx-request`
- [x] Added `@keyframes spin` for overlay spinner
- [x] Added `stat-loading` dim effect class for stat containers

## Key Decisions

- **Kept period filter as full-page swap** for Orders/Bookings — the table data depends on the filtered range, so a targeted stats-only swap would leave stale table rows. The `loading-overlay` class provides visual feedback during the swap instead.
- **Chat used existing `skeleton_chat` template** — the inline skeleton chat markup from the plan was unnecessary since `empty_state.html` already defined `{{define "skeleton_chat"}}` and `business.html` already had `#skeleton-area` wired. Only `loadClient()` JS needed updating.
- **No backend changes needed** — all existing handlers work as-is. `GetOrdersStatsGrid`, `GetBookingsStatsGrid`, `GetPaymentsStatsGrid`, and `GetAnalyticsStatsGrid` endpoints already existed and accept the `range` query param.

## Files Modified

| File | Changes |
|------|---------|
| `web/templates/components/ui/skeletons.html` | Added 9 new skeleton templates |
| `web/templates/pages/business/dashboard/reports.html` | Skeleton + indicator + hidden skeleton divs + JS updates |
| `web/templates/pages/business/dashboard/orders.html` | Skeleton stats + morph + loading-overlay + indicator |
| `web/templates/pages/business/dashboard/bookings.html` | Same as orders |
| `web/templates/pages/business/dashboard/payments.html` | Skeleton stats + morph + indicator |
| `web/templates/pages/business/dashboard/analytics.html` | Skeleton stats + morph swap |
| `web/templates/pages/business/business.html` | `content-fade-in` class on content-area |
| `web/static/js/modules/business.js` | Skeletons in loadClient(), content-fade-in, autoSelectFirstChat() |
| `web/static/js/modules/client.js` | `loadBusiness()` uses fetch() + skeleton skeleton-area |
| `web/static/js/modules/client_chat.js` | Client sidebar virtual scrolling |
| `web/templates/pages/client/client.html` | Skeleton-area moved outside chat-area, switchToChats/openClientProducts/openClientServices use fetch() + skeleton |
| `web/templates/pages/client/client_discover.html` | Discover skeleton cards on search |
| `web/templates/pages/client/client_discover_content.html` | Discover skeleton cards on search |
| `web/templates/components/ui/skeletons.html` | Added skeleton_discover_cards |
| `web/static/css/components.css` | Added shimmer keyframes, content-fade-in, loading-overlay, spin, img-skeleton, img-fade-in |
| `web/static/js/core/app.js` | `initLazyImages()` + `initSkeletonTimeouts()` — lazy image loading, skeleton timeout/retry, `showSkeletonError()`, `retrySkeleton()` |
| `web/templates/pages/business/dashboard/products.html` | Img wrapped in img-skeleton for blur-up, pagination overlay |
| `web/templates/pages/business/dashboard/services.html` | Img wrapped in img-skeleton for blur-up, pagination overlay |
| `web/templates/pages/client/client_products.html` | Img wrapped in img-skeleton for blur-up |
| `web/templates/pages/client/client_services.html` | Img wrapped in img-skeleton for blur-up |
| `web/static/js/modules/business.js` | `loadClient()` skeleton timeout + error state on catch |
| `web/static/js/modules/client.js` | `loadBusiness()` skeleton timeout + error state on catch |
| `web/templates/pages/client/client.html` | `switchToChats`/`openClientProducts`/`openClientServices` — timeout + error state |
| `web/templates/components/ui/skeletons.html` | Added `skeleton_discover_cards`, `skeleton_error` |

## Sprint 6 — Remaining Loading UX (Completed)

- **Auto-select first conversation**: `autoSelectFirstChat()` in business.js selects the first client with unread messages (or first client) on page load. Runs 400ms after DOMContentLoaded.
- **Client chat skeleton**: `loadBusiness()` uses fetch() instead of htmx.ajax(), shows skeleton-area before request, hides on response. `switchToChats()`, `openClientProducts()`, `openClientServices()` also wired with skeleton.
- **Discover search skeleton**: Both standalone discover page and discover content partial show 5 skeleton cards during search fetch.
- **Blur-up image placeholders**: Product/service images (business + client grids) wrapped in `.img-skeleton` container. `img-fade-in` class on images transitions opacity 0→1 on load. `img-fallback` shown on error. Handled by `initLazyImages()` in app.js with HTMX `afterSwap` re-init.
- **Pagination loading indicator**: Products and Services pages show a spinner overlay when pagination links are clicked.
- **HTMX skeleton timeout + error state**: `initSkeletonTimeouts()` in app.js: 20s timeout on HTMX skeleton requests, replaces skeleton with error icon + "Failed to load" + Retry button on timeout/error. `showSkeletonError()` and `retrySkeleton()` global functions. Fetch-based chat loading (loadClient, loadBusiness, switchToChats, openClientProducts/Services) also have 20s timeout + error state on catch.
- `go vet ./...` and `go build ./...` pass.
