# Production Hardening Plan

> Status: Planned · Priority labels are relative within severity tiers.

## HIGH Priority

### H1. Stored XSS in Product & Service Pickers

**Files:**
- `web/static/js/modules/product_picker.js` — `pickerRenderProducts()` builds innerHTML from product name, SKU, description, image URL (lines 284–329)
- `web/static/js/modules/service_picker.js` — `renderServicePicker()` builds innerHTML from service name, description, price, category (lines 250–274), and category filter buttons (line 217)
- Both fetch JSON from `/business/conversations/:id/products` and `/business/conversations/:id/services`

**Vector:** A business creates a product with a malicious name/description containing `<script>`. When a client opens the picker, the JS renders it via innerHTML without sanitization. Stored XSS — the payload lives in the DB and executes on every load.

**Fix:**
1. Sanitize all user-supplied fields (`name`, `description`, `sku`, `category`) before inserting into innerHTML — use `textContent` for text nodes, or `encodeURIComponent`/custom sanitizer for attribute values.
2. Apply the same sanitization to the service picker's category filter buttons (line 217) where `cat` is injected directly into the button's onclick handler.
3. Add Content-Security-Policy header to prevent inline script execution as defense-in-depth.

---

### H2. Nil Pointer Dereferences in Client Auth

**File:** `internal/handlers/client/client_auth.go`

**Issue A — `getOrCreateConversation` (lines 236–239):**
```go
var client models.Client
if err := db.DB.Where("id = ?", clientID).First(&client).Error; err != nil {
    log.Print("Client not found by id ", clientID)
}
```
The `err` is logged but never returned. The function returns `&client` which is a zero-value struct (ID=0). Callers (`GetClientMessages`, `CreateClientMessage`, `ClientDashboard`) then use `client.ID` in queries — orders with `client_id = 0` return zero results.

**Fix:** Return an error when the client is not found instead of silently swallowing.

**Issue B — `ClientUpdateOrder` (lines 726–738):**
In multi-item JSON update, stock is restored for old items before verifying new items are valid. If a product lookup fails (`db.DB.First(&product, item.ProductID)` errors), the old stock has already been incremented but no new stock is decremented, leaving inventory inconsistent.

**Fix:** Use a DB transaction — roll back if any item validation fails.

---

### H3. Admin Login Not Rate-Limited

**Files:**
- `internal/middleware/ratelimit.go` — `rateLimitedPaths` and `rateLimitedPrefixes` define what gets rate-limited (line 93–104)
- `internal/routes/admin_routes.go` — `POST /admin/login` (line 12) is unprotected

The admin login endpoint is not in any rate-limited path list. Attackers can brute-force credentials via the admin login form with no throttling.

**Fix:** Add `/admin/login` to `rateLimitedPaths` in `ratelimit.go`. Consider adding a dedicated admin-only rate limiter with lower thresholds (e.g., 3 req/s, burst 5) vs the auth limiter (5 req/s, burst 10).

---

## MEDIUM Priority

### M1. Hardcoded Production Email & Domain

**File:** `web/templates/pages/legal/*.html`, `web/templates/pages/business/business_login.html`, `web/templates/pages/error_500.html`, `web/templates/partials/landing/`, `internal/handlers/dev.go`

11 occurrences of `support@salesmee.com` hardcoded across error pages, legal pages (privacy, terms, cookies, refund), checkout, and landing page. Should be configurable.

Additional items:
- `business_login.html:117` — Demo credentials `demo@salesmee.com / password` shown unconditionally (not gated behind dev env)
- `landing_head.html:11` — `og:url` hardcoded to `https://salesmee.com`
- `dev.go:80–95` — Email preview templates hardcode `https://app.salesmee.com`

**Fix:**
1. Add `SupportEmail` to config and inject into template data.
2. Gate demo credentials in login template behind `{{if eq .Env "dev"}}`.
3. Read `AppDomain` config for meta tags and email templates.
4. Move email template URLs to config values.

---

### M2. Hardcoded Demo Credentials in Login Template

**File:** `web/templates/pages/business/business_login.html:117`

```html
<div><span class="font-medium">Demo:</span> demo@salesmee.com / password</div>
```

Shown unconditionally on the business login page. Exposes valid-looking credentials and an email pattern. Should be gated behind dev/staging environment.

**Fix:** Gate behind `{{if .IsDev}}` condition using config-driven template variable.

---

### M3. Client Token Cookie Missing Secure Flag

**File:** `internal/handlers/client/client_auth.go:135`

```go
c.SetCookie("client_token", token, 86400, "/", "", false, false)
```

Unlike the social auth handler which sets `secure := config.C.AppEnv != "dev"`, the OTP-based auth path hardcodes `false` for `secure`. In production, the token cookie can be sent over unencrypted HTTP connections.

**Fix:** Use `config.C.AppEnv != "dev"` for the secure flag, matching the pattern in `social_auth.go`.

---

### M4. In-Memory Registration Store (No Expiry)

**File:** `internal/handlers/register.go`

The `RegStore` in-memory store holds uncompleted OAuth registration data with no TTL. Data accumulates until the app restarts.

**Fix:** Add a TTL (e.g., 15 minutes) with a background cleanup goroutine, or switch to a cookie/token-encoded store.

---

### M5. WebSocket Rate Limiting

**File:** `internal/ws/hub.go`

WebSocket connections can send unlimited messages through the system without rate limiting. No per-connection or per-IP throttling exists for WS message delivery.

**Fix:** Add per-connection rate limiting (e.g., token bucket per WS session) with appropriate thresholds for chat vs broadcast messages.

---

### M6. Audit Log Stored Action Mismatch

**File:** `internal/handlers/admin/admin.go:297,325,394`

`DeleteBusiness` stores `action: "delete", resource: "business"`; `DeleteClient` stores `action: "delete", resource: "client"`. The audit filter previously offered `delete_business`/`delete_client` as separate action values (matched action, not resource+action combo). **This was already partially fixed** by mapping both to `"delete"` in the handler, but the dropdown options still show "Delete Business" / "Delete Client" as action-level filters when they're really filter combinations of action+resource.

**Fix (cleanup):** Replace the two delete action dropdown entries with a single "Delete" option. Or keep the split labels but filter on both `action` AND `resource` columns.

---

## LOW Priority

### L1. Hardcoded Page Size in Admin Templates

Pagination buttons in all 4 admin templates hardcode `page=1` in HTMX URLs. If a user navigates to page 5 and then applies a filter, the request goes to `page=1`, resetting pagination. This is a minor UX annoyance.

**Fix:** Include `page` in the filter form and carry it through.

---

### L2. No CSRF Token on Admin Login Form

**File:** `web/templates/pages/admin/admin_login.html`

The admin login form `<form method="POST">` has no CSRF hidden input or `hx-post`. It can be targeted by CSRF attacks from other origins.

**Fix:** Add CSRF middleware to `POST /admin/login` route (may require adjusting `CSRFMiddleware()` to not skip GET + specific paths), or add a custom CSRF token to the login form.

---

### L3. Admin Logout Missing POST Method

**File:** `internal/routes/admin_routes.go:18`

```go
adminGroup.GET("/logout", admin.AdminLogout)
```

Uses GET for logout. This is susceptible to CSRF-based logout (an attacker can trigger logout via `<img src="/admin/logout">`).

**Fix:** Change to POST and use a form.

