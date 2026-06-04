# Threadly Improvement Plan

This document outlines key upgrades for the Threadly SaaS CRM system, focusing on analytics, UX improvements, RBAC, and architecture refactoring.

---

## 1. Dashboard Statistics + Time Filtering

### Objective
Build a dynamic analytics dashboard with flexible time-based filtering and real-time metric updates.

### Features

#### Date Range Filters
- Last Year 
- This Year (Year-To-Date)
- Last 6 Months
- Last 3 Months
- Last Month
- This Month

#### Dashboard Metrics
- Total Orders / Bookings
- Completed Orders
- Pending Orders
- Confirmed Orders
- Cancelled Orders

#### Dynamic Behavior
- All metrics must update based on selected date filter
- Queries must be optimized for performance (indexed + paginated where needed)

#### Optional Enhancements
- Comparison indicators:
  - % change vs previous period
  - Trend arrows (↑ ↓ →)

---

## 2. Conversation Progress Tracking (UX/UI Upgrade)

### Objective
Redesign conversation tracking system into a modern, clean, and behavior-driven customer intelligence UI.

### UI Improvements
- Replace circular progress bar with:
  - Segmented progress bar OR modern linear/ring hybrid indicator
- Clean dashboard-friendly minimal design

### Customer Level System

#### Level Calculation Inputs
- Number of completed orders/bookings
- Pending interactions
- Confirmed transactions
- Cancelled order impact

#### Level Output
- Customer levels:
  - Level 1–5 OR
  - Bronze / Silver / Gold / Platinum system

### UI Elements

- Customer level badge
- Progress toward next level
- Activity score visualization

### Insights Panel

- Customer activity summary
- Behavior trend indicator:
  - Active
  - Inactive
  - High Value
- Engagement overview

### Requirements
- Fully responsive UI
- Minimal, modern dashboard design system

---

## 3. Multi-User Business Login + Roles (RBAC)

### Objective
Enable multi-user access per business with strict role-based permissions.

### Authentication Model
- Multi-user per business account
- Business-scoped authentication

### Roles

#### Manager
- Manage users
- View all analytics
- Edit business settings
- Manage orders/bookings

#### Regular User
- View assigned data only
- Limited order and booking handling
- No access to settings

### Permissions System (Recommended Upgrade)
- orders:read
- orders:write
- analytics:view
- customers:manage
- settings:manage

### Additional Features
- User invitation system (optional)
- Role assignment per business
- Secure session handling per business context

---

## 4. General UX / System Improvements

### Objective
Improve overall system usability, performance, and visual consistency.

### UI/UX Improvements
- Standardize UI components:
  - Buttons
  - Cards
  - Charts
  - Modals
- Improve design consistency across dashboard
- Enhance visual hierarchy (colors, spacing, typography)

### Performance Improvements
- Optimize analytics queries
- Add DB indexing for frequently queried fields
- Implement pagination for large datasets

### UX Enhancements
- Add loading states
- Add skeleton loaders for:
  - Dashboard stats
  - Progress tracking
  - Analytics panels

### Responsiveness
- Ensure full mobile responsiveness across all modules

---

## 5. Architecture Refactor: Hexagonal Architecture (Ports & Adapters)

### Objective
Refactor the codebase into a clean hexagonal architecture to improve scalability, testability, and maintainability.

---

### Target Architecture
      +----------------------+
        |     HTTP / HTMX      |
        |   (Delivery Layer)   |
        +----------+-----------+
                   |
                   v
        +----------------------+
        |   Application Core   |
        |   (Use Cases Layer)  |
        +----------+-----------+
                   |
     +-------------+--------------+
     |                            |
     v                            v

+------------------+ +----------------------+
| Domain Layer | | Infrastructure |
| (Business Logic) | | (DB, Cache, APIs) |
+------------------+ +----------------------+



---

### Refactor Goals

#### 1. Domain Layer (Pure Logic)
- Entities:
  - User
  - Business
  - Customer
  - Order
  - Conversation
- Must contain NO framework dependencies

---

#### 2. Application Layer (Use Cases)
- Business workflows:
  - CreateOrder
  - UpdateConversation
  - GenerateAnalytics
  - AssignUserRole

- Orchestrates domain logic

---

#### 3. Infrastructure Layer
- GORM database implementation
- External services (email, notifications)
- Cache systems (optional)

---

#### 4. Delivery Layer
- HTTP handlers (Echo / Gin)
- HTMX endpoints
- WebSocket handlers

---

### Key Principles
- No business logic in handlers
- No database logic in domain layer
- Dependency inversion enforced via interfaces
- All external systems abstracted behind ports

---

### Migration Strategy
- Start by extracting services from handlers
- Introduce interfaces for repositories
- Gradually move logic into use-case layer
- Keep system running during refactor (incremental migration)

---

### Outcome
- Highly scalable SaaS architecture
- Easy testing (mockable layers)
- Clean separation of concerns
- Ready for enterprise-level expansion

---
