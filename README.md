# salesmee — Chat-Based CRM for Businesses

A comprehensive CRM system connecting businesses with their clients through a real-time chat interface. Built with Go (Gin) and HTMX.

## Tech Stack

- **Backend:** Go 1.x, Gin web framework, GORM (ORM)
- **Database:** PostgreSQL (production) / SQLite (development)
- **Frontend:** Tailwind CSS v3, HTMX, vanilla JS
- **Templating:** Go `html/template` with `{{define}}`/`{{template}}` blocks
- **Auth:** JWT tokens (cookie + bearer), OTP for clients, OAuth (Google, Facebook)
- **Payments:** Stripe, Paddle (subscriptions); cash/card/bank/mobile (in-person)
- **AI:** Groq API (Llama 3.1 8B) for salesmee Assist chatbot

## Features

### Business Dashboard
- Analytics dashboard with time-range filtering (This Month, Last 3M, YTD, etc.)
- Revenue tracking, order/booking stats, active client metrics, low-stock alerts
- Walk-in counter: Quick Order and Quick Booking modals with 3-step wizards
- Onboarding wizard: 5-step guided setup (Welcome → Products → Profile → Share → Test Order)
- Reports dashboard with revenue, sales, client, and tax reports + CSV exports

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
- Business hours validation, buffer time, max bookings per slot
- Same 4-step status visualization and receipt feature

### Client Communication
- Real-time chat with polling (5s interval)
- Inline order/booking cards within chat with action buttons
- Per-client unread badges (messages + pending orders/bookings)
- Conversation progress tracking (7-stage funnel)
- Customer insights with Bronze→Diamond tier system
- Action system with enhanced progress tracking

### Team Management
- Team member CRUD with invite tokens and email acceptance
- Role-based permissions (owner/manager/staff)
- Per-team-member location assignment
- Commission tracking on orders and bookings
- Separate team login flow

### Multi-Location Support
- CRUD for physical business locations (address, phone, timezone, map coordinates)
- Per-location filtering on products, services, orders, bookings, and analytics
- Location selector on dashboard overview

### Payment System
- Polymorphic payments (link to Order or Booking)
- Methods: cash, card, bank_transfer, mobile_money
- Payment method CRUD with JSONB details
- Confirm/reject pending payments
- Public payment instructions per business
- Client-submitted payment claims with business approval

### Subscription & Billing
- 3 plans: Silver, Gold, Diamond
- Stripe + Paddle integrations
- Monthly/yearly billing, trial period
- Billing portal, plan change/cancel
- Feature gating and resource limits per plan
- Webhook handling for subscription lifecycle events

### AI Assistant (salesmee Assist)
- Floating AI chatbot with wand-magic-sparkles theme
- Business context: suggests product recommendations, drafts replies, platform help
- Client context: helps draft messages, explains ordering/booking, platform navigation
- Powered by Groq API (Llama 3.1 8B)
- Ephemeral conversation history in localStorage

### Client Portal
- Client dashboard with connected business list and unread badges
- Real-time chat with businesses
- Place orders and book services
- View and manage order/booking history
- Submit payment claims
- Leave reviews for completed orders/bookings
- Discover and connect to new businesses
- Google & Facebook OAuth login

### Public Profile & Client Connect
- `/b/:slug` — public business profile with products, services, connect CTA
- `/api/connect/:slug` — OTP-based client self-registration flow
- Share page with QR code, social sharing, profile preview

### Notification System
- In-app notification log with read/unread tracking
- Email notification sending with background scheduler
- Configurable notification preferences (booking reminders, status changes, payment reminders, abandoned cart, re-engagement)

### Security
- JWT-based auth for businesses (owner) and team members
- JWT-based auth for clients
- CSRF protection on all state-changing requests
- Rate limiting on auth endpoints (5/s) and API endpoints (60/s)
- Role-based permission middleware
- OTP-verified profile changes with 15-minute expiry
- Rate-limited OTP sending

### Theme
- Dark mode with CSS variables + localStorage persistence
- Responsive design with mobile bottom nav
- Glass navbar effect with sticky positioning

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
cmd/
  server/              — Entry point (migrations, seed data, routes, server start)
internal/
  handlers/
    business/          — Dashboard handlers (20 files: orders, bookings, products, services,
                         payments, analytics, subscription, onboarding, public, hours,
                         reports, reviews, locations, team, team_auth, assist, widgets,
                         notifications, profile_change, business)
    client/            — Client handlers (5 files: auth, dashboard, orders/bookings/cancel,
                         assist, reviews, social auth, discover, connect)
    admin/             — Admin panel handlers
    routes/            — Route registration (public, business, client, admin)
  models/              — GORM models (28 models across 19 files)
  middleware/          — Auth middleware, CSRF, rate limiting, roles, subscription gating
  services/            — Auth, email, onboarding, payment (Stripe/Paddle), subscription,
                         notifier, assist (Groq AI)
  data/                — Country, currency, business type data
  db/                  — Database connection (SQLite/PostgreSQL)
web/
  templates/
    pages/             — Page templates (business/, client/, admin/, legal/)
    components/        — Component templates (chat/, modals/, ui/, cards/, badges/,
                         assist/, onboarding/, client/)
    layouts/           — Base layout template
    partials/          — Landing page partials
  static/
    css/               — Source + built Tailwind CSS
    js/modules/        — JS modules (assist, business/client chat, theme, onboarding,
                         product/service picker, shared utilities)
    js/core/           — Core JS (app, theme)
    uploads/           — File uploads (logos, product images)
```
