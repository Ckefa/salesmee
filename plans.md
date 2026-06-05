## 3. Multi-User Business Login + Roles (RBAC) — ❌ NOT STARTED

Enable multi-user access per business with role-based permissions.

### Roles
- **Manager** — manage users, view all analytics, edit settings, manage orders/bookings
- **Regular User** — view assigned data, limited order/booking handling, no settings

### Permissions System
- `orders:read`, `orders:write`, `analytics:view`, `customers:manage`, `settings:manage`

### Additional
- User invitation system
- Role assignment per business
- Secure session handling per business context

---

## 4. General UX / System Improvements — ⏳ IN PROGRESS

### ✅ Implemented
- Dark mode toggle (CSS vars + localStorage + prefers-color-scheme)
- Stackable toast notification system with auto-dismiss progress bar
- HTMX loading indicators on buttons
- CSRF protection middleware
- Rate limiting
- Cookie consent banner
- Responsive bottom nav on mobile

### 🎯 Still Wanted

#### UI/UX
- Standardize UI components (buttons, cards, charts, modals)
- Improve visual hierarchy (colors, spacing, typography)
- Empty states for all list views (clients, orders, bookings, products, services)

#### Performance
- Optimize analytics queries
- Add DB indexing for frequently queried fields
- Implement pagination for large datasets

#### UX Enhancements
- Skeleton loaders for dashboard stats, progress tracking, analytics
- Command palette (Cmd+K) for quick navigation and actions
- Page transitions (CSS view transitions)
- Animated micro-interactions (button scale, card hover)

#### Responsiveness
- Ensure full mobile responsiveness across all modules
- Breakpoints: desktop (1280+), tablet (768–1279), mobile (<768)

---

## 5. Architecture Refactor: Hexagonal Architecture (Ports & Adapters) — ❌ NOT STARTED

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

## 6. Feature Roadmap

### Tier 1 — High Value, Launch-adjacent

#### 6.1 Promotions, Discounts & Coupons
- Discount codes (%, fixed amount, BOGO, bundle deals)
- Auto-apply promos (first-time client, seasonal, holiday)
- Limited-time offers with expiry
- Coupon redemption tracking

#### 6.2 Automated Notifications & Reminders
- Booking reminders (1h/24h before) via email
- Order status change notifications to client
- Payment due reminders
- Abandoned cart follow-ups (pending orders)
- Re-engagement for inactive clients
- Configurable timing and channels per business

#### 6.3 Reviews & Ratings
- Client rates completed orders/bookings (1–5 stars + comment)
- Average rating displayed on public profile `/b/:slug`
- Business can respond to reviews
- Aggregate rating analytics

#### 6.4 Reporting & Exports
- Revenue reports (daily/weekly/monthly/custom range)
- Sales by product/service
- Staff performance reports
- Client acquisition trends
- CSV/PDF export of orders, bookings, payments, clients
- Tax summary report

### Tier 2 — Operations & Growth

#### 6.5 Staff / Team Management
- Staff profiles (name, role, photo, contact)
- Assign staff to services
- Commission tracking per order/booking per staff
- Staff activity log

#### 6.6 Business Hours & Availability
- Weekly recurring hours
- Special hours / holiday closures
- Buffer time between bookings
- Max bookings per time slot
- Online/offline toggle for public profile

#### 6.7 Client Portal Self-Service
- View/manage appointments
- Cancel/reschedule bookings (within business rules)
- Reorder past orders
- Save favorites/bookmarked services
- Notification preferences

#### 6.8 Broadcast / Announcements
- One-to-many messages to all clients or filtered segments
- Promotional announcements, service updates
- Opt-out per client

### Tier 3 — Monetization & Differentiation

#### 6.9 Customer Loyalty / Rewards Program
- Points-per-spend or per-visit
- Redeem points for discounts or free services
- Tie into existing CustomerInsight tier system
- Birthday/anniversary automated rewards
- Points balance visible in client dashboard

#### 6.10 Gift Cards & Vouchers
- Sell gift cards (fixed amount or specific service)
- Redeem at checkout
- Balance tracking
- Send via email

#### 6.11 Invoice / Billing System
- Formal invoices with invoice #, tax info, business details
- PDF download + email to client
- Payment terms (due on receipt, net 15/30)
- Deposit / partial payment plans

#### 6.12 Tax Configuration
- Tax rates per country/location
- Auto-apply tax to orders/bookings
- Tax-inclusive or exclusive pricing
- VAT/GST support
- Tax report for filing

### Tier 4 — Scale & Integrations

#### 6.13 Multi-Location / Multi-Branch
- Multiple locations per business account
- Per-location inventory, staff, products, services
- Location selector on public profile
- Reporting by location

#### 6.14 Product / Service Categories
- Categorize products and services
- Category-based filtering on public page and dashboards

#### 6.15 Waitlist
- Join waitlist for fully booked slots
- Auto-notify when slot opens
- Priority based on client tier

#### 6.16 Integrations
- Calendar sync (Google Calendar, iCal)
- Accounting sync (QuickBooks, Xero)
- SMS notifications (Twilio)
- Webhooks / Zapier / Make
- Public REST API

---

## Creative Additions (Backlog)

- **"salesmee Assist"** — floating AI assistant chatbot
- **Achievement badges** — "First Connect", "10 Orders", "5-Star Service"
- **Sound design** — optional notification sounds
- **"salesmee Streak"** — daily active usage counter
- **Conversation themes** — per-business chat color themes
