# SalesMee — Implementation Plan

**Priority:** Critical → High → Medium → Low

---

## Phase 1: Security Hardening

### 1.1 Secure Cookie Helper
**Files:** `internal/handlers/auth.go`, `internal/handlers/client/client_auth.go`, `internal/handlers/business/team_auth.go`, `internal/middleware/csrf.go`, `internal/handlers/admin/admin.go`

**Changes:**
- Create `internal/services/cookie.go` with `SetSecureCookie()` helper
- Set `Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode`
- Gate `Secure` behind env check so local dev works over HTTP
- Replace all raw `c.SetCookie()` calls

### 1.2 Fix Admin Cookie
**File:** `internal/handlers/admin/admin.go`

**Changes:**
- Replace `email:password[:20]` cookie with JWT signed by `JWT_SECRET`
- Store `admin_id`, `admin_email` as claims
- Validate via existing AdminMiddleware

### 1.3 Fix Host Header Injection
**Files:** `internal/handlers/seo.go`, `internal/handlers/business/business.go`, `internal/handlers/business/subscription.go`, `internal/handlers/auth.go`, `internal/services/email.go`

**Changes:**
- Read `APP_URL` from env, store globally or pass as config
- Replace `c.Request.Host` usage with configured base URL
- Add a `baseURL()` helper or pass `BaseURL` param

### 1.4 Add Rate Limiting to OTP/Auth Endpoints
**Files:** `internal/middleware/ratelimit.go`, `internal/routes/business_routes.go`, `internal/routes/client_routes.go`

**Changes:**
- Extend `rateLimitedPaths` to include `/client/send-otp`, `/client/verify-otp`, `/api/connect/:slug`, `/business/register*`, `/business/team/login`
- Use `RateLimitAuth()` on OTP endpoints (5/s burst 10)
- Add per-IP + per-email rate limiting for OTP verify

### 1.5 Harden WebSocket Origin Check
**File:** `internal/ws/handler.go`

**Changes:**
- Replace `return true` with origin validation against configured allowlist
- Read `ALLOWED_ORIGINS` from env (comma-separated), fall back to `APP_URL`

### 1.6 Add Security Headers Middleware
**Files:** `internal/middleware/security.go` (new), `internal/routes/routes.go`

**Changes:**
- Create `SecurityHeadersMiddleware()` setting: CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- Apply globally in route setup

### 1.7 Add Panic Recovery
**File:** `cmd/server/main.go`

**Changes:**
- Add `router.Use(gin.Recovery())` after `gin.Default()` or explicitly

### 1.8 Fix OTP Logging Leakage
**Files:** `internal/services/customer_auth.go`, `internal/handlers/business/business.go`

**Changes:**
- Replace `fmt.Printf` OTP logging with structured debug-only logger (e.g., `slog.Debug`)
- Mask OTP in log output, only log email + "OTP sent"

### 1.9 Centralize Env Var Config
**New File:** `internal/config/config.go`

**Changes:**
- Create a `Config` struct with all env vars
- Validate required vars at startup
- Replace raw `os.Getenv()` across codebase

---

## Phase 2: Reliability

### 2.1 Error Handling for Silent DB Discards
**Files:** `internal/handlers/business/orders.go`, `bookings.go`, `payments.go`, `business.go`, `products.go`, `services.go`

**Changes:**
- Audit every `h.db.Save()`, `h.db.Create()`, `h.db.Update()`, `h.db.Delete()` that discards error
- Wrap in `if err := ...; err != nil { log.Error(...); c.JSON(500, ...); return }`
- Add `dbSave()` / `dbCreate()` helper methods on `BusinessHandler` that handle logging

### 2.2 Fix Stock Deduction Race Condition
**File:** `internal/handlers/business/orders.go`

**Changes:**
- Use `db.Clauses(clause.Locking{Strength: "UPDATE"})` when reading product stock
- Or use optimistic locking with a `version` column on Product

### 2.3 Move OTP Store to DB
**File:** `internal/handlers/business/profile_change_store.go`

**Changes:**
- Create `ProfileChangeRequest` model in DB
- Move in-memory map to DB-backed storage
- Add TTL cleanup via GORM `-` or `deleted_at`

---

## Phase 3: Maintainability

### 3.1 Create Status Constants
**New File:** `internal/models/status.go`

**Changes:**
- Define `StatusPending`, `StatusConfirmed`, `StatusFulfilled`, `StatusCancelled`, etc. as typed constants
- Replace all string literals across handler files

### 3.2 Refactor BusinessHandler God Object
**Files:** Split `internal/handlers/business/` into sub-handlers

**Changes:**
- Create `OrderHandler`, `BookingHandler`, `PaymentHandler`, `ProductHandler`, `ServiceHandler`, `ProfileHandler`, `AnalyticsHandler`, `ReportHandler`, `TeamHandler`, `LocationHandler`, `HoursHandler`
- Each gets its own file with receiver on its own struct
- Share DB + Hub via a common `HandlerDeps` struct

### 3.3 Consolidate Duplicated JS Code
**Files:** `web/static/js/modules/business_chat.js`, `client_chat.js`, `product_picker.js`, `service_picker.js`

**Changes:**
- Extract shared functions into `chat_common.js`: `escapeHtml`, `formatTime`, `scrollToBottom`, `markAsRead`, `tickSvg`, `setMessageTickState`, `renderMediaMessage`, `showTypingIndicator`, `hideTypingIndicator`
- Create a `Wizard` base module for the 3-step picker pattern
- Use `const`/`let` consistently, wrap in IIFE or ES module

### 3.4 Split Monolithic CSS
**File:** `web/static/css/styles.css`

**Changes:**
- Split into: `tokens.css`, `layout.css`, `components.css`, `utilities.css`
- Each file scoped to `@layer`

---

## Phase 4: Polish

### 4.1 Replace `location.reload()` with HTMX Partial Swaps
**Files:** `web/templates/pages/business/dashboard/orders.html`, `bookings.html`

**Changes:**
- Replace inline JS `location.reload()` handlers with HTMX `hx-*` attributes
- Use `hx-swap` to update only the table body / stats grid

### 4.2 Use `http` Constants
**File:** `internal/handlers/business/business.go`

**Change:** Replace numeric status codes with `http.Status*` constants.

### 4.3 Remove Dead Code
**File:** `internal/handlers/business/business.go`

**Change:** Remove `GetLogoUploadPage` (unrouted duplicate of `GetDashboard`).

============= Tests ===============

## Browser Test Steps

### Phase 1 — Security
1. **Cookies:** Log in as business owner. DevTools → Application → Cookies. Verify all cookies have `HttpOnly` + `SameSite: Strict`. `Secure` flag present unless on dev.
2. **Security headers:** DevTools → Network tab → response headers on any page. Look for `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`.
3. **OTP not leaked:** Trigger "forgot password" or login OTP. Verify no OTP value in terminal logs (only `[DEV] OTP for email: ...` in dev mode).
4. **Rate limiting:** Click "Send OTP" rapidly 10+ times. After ~5 requests you should see 429 responses.

### Phase 2 — Reliability
5. **DB errors:** Create an order with invalid/missing data. Should see a server error notification, not a blank page.
6. **Stock race:** Open two tabs, create orders for the same product simultaneously. Both succeed or one fails gracefully — never negative stock.

### Phase 3 — Maintainability (visual)
7. **CSS split:** Load any page — cards, buttons, sidebar, chat bubbles, inputs all render correctly.
8. **JS consolidation:** Open Orders/Bookings → 3-step Quick Wizard works. Chat messages render properly.

### Phase 4 — Polish (interaction)
9. **Orders (no reload):** Click Send/Confirm/Mark Paid/Complete/Cancel. Page updates without full browser reload (no flash, scroll preserved).
10. **Bookings (no reload):** Same — Approve/Mark Paid/Complete/Cancel.
11. **Products:** Add/edit/delete via modal. Grid updates without reload.
12. **Services:** Same as products.
13. **Payments:** Confirm/reject a payment claim. Ledger updates without reload.
14. **Share page:** Click "Regenerate Links". QR code + slug update without reload.
