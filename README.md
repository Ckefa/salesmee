# salesmee: A Chat-Based CRM System with Business Tools

salesmee is a comprehensive customer relationship management (CRM) system built with Go and Gin, featuring real-time chat capabilities similar to WhatsApp, combined with powerful business management tools.

## 🏗️ Architecture

- **Backend**: Go with Gin framework
- **Frontend**: HTML templates with Tailwind CSS
- **Database**: SQLite (development) / PostgreSQL (production)
- **Authentication**: JWT-based with role-based access
- **Real-time**: WebSocket-like polling for live updates

---

## 🚀 Features of the System

### 🏢 Business Features

#### **Authentication & Security**
- **Secure Login**: Business user authentication with email/password
- **JWT Sessions**: Token-based authentication with automatic renewal
- **Role-based Access**: Business-only routes and permissions
- **Session Management**: Automatic logout and session validation

#### **Dashboard & Analytics**
- **Business Dashboard**: Comprehensive overview with key metrics
- **Revenue Tracking**: Total revenue, order revenue, booking revenue
- **Performance Metrics**: Order counts, booking counts, conversion rates
- **Client Analytics**: Active clients, conversation statistics
- **Visual Charts**: Revenue trends and business performance indicators

#### **Product & Service Management**
- **Product Catalog**: Full CRUD operations for products
- **Service Catalog**: Complete service management system
- **Inventory Tracking**: Stock levels and low-stock alerts
- **Media Upload**: Product images and service photos
- **Pricing Control**: Flexible pricing with discounts and variations

#### **Order Management**
- **Order Processing**: Complete order lifecycle management
- **Status Tracking**: Pending → Confirmed → Fulfilled → Cancelled
- **Order Details**: Line items, quantities, pricing, delivery info
- **Payment Integration**: Payment tracking and partial payment support
- **Order History**: Complete order archive with search and filter

#### **Booking & Scheduling**
- **Service Booking**: Complete booking system for services
- **Calendar Integration**: Date/time scheduling with availability
- **Booking Management**: Status tracking and confirmation system
- **Resource Planning**: Service provider scheduling and capacity
- **Automated Reminders**: Booking confirmations and follow-ups

#### **Client Communication**
- **Real-time Chat**: WhatsApp-like messaging interface
- **Conversation History**: Complete message archive with search
- **Media Sharing**: Image, document, and file sharing in chat
- **Message Types**: Text, images, documents, system notifications
- **Quick Actions**: One-click order creation from chat

#### **Advanced Features**
- **Conversation Progress**: Automated sales funnel tracking
- **Action Management**: Task creation from client messages
- **Quick Actions**: Rapid order/booking creation from chat
- **Payment Requests**: Integrated payment collection system
- **Goal Setting**: Custom business objectives and tracking

---

### 👥 Client Features

#### **Client Authentication**
- **OTP-based Login**: Secure one-time password authentication
- **Email Verification**: Verified client access with email OTP
- **Session Management**: Secure client sessions with tokens
- **Multi-device Support**: Login from multiple devices

#### **Client Dashboard**
- **Personal Profile**: Client information management
- **Order History**: Complete order tracking and status
- **Booking Management**: Service booking overview and calendar
- **Communication Hub**: Centralized messaging interface
- **Business Directory**: Discover and connect with businesses

#### **Client Chat Interface**
- **Rich Messaging**: Text, emojis, media sharing
- **Order Integration**: Direct order placement from chat
- **Booking Integration**: Service booking from conversation
- **Real-time Updates**: Live status updates and notifications
- **Message History**: Searchable conversation archive

#### **Self-Service Features**
- **Order Editing**: Modify order quantities and notes
- **Booking Management**: Reschedule or cancel bookings
- **Payment Processing**: Secure payment integration
- **Profile Updates**: Manage personal information

---

### 🔧 System Features

#### **Database Management**
- **Dual Database Support**: SQLite for development, PostgreSQL for production
- **Auto-migration**: Automatic schema updates and version control
- **Data Integrity**: Foreign key constraints and validation
- **Backup Support**: Database backup and recovery tools

#### **Security & Performance**
- **Input Validation**: Comprehensive data validation and sanitization
- **SQL Injection Protection**: Parameterized queries and ORM safety
- **Rate Limiting**: Request throttling and abuse prevention
- **Performance Optimization**: Efficient queries and connection pooling

#### **Development Tools**
- **Hot Reload**: Automatic server restart during development
- **Environment Configuration**: Flexible environment-based settings
- **Logging System**: Comprehensive error and activity logging
- **Migration Scripts**: Database schema versioning

---

## 📱 User Interface Features

#### **Responsive Design**
- **Mobile-First**: Optimized for mobile devices
- **Cross-Platform**: Works on desktop, tablet, and mobile
- **Modern UI**: Clean, intuitive interface with Tailwind CSS
- **Accessibility**: WCAG compliant design patterns

#### **Interactive Elements**
- **Real-time Chat**: Live messaging with typing indicators
- **Modal Dialogs**: Rich forms for data entry and editing
- **Drag & Drop**: File uploads and media management
- **Auto-complete**: Smart suggestions for products and services

#### **Visual Feedback**
- **Status Indicators**: Real-time order and booking status
- **Progress Tracking**: Visual progress bars and completion states
- **Notification System**: Toast notifications and alerts
- **Loading States**: Skeleton screens and spinners

---

## 🔄 Message Types & Data Flow

### **Message Object System**
- **Unified Messaging**: Text, orders, bookings as unified message objects
- **Chronological Display**: Timeline-based conversation view
- **Rich Content**: Support for media, documents, and structured data
- **Edit Capabilities**: In-line editing of orders and bookings
- **Status Updates**: Real-time status synchronization

### **Order Processing**
- **Dynamic Quantities**: Real-time quantity updates and price recalculation
- **Product Association**: Automatic product lookup and details display
- **Status Workflow**: Automated order status progression
- **Payment Integration**: Multi-payment method support
- **Delivery Tracking**: Address and delivery date management

### **Booking System**
- **Service Scheduling**: Time-slot based booking system
- **Availability Management**: Real-time service provider availability
- **Conflict Detection**: Automatic double-booking prevention
- **Reminder System**: Automated booking confirmations
- **Rescheduling**: Easy date/time modification

---

## 🔐 Authentication & Authorization

### **Multi-Role System**
- **Business Users**: Full business management access
- **Client Users**: Secure client-specific features
- **API Security**: JWT-based API authentication
- **Session Management**: Automatic token refresh and validation

### **Permission Management**
- **Business Permissions**: Full CRUD operations on business data
- **Client Permissions**: Limited to own data and conversations
- **Route Protection**: Middleware-based access control
- **Data Isolation**: Strict data separation by user role

---

## 📊 Data Models & Relationships

### **Core Entities**
- **Business**: Business profile and settings
- **Client**: Customer information and authentication
- **Conversation**: Chat sessions and message history
- **Order**: Product orders with items and payments
- **Booking**: Service appointments with scheduling
- **Product**: Catalog items with inventory
- **Service**: Bookable services with pricing

### **Relationships**
- **Business ↔ Clients**: One-to-many client relationships
- **Business ↔ Orders**: Order tracking per business
- **Client ↔ Conversations**: Communication history
- **Order ↔ OrderItems**: Detailed order line items
- **Booking ↔ Services**: Service appointment details

---

## 🚀 Deployment & Configuration

### **Environment Support**
- **Development**: SQLite database with local file storage
- **Production**: PostgreSQL with connection pooling
- **Docker Ready**: Containerized deployment support
- **Cloud Compatible**: AWS, Google Cloud, Azure deployment

### **Configuration Management**
- **Environment Variables**: Secure configuration management
- **Database Migration**: Automatic schema updates
- **Asset Management**: Static file serving and optimization
- **SSL/TLS Support**: HTTPS and secure communication

---

## Bugs
- ~~Orders and booking are loaded twice in business chat~~ ✅ **FIXED** - Removed duplicate order/booking loading from backend GetMessages function, now only MessageObjs populate frontend

## 📈 Upcoming Features

## Planning
 - order/service => payment tracking
   flow: buzz creates 
         order/service modal appears in client chat 
         client confirms the order/service
         buzz initiate payment via quick action buttons
         the payment invoice modal card will contain order/service id, category order/service, name and price
         after
         client confirms payment the buzz approve payments
        
   
### **Enhanced Communication**
- **File Sharing**: Multi-format file sharing (images, documents, media)
- **Business Products Page**: Detailed product catalog with images
- **Business Services Page**: Comprehensive service listings with descriptions
- **Advanced Search**: Full-text search across products, services, and conversations

### **Business Intelligence**
- **Analytics Dashboard**: Advanced business metrics and insights
- **Customer Analytics**: Detailed customer behavior analysis
- **Revenue Reports**: Comprehensive financial reporting
- **Performance Metrics**: KPI tracking and business health indicators

---

## 🛠️ Technical Stack

### **Backend Technologies**
- **Go**: High-performance backend language
- **Gin Framework**: Lightweight HTTP web framework
- **GORM**: Powerful ORM with database abstraction
- **JWT**: Secure authentication and authorization
- **PostgreSQL**: Production-grade relational database
- **SQLite**: Development-friendly file database

### **Frontend Technologies**
- **HTML5**: Modern semantic markup
- **Tailwind CSS**: Utility-first CSS framework
- **JavaScript**: Interactive client-side functionality
- **HTMX**: Dynamic content updates without page reloads
- **Font Awesome**: Professional icon library

### **Development Tools**
- **Hot Reload**: Live development server
- **Environment Management**: Flexible configuration system
- **Migration Tools**: Database schema versioning
- **Testing Framework**: Comprehensive test coverage

---

## 📋 Installation & Setup

### **Prerequisites**
- Go 1.19+ installed
- PostgreSQL 12+ (for production)
- Git for version control

### **Quick Start**
```bash
# Clone the repository
git clone https://github.com/your-username/salesmee.git
cd salesmee

# Install dependencies
go mod download

# Set up environment (copy .env.example to .env)
cp .env.example .env

# Run development server
go run cmd/server/main.go
```

### **Database Setup**
```bash
# PostgreSQL Setup (Production)
sudo -u postgres createdb salesmee
sudo -u postgres createuser salesmee_user

# Configure environment variables
export DB_HOST=localhost
export DB_USER=salesmee_user
export DB_PASSWORD=your_password
export DB_NAME=salesmee
export DB_PORT=5432
```

---

## 🤝 Contributing & Support

salesmee is an open-source project designed to help businesses manage customer relationships through modern chat interfaces. We welcome contributions and feedback!

### **Code Quality**
- Follow Go best practices and conventions
- Maintain test coverage above 80%
- Use meaningful commit messages
- Document new features and changes

### **Feature Requests**
- Open GitHub issues for new feature ideas
- Submit pull requests for improvements
- Report bugs with detailed reproduction steps
- **Suggest enhancements for user experience

---
## Plans

### 8. Prioritized Implementation Order
**Phase 1 — Foundation (must-haves)**
1. Add `Slug` and `IsPublic` fields to Business model + slug generation on registration
2. Create `/b/{slug}` business profile page (public, no auth needed)
3. Create `/api/connect/{slug}` + OTP flow for self-registration
4. Update client auth to allow creating client records on-the-fly (not just pre-created by business)
5. Business share dashboard page with link + QR code
**Phase 2 — Discovery (nice-to-haves)**
6. Public business directory at `/businesses` with search
7. Categories/tags for filtering
8. Client sidebar search + "Discover" CTA
9. Location-based proximity search
**Phase 3 — Growth (growth-hacking)**
10. Email invites from business dashboard
11. Embeddable website widget
12. Social sharing buttons
13. Referral tracking

## Page-by-Page Improvements
### 1. Landing Page (`index.html`) — High Impact / Medium Effort
**Issues:** 1520-line monolith, no mobile nav, no demo/interactive elements, weak CTA hierarchy.
**Improvements:**
- **Split into header/hero/features/testimonials/pricing/cta/footer partials** — maintainable
- **Sticky nav** with transparent→solid scroll effect (glassmorphism)
- **Hero animation** — chat bubble typing animation (SVG/Lottie) instead of static icon
- **Feature section** — interactive preview cards that show chat mockups on hover (CSS-only transitions)
- **Testimonial carousel** — autoplay with dot nav, pull quotes with business avatars
- **Pricing toggles** — monthly/yearly with animated toggle
- **Scroll-triggered reveal** — fade-in-up on sections using IntersectionObserver (lightweight, no library)
- **Mobile hamburger** — slide-out menu with backdrop blur
- **Floating CTA** — "Start Free Trial" pinned bottom on mobile scroll
**Theme:** Dark teal hero gradient → warm white feature sections. Use the existing float/pulse animations but make them smoother.
---
2. Business Login & Register — Medium Impact / Low Effort
Issues: Plain forms, no visual feedback, no demo account hint, no brand personality.
Improvements:
- Animated background — gentle gradient mesh (teal→indigo) or subtle pattern overlay
- Card glassmorphism — frosted glass effect on the login card
- Password visibility toggle — eye icon in input
- "Remember me" checkbox — styled toggle
- Loading state — button shows spinner + "Signing in..." during POST
- Form validation — real-time email format check, min-length for password, inline error messages below fields not at top
- Register → Auto-login — After registration, auto-login and redirect to dashboard instead of requiring another login
- Demo credentials hint — subtle "Demo: demo@salesmee.com / password" for quick testing
- Business type icons — on register, show a grid of clickable business type cards (automotive, salon, etc.) with icons, not a dropdown
Registration Flow Redesign:
Step 1: [Business Name] [Email]
Step 2: [Business Type] ← icon grid selection
Step 3: [Password] [Confirm Password] + strength meter
Progress bar at top, slide transitions between steps.
---
3. Business Dashboard — High Impact / Medium Effort
Issues: Dense layout, no data visualization, no quick-actions palette, no mobile support.
Improvements:
- Command Palette (Cmd+K) — modal overlay to search clients, navigate pages, create orders/bookings quickly. Like Linear/Slack quick switcher
- Widget-based layout — draggable cards for: Revenue (sparkline), Active Conversations (number + trend), Pending Orders (queue), Recent Activity (feed)
- Mini real-time stats — small animated counters for unread, pending, online clients
- Notification bell — dropdown with recent notifications (new client connected, order placed, booking requested)
- Client list search — filter-as-you-type input at top of sidebar
- Online indicators — green dot next to online clients, with "Online now (3)" summary
- Empty state — when no clients, show a friendly illustration with "Share your business link to get started" + quick link to share page
- Responsive sidebar — collapses to icon-only on medium screens, bottom nav on mobile
Currently the main business.html has no client search — the sidebar lists all clients with no way to filter. This is a major UX gap for businesses with many clients.
---
4. Business Chat — High Impact / High Effort
Issues: Polling-based (5s), no typing indicators, no rich messages, plain input.
Improvements:
- Typing indicator — "Client is typing..." with animated dots (CSS only, triggered by a lightweight endpoint)
- Smart suggestions bar — above input, show quick-action chips: "Send Order", "Book Appointment", "Request Payment" based on conversation context (inferred from business type)
- Message search — inside a conversation, search through message history
- Conversation sidebar — show last message preview, unread count, timestamp relative ("2m ago")
- Message status — Sent ✓, Delivered ✓✓, Read (blue double-check) — like WhatsApp
- Rich message actions — inline confirm/reject buttons on order cards without page reload
- Quick replies — business can save canned responses and insert with / commands
- Split view — right panel shows customer details (order history, booking history, notes) when clicking a customer info button
Currently: The chat_header.html has an "Analytics" button but the features it points to (Quotation, Payment, Goal) show "coming soon" notifications. Either implement them or remove the buttons.
---
5. Client Dashboard — Medium Impact / Low Effort
Issues: Basic list, no search, no visual differentiation between connected businesses.
Improvements:
- Business cards — richer cards with business type icon overlay, last message preview, time ago
- Search/filter — filter connected businesses by name or type
- Pin favorites — star icon to pin frequently used businesses to top
- Category badges — colored badges for business types (e.g., blue for dental, green for fitness)
- Empty state — "You haven't connected to any businesses yet" with illustration and "Find Businesses" CTA
- Business online status — green dot if the business has active sessions
---
6. Client Discover (client_discover.html) — Medium Impact / Low Effort
Issues: Plain list, no categories, no "already connected" visual, no business details.
Improvements:
- Category filter chips — horizontal scrollable chips at top: All, Automotive, Salon, Dental, etc. with icons
- Already connected badge — green "Connected" badge on businesses the client already has a conversation with, with "Open Chat" button instead of "Connect"
- Business cards redesign — card with subtle hover elevation, business type icon, star rating placeholder, location if available
- Infinite scroll — load more as user scrolls instead of showing everything at once
- Search with debounce — already implemented (300ms), but add search suggestions/history
- Results count — "Showing X of Y businesses"
---
7. Business Share Page — Low Impact / Low Effort
Issues: Already decent, but could be more engaging.
Improvements:
- Share analytics preview — "Your link has been viewed X times, Y clients connected"
- Social share buttons — WhatsApp, Twitter, Email share links (using the profile URL)
- QR download fix — current download uses canvas from cross-origin image which may fail; use fetch → blob → download instead
- Copy success animation — checkmark + "Copied!" toast instead of just changing icon
- Preview card — show a live preview of what the public profile looks like (iframe or mockup)
---
8. Client Login / OTP Pages — Medium Impact / Low Effort
Issues: OTP code is shown in UI (debug), no resend timer, no visual feedback.
Improvements:
- Remove OTP display — security risk, even in dev
- Resend timer — "Resend code in 0:30" countdown with clickable state after expiry
- Auto-submit OTP — as soon as 6 digits entered, auto-submit (already keyboard-friendly)
- Email validation — real-time email format check before allowing "Get OTP" click
- Animated transition — smooth slide from email entry to OTP entry
- Loading state — OTP input shows spinner while verifying
---
9. Public Profile (public_profile.html) — Medium Impact / Medium Effort
Issues: Sparse, no social proof, no booking widget.
Improvements:
- Online indicator — Show if the business is active/accepting clients
- Business hours — display hours if available
- Quick preview cards — show top 3 services with price, or all products
- Direct connect card — always-visible CTA card that follows on scroll
- Review/Rating placeholder — structure for future reviews
- Share buttons — share this business on social media
---
Global System Improvements
Design System (Medium Effort, High Impact)
- Create a tailwind.config.js with custom tokens (brand colors won't work via CDN — need proper config)
- Or use CSS custom properties in a single styles.css file to define the design tokens
- Build reusable component classes with @apply directives
Animation System (Low Effort, High Impact)
- Page transitions — fade between pages using CSS view transitions (Chrome 111+)
- List animations — staggered fade-in for lists (clients, messages)
- Micro-interactions — button press (scale 0.97), card hover (slight elevation), smooth sidebar collapse
- Toast redesign — slide-in from top-right with progress bar for auto-dismiss, stackable
Loading States (Low Effort, High Impact)
- Skeleton screens — for chat loading, dashboard stats, client lists instead of full-page spinners
- HTMX indicators — use htmx-indicator class to show spinners on buttons during requests (already available but not used)
- Progress bar — thin top-of-page progress bar for navigation (like YouTube)
Empty States (Low Effort, Medium Impact)
- Every list view needs a thoughtful empty state: illustration + message + CTA
- Examples: No clients → "Share your link", No messages → "Send your first message", No orders → "Create an order"
Responsive Design (Medium Effort, High Impact)
- Current layouts break below ~1024px
- Mobile approach: Bottom nav bar for business/client, slide-out drawers for sidebar, full-width chat
- Breakpoint targets: desktop (1280+), tablet (768-1279), mobile (<768)
---
Creative Additions for User Retention
1. "salesmee Assist" — Floating AI Assistant
A small floating action button (bottom-right) that opens a chat-like assistant panel. Can help with:
- "How do I create an order?"
- "Show me my unread conversations"
- "What's my revenue this week?"
- Powered by simple keyword matching or LLM integration
2. Gamification / Progress
- Business onboarding checklist — "Complete your profile → Share your link → Get your first client" with progress bar
- Achievement badges — "First Connect", "10 Orders Processed", "5-Star Service" — shown in dashboard
3. Sound Design
- Optional notification sounds for new messages (like Slack)
- Toggle in settings
4. "salesmee Streak" — Daily Active Usage
- Shows a streak counter for daily engagement
- "You've been active 7 days in a row! 🎉"
5. Conversation Themes
- Clients can pick a chat theme color (pre-set palettes) per business
- Business can set brand colors that clients see
---
Priority Recommendations (Build Order)
Priority	Task
P0	Loading states + HTMX indicators
P0	Empty states for all lists
P0	Form validation + feedback
P0	Client/business list search
P1	Design system (tailwind config or CSS tokens)
P1	Dark mode toggle
P1	Toast redesign (stackable, animated)
P1	Responsive sidebar/collapse
P1	Auto-login after registration
P1	OTP UX (auto-submit, resend timer)
P1	Register multi-step with icons
P2	Command palette (Cmd+K)
P2	Conversation search
P2	Landing page modularization
P2	Discover page category filters
P2	Share page analytics + social
P2	Skeleton screens
P2	Notification bell dropdown
P3	Typing indicators
P3	Message reactions
P3	Drag-and-drop widgets
P3	Sound notifications
P3	AI assistant
P3	PWA / offline
---

*salesmee: Transforming customer relationships through intelligent communication and management tools.*
