# salesmee — Chat-Based CRM for Businesses

A comprehensive CRM system connecting businesses with their clients through a real-time chat interface. Built with Go (Gin) and HTMX.

## Tech Stack

- **Backend:** Go 1.x, Gin web framework, GORM (ORM)
- **Database:** PostgreSQL (production) / SQLite (development)
- **Frontend:** Tailwind CSS v3, HTMX, vanilla JS
- **Templating:** Go `html/template` with `{{define}}`/`{{template}}` blocks
- **Auth:** JWT tokens (cookie + bearer), OTP for clients, OAuth (Google, Facebook)
- **Payments:** Stripe, PayPal, Paddle (subscriptions); cash/card/bank/mobile (in-person)

## Features

### Business Dashboard
- Analytics dashboard with time-range filtering (This Month, Last 3M, YTD, etc.)
- Revenue tracking, order/booking stats, active client metrics, low-stock alerts
- Walk-in counter: Quick Order and Quick Booking modals with 3-step wizards
- Onboarding wizard: 5-step guided setup (Welcome → Products → Profile → Share → Test Order)

### Product & Service Management
- Full CRUD for products (with inventory tracking, image gallery, low-stock alerts)
- Full CRUD for services (with duration, negotiable pricing)
- Searchable pickers in order/booking modals

### Order Management
- Lifecycle: draft → pending → client_confirmed/confirmed → paid → fulfilled/completed
- 4-step status visualization (Pending/Confirmed/Paid/Completed)
- Quick mark-as-paid, partial payment support
- Print-optimized receipt page with Print/Save PDF
- Client-side search/filter bar with status dropdown

### Booking & Scheduling
- Lifecycle: pending → client_confirmed → paid → completed
- Duration-based scheduling with date/time picker
- Same 4-step status visualization and receipt feature

### Client Communication
- Real-time chat with polling (5s interval)
- Inline order/booking cards within chat with action buttons
- Per-client unread badges (messages + pending orders/bookings)
- Conversation progress tracking (7-stage funnel)
- Customer insights with Bronze→Diamond tier system

### Payment System
- Polymorphic payments (link to Order or Booking)
- Methods: cash, card, bank_transfer, mobile_money
- Payment method CRUD with JSONB details
- Confirm/reject pending payments
- Public payment instructions per business

### Public Profile & Client Connect
- `/b/:slug` — public business profile with products, services, connect CTA
- `/api/connect/:slug` — OTP-based client self-registration flow
- Share page with QR code, social sharing, profile preview

### Subscription & Billing
- 3 plans: Silver, Gold, Diamond
- Stripe + PayPal + Paddle integrations
- Monthly/yearly billing, trial period
- Billing portal, plan change/cancel

### Theme
- Dark mode with CSS variables + localStorage persistence
- Responsive design with mobile bottom nav

## Build & Run

```sh
npm run css:build        # Build Tailwind CSS
go run ./cmd/server      # Run dev server (SQLite)
go build ./...           # Build all packages
go vet ./...             # Vet all packages
```

Set `ENV=dev` with `DB_PATH` for SQLite, or `DB_HOST` + `DB_*` for PostgreSQL.

## Project Structure

```
internal/
  handlers/
    business/        — Dashboard handlers (orders, bookings, products, services, payments, analytics, subscription, onboarding, public profile)
    client/          — Client auth (OTP, OAuth)
    routes/          — Route registration (public, business, client, admin)
  models/            — GORM models (Business, Client, Order, Booking, Payment, etc.)
  middleware/        — Auth middleware, CSRF, rate limiting
  services/          — Auth, email, onboarding, payment, subscription helpers
  db/                — Database connection (SQLite/PostgreSQL)
web/
  templates/         — HTML templates (pages, components, partials)
  static/
    css/             — Source + built Tailwind CSS
    js/modules/      — JS modules (chat, theme, onboarding, shared)
    uploads/         — File uploads
cmd/
  server/            — Entry point
```
