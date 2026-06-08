## 1. General UX / System Improvements — ❌ NOT STARTED

### UI/UX
- Standardize UI components (buttons, cards, charts, modals)
- Improve visual hierarchy (colors, spacing, typography)
- Empty states for all list views (clients, orders, bookings, products, services)

### Performance
- Optimize analytics queries
- Add DB indexing for frequently queried fields
- Implement pagination for large datasets

### UX Enhancements
- Skeleton loaders for dashboard stats, progress tracking, analytics
- Command palette (Cmd+K) for quick navigation and actions
- Page transitions (CSS view transitions)
- Animated micro-interactions (button scale, card hover)

### Responsiveness
- Ensure full mobile responsiveness across all modules
- Breakpoints: desktop (1280+), tablet (768–1279), mobile (<768)

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

#### 3.2 Staff / Team Management
- Staff profiles (name, role, photo, contact)
- Assign staff to services
- Commission tracking per order/booking per staff
- Staff activity log

#### 3.3 Broadcast / Announcements
- One-to-many messages to all clients or filtered segments
- Promotional announcements, service updates
- Opt-out per client

### Tier 3 — Monetization & Differentiation

#### 3.4 Customer Loyalty / Rewards Program
- Points-per-spend or per-visit
- Redeem points for discounts or free services
- Tie into existing CustomerInsight tier system
- Birthday/anniversary automated rewards
- Points balance visible in client dashboard

#### 3.5 Gift Cards & Vouchers
- Sell gift cards (fixed amount or specific service)
- Redeem at checkout
- Balance tracking
- Send via email

#### 3.6 Invoice / Billing System
- Formal invoices with invoice #, tax info, business details
- PDF download + email to client
- Payment terms (due on receipt, net 15/30)
- Deposit / partial payment plans

#### 3.7 Tax Configuration
- Tax rates per country/location
- Auto-apply tax to orders/bookings
- Tax-inclusive or exclusive pricing
- VAT/GST support
- Tax report for filing

### Tier 4 — Scale & Integrations

#### 3.8 Multi-Location / Multi-Branch
- Multiple locations per business account
- Per-location inventory, staff, products, services
- Location selector on public profile
- Reporting by location

#### 3.9 Product / Service Categories
- Categorize products and services
- Category-based filtering on public page and dashboards

#### 3.10 Waitlist
- Join waitlist for fully booked slots
- Auto-notify when slot opens
- Priority based on client tier

#### 3.11 Integrations
- Calendar sync (Google Calendar, iCal)
- Accounting sync (QuickBooks, Xero)
- SMS notifications (Twilio)
- Webhooks / Zapier / Make
- Public REST API

---

## 4. Creative Additions (Backlog)

- **"salesmee Assist"** — floating AI assistant chatbot
- **Achievement badges** — "First Connect", "10 Orders", "5-Star Service"
- **Sound design** — optional notification sounds
- **"salesmee Streak"** — daily active usage counter
- **Conversation themes** — per-business chat color themes
