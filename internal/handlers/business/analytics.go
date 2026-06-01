package business

import (
	"net/http"
	"sort"
	"oneflow/internal/data"
	"oneflow/internal/models"

	"github.com/gin-gonic/gin"
)

type TopProduct struct {
	Name    string
	Revenue float64
	Count   int64
}

type MonthlyRevenue struct {
	Month   string
	Revenue float64
}

func (h *BusinessHandler) GetAnalytics(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	var totalOrders, pendingOrders, confirmedOrders, fulfilledOrders, cancelledOrders int64
	h.db.Model(&models.Order{}).Where("business_id = ?", businessID).Count(&totalOrders)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingOrders)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "confirmed").Count(&confirmedOrders)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "fulfilled").Count(&fulfilledOrders)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "cancelled").Count(&cancelledOrders)

	var totalBookings, pendingBookings, confirmedBookings, completedBookings, cancelledBookings int64
	h.db.Model(&models.Booking{}).Where("business_id = ?", businessID).Count(&totalBookings)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingBookings)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "confirmed").Count(&confirmedBookings)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "completed").Count(&completedBookings)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "cancelled").Count(&cancelledBookings)

	var ordersRevenue, bookingsRevenue float64
	h.db.Model(&models.Order{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "fulfilled"}).Scan(&ordersRevenue)
	h.db.Model(&models.Booking{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "completed"}).Scan(&bookingsRevenue)
	totalRevenue := ordersRevenue + bookingsRevenue

	var topProducts []TopProduct
	h.db.Raw(`
		SELECT p.name, SUM(oi.total_price) as revenue, SUM(oi.quantity) as count
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		JOIN orders o ON o.id = oi.order_id
		WHERE o.business_id = ? AND o.status IN ('confirmed', 'fulfilled')
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT 10
	`, businessID).Scan(&topProducts)

	var activeClients int64
	h.db.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&activeClients)

	var orders []models.Order
	h.db.Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "fulfilled"}).Find(&orders)

	var bookings []models.Booking
	h.db.Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "completed"}).Find(&bookings)

	monthMap := make(map[string]float64)
	for _, o := range orders {
		month := o.CreatedAt.Format("2006-01")
		monthMap[month] += o.TotalAmount
	}
	for _, b := range bookings {
		month := b.CreatedAt.Format("2006-01")
		monthMap[month] += b.TotalAmount
	}

	var monthlyRevenue []MonthlyRevenue
	for month, revenue := range monthMap {
		monthlyRevenue = append(monthlyRevenue, MonthlyRevenue{Month: month, Revenue: revenue})
	}
	sort.Slice(monthlyRevenue, func(i, j int) bool {
		return monthlyRevenue[i].Month < monthlyRevenue[j].Month
	})
	if len(monthlyRevenue) > 6 {
		monthlyRevenue = monthlyRevenue[len(monthlyRevenue)-6:]
	}

	c.HTML(http.StatusOK, "analytics.html", gin.H{
		"Business":          currentBusiness,
		"ActivePage":        "analytics",
		"TotalRevenue":      totalRevenue,
		"OrdersRevenue":     ordersRevenue,
		"BookingsRevenue":   bookingsRevenue,
		"TotalOrders":       totalOrders,
		"PendingOrders":     pendingOrders,
		"ConfirmedOrders":   confirmedOrders,
		"FulfilledOrders":   fulfilledOrders,
		"CancelledOrders":   cancelledOrders,
		"TotalBookings":     totalBookings,
		"PendingBookings":   pendingBookings,
		"ConfirmedBookings": confirmedBookings,
		"CompletedBookings": completedBookings,
		"CancelledBookings": cancelledBookings,
		"TopProducts":       topProducts,
		"ActiveClients":     activeClients,
		"MonthlyRevenue":    monthlyRevenue,
		"Countries":         data.Countries,
		"Currencies":        data.Currencies,
	})
}
