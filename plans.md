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

## Creative Additions (Backlog)

- **"salesmee Assist"** — floating AI assistant chatbot
- **Achievement badges** — "First Connect", "10 Orders", "5-Star Service"
- **Sound design** — optional notification sounds
- **"salesmee Streak"** — daily active usage counter
- **Conversation themes** — per-business chat color themes
