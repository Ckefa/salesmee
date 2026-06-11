package assist

import (
	"salesmee/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

func PageDescription(path string) string {
	switch {
	case path == "/business/dashboard":
		return "Dashboard Overview — your business stats, recent orders, bookings, and low stock alerts"
	case path == "/business/analytics":
		return "Analytics Dashboard — revenue trends, sales breakdown, and client growth charts"
	case path == "/business/orders":
		return "Orders Management — view all orders, track status, update, and manage the order lifecycle"
	case path == "/business/bookings":
		return "Bookings Management — service bookings with scheduling, status tracking, and calendar view"
	case path == "/business/products":
		return "Products Catalog — manage your product inventory, pricing, images, and stock levels"
	case path == "/business/services":
		return "Services Catalog — manage your service offerings, pricing, and descriptions"
	case path == "/business/payments":
		return "Payments Ledger — all payment records; confirm or reject client payment claims"
	case path == "/business" || path == "/business/":
		return "Customers — all connected clients with message previews and unread badges"
	case path == "/business/subscription":
		return "Subscription & Billing — current plan, usage limits, and upgrade options"
	case path == "/business/reports":
		return "Reports — revenue reports, sales breakdown, client growth, and CSV exports"
	case path == "/business/team":
		return "Team Management — invite team members, set roles and permissions"
	case path == "/business/locations":
		return "Locations — manage multiple business locations"
	case path == "/business/hours":
		return "Business Hours — set weekly hours, special closures, and availability"
	case strings.HasPrefix(path, "/client/businesses/"):
		return "Conversation with a business — send messages, place orders, book services, and make payments"
	case path == "/client/" || path == "/client":
		return "Customer Portal — your connected businesses and recent conversations"
	case path == "/client/discover":
		return "Discover Businesses — search for and connect with new businesses"
	default:
		return ""
	}
}

var dataKeywords = []string{
	"revenue", "sales", "orders", "bookings", "paid", "pending",
	"completed", "cancelled", "confirmed", "stats", "total",
	"growth", "earnings", "income", "profit", "how many",
	"how much", "data", "dashboard", "analytics", "summary",
	"overview", "count", "number of", "what is", "what are",
}

func isDataQuery(message string) bool {
	lower := strings.ToLower(message)
	for _, kw := range dataKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func parseTimePeriod(message string) (time.Time, time.Time, string) {
	now := time.Now()
	lower := strings.ToLower(message)

	switch {
	case strings.Contains(lower, "this week"):
		wd := now.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		start := now.AddDate(0, 0, -int(wd)+int(time.Monday))
		return start, now, "this week"
	case strings.Contains(lower, "last week"):
		wd := now.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		weekStart := now.AddDate(0, 0, -int(wd)+int(time.Monday))
		start := weekStart.AddDate(0, 0, -7)
		return start, weekStart, "last week"
	case strings.Contains(lower, "this month"):
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now, "this month"
	case strings.Contains(lower, "last month"):
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, end, "last month"
	case strings.Contains(lower, "this year"):
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return start, now, "this year"
	case strings.Contains(lower, "today") || strings.Contains(lower, "this day"):
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, now, "today"
	case strings.Contains(lower, "yesterday"):
		start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, end, "yesterday"
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now, "this month"
	}
}

type dataSnapshot struct {
	periodLabel     string
	orderCount      int64
	completedOrders int64
	pendingOrders   int64
	confirmedOrders int64
	cancelledOrders int64
	orderRevenue    float64
	bookingCount    int64
	completedBookings int64
	pendingBookings int64
	cancelledBookings int64
	bookingRevenue  float64
	totalRevenue    float64
	newClients      int64
	totalCollected  float64
	activeProducts  int64
	lowStockCount   int64
	productCount    int64
	serviceCount    int64
}

func fetchData(db *gorm.DB, businessID uint, page, periodLabel string, start, end time.Time) dataSnapshot {
	var d dataSnapshot
	d.periodLabel = periodLabel

	switch {
	case page == "/business/dashboard" || page == "/business/analytics":
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Count(&d.orderCount)
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "fulfilled").Count(&d.completedOrders)
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "pending").Count(&d.pendingOrders)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Count(&d.bookingCount)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "completed").Count(&d.completedBookings)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "pending").Count(&d.pendingBookings)
		db.Raw("SELECT COALESCE(SUM(paid_amount), 0) FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Scan(&d.orderRevenue)
		db.Raw("SELECT COALESCE(SUM(paid_amount), 0) FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Scan(&d.bookingRevenue)
		d.totalRevenue = d.orderRevenue + d.bookingRevenue
		db.Model(&models.Conversation{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Count(&d.newClients)

	case page == "/business/orders":
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Count(&d.orderCount)
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "fulfilled").Count(&d.completedOrders)
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status IN ?", businessID, start, end, []string{"pending", "client_confirmed"}).Count(&d.pendingOrders)
		db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "cancelled").Count(&d.cancelledOrders)
		db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Scan(&d.orderRevenue)

	case page == "/business/bookings":
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Count(&d.bookingCount)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "completed").Count(&d.completedBookings)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status = ?", businessID, start, end, "cancelled").Count(&d.cancelledBookings)
		db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status IN ?", businessID, start, end, []string{"pending", "client_confirmed"}).Count(&d.pendingBookings)
		db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ?", businessID, start, end).Scan(&d.bookingRevenue)

	case page == "/business/products":
		db.Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&d.activeProducts)
		db.Model(&models.Product{}).Where("business_id = ? AND stock <= min_stock AND is_active = ?", businessID, true).Count(&d.lowStockCount)

	case page == "/business/services":
		db.Model(&models.Service{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&d.serviceCount)

	case page == "/business/payments":
		db.Raw("SELECT COALESCE(SUM(amount), 0) FROM payments WHERE (order_id IN (SELECT id FROM orders WHERE business_id = ?) OR booking_id IN (SELECT id FROM bookings WHERE business_id = ?)) AND created_at BETWEEN ? AND ? AND status = ?", businessID, businessID, start, end, "completed").Scan(&d.totalCollected)
	}

	return d
}

func formatDataContext(d dataSnapshot) string {
	var parts []string

	parts = append(parts, "Data for "+d.periodLabel+":")

	if d.totalRevenue > 0 || d.orderCount > 0 || d.bookingCount > 0 {
		if d.totalRevenue > 0 {
			parts = append(parts, formatFloat("Revenue", d.totalRevenue))
		}
		if d.orderCount > 0 || d.bookingCount > 0 {
			var counts []string
			if d.orderCount > 0 {
				counts = append(counts, formatInt(d.orderCount)+" orders")
			}
			if d.bookingCount > 0 {
				counts = append(counts, formatInt(d.bookingCount)+" bookings")
			}
			parts = append(parts, strings.Join(counts, ", "))
		}
	}

	if d.completedOrders > 0 || d.pendingOrders > 0 || d.confirmedOrders > 0 || d.cancelledOrders > 0 {
		var statuses []string
		if d.completedOrders > 0 {
			statuses = append(statuses, formatInt(d.completedOrders)+" completed")
		}
		if d.confirmedOrders > 0 {
			statuses = append(statuses, formatInt(d.confirmedOrders)+" confirmed")
		}
		if d.pendingOrders > 0 {
			statuses = append(statuses, formatInt(d.pendingOrders)+" pending")
		}
		if d.cancelledOrders > 0 {
			statuses = append(statuses, formatInt(d.cancelledOrders)+" cancelled")
		}
		parts = append(parts, "Orders: "+strings.Join(statuses, ", "))
	}

	if d.completedBookings > 0 || d.pendingBookings > 0 || d.cancelledBookings > 0 {
		var bstatuses []string
		if d.completedBookings > 0 {
			bstatuses = append(bstatuses, formatInt(d.completedBookings)+" completed")
		}
		if d.pendingBookings > 0 {
			bstatuses = append(bstatuses, formatInt(d.pendingBookings)+" pending")
		}
		if d.cancelledBookings > 0 {
			bstatuses = append(bstatuses, formatInt(d.cancelledBookings)+" cancelled")
		}
		parts = append(parts, "Bookings: "+strings.Join(bstatuses, ", "))
	}

	if d.newClients > 0 {
		parts = append(parts, formatInt(d.newClients)+" new clients")
	}

	if d.activeProducts > 0 {
		parts = append(parts, formatInt(d.activeProducts)+" active products")
		if d.lowStockCount > 0 {
			parts = append(parts, formatInt(d.lowStockCount)+" low stock alerts")
		}
	}

	if d.serviceCount > 0 {
		parts = append(parts, formatInt(d.serviceCount)+" active services")
	}

	if d.totalCollected > 0 {
		parts = append(parts, formatFloat("Total collected", d.totalCollected))
	}

	if len(parts) <= 1 {
		return ""
	}

	return strings.Join(parts, " ")
}

func formatFloat(label string, val float64) string {
	return label + ": " + formatAmount(val)
}

func formatAmount(val float64) string {
	s := strings.TrimRight(strings.TrimRight(formatDec(val), "0"), ".")
	return "$" + s
}

func formatDec(val float64) string {
	intPart := int(val)
	decPart := int((val - float64(intPart)) * 100)
	if decPart < 0 {
		decPart = -decPart
	}
	return itoa(intPart) + "." + pad(decPart)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

type DataQuerier interface {
	Query(db *gorm.DB, businessID uint, page, periodLabel string, start, end time.Time) dataSnapshot
}

type defaultQuerier struct{}

func (q defaultQuerier) Query(db *gorm.DB, businessID uint, page, periodLabel string, start, end time.Time) dataSnapshot {
	return fetchData(db, businessID, page, periodLabel, start, end)
}

var querier DataQuerier = defaultQuerier{}

func BuildDataContext(db *gorm.DB, businessID uint, page, message string) string {
	desc := PageDescription(page)
	if desc == "" {
		return ""
	}

	if !isDataQuery(message) {
		return "Current page: " + desc
	}

	start, end, periodLabel := parseTimePeriod(message)
	d := querier.Query(db, businessID, page, periodLabel, start, end)
	dataStr := formatDataContext(d)

	var result string
	if dataStr != "" {
		result = "Current page: " + desc + " " + dataStr
	} else {
		result = "Current page: " + desc
	}

	return result
}
