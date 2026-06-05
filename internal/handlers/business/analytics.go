package business

import (
	"net/http"
	"sort"
	dataPkg "salesmee/internal/data"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
)

var _ = models.Order{}

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

	data := h.computeAnalyticsData(businessID, "this_month")

	c.HTML(http.StatusOK, "analytics.html", gin.H{
		"Business":          currentBusiness,
		"ActivePage":        "analytics",
		"TotalRevenue":      data.TotalRevenue,
		"OrdersRevenue":     data.OrdersRevenue,
		"BookingsRevenue":   data.BookingsRevenue,
		"TotalOrders":       data.TotalOrders,
		"PendingOrders":     data.PendingOrders,
		"ConfirmedOrders":   data.ConfirmedOrders,
		"FulfilledOrders":   data.FulfilledOrders,
		"CancelledOrders":   data.CancelledOrders,
		"TotalBookings":     data.TotalBookings,
		"PendingBookings":   data.PendingBookings,
		"ConfirmedBookings": data.ConfirmedBookings,
		"CompletedBookings": data.CompletedBookings,
		"CancelledBookings": data.CancelledBookings,
		"TopProducts":       data.TopProducts,
		"ActiveClients":     data.ActiveClients,
		"MonthlyRevenue":    data.MonthlyRevenue,
		"Countries":         dataPkg.Countries,
		"Currencies":        dataPkg.Currencies,
		"Onboarding":        h.onboardingData(businessID),
	})
}

type analyticsData struct {
	TotalOrders, PendingOrders, ConfirmedOrders, FulfilledOrders, CancelledOrders int
	TotalBookings, PendingBookings, ConfirmedBookings, CompletedBookings, CancelledBookings int
	OrdersRevenue, BookingsRevenue, TotalRevenue float64
	TopProducts []TopProduct
	ActiveClients int
	MonthlyRevenue []MonthlyRevenue
}

func (h *BusinessHandler) computeAnalyticsData(businessID uint, rangeKey string) analyticsData {
	startTime, endTime, _ := timeRangeBounds(rangeKey)
	timeClause := "business_id = ? AND created_at BETWEEN ? AND ?"

	var d analyticsData
	var tmp int64

	h.db.Model(&models.Order{}).Where(timeClause, businessID, startTime, endTime).Count(&tmp); d.TotalOrders = int(tmp)
	h.db.Model(&models.Order{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "pending").Count(&tmp); d.PendingOrders = int(tmp)
	h.db.Model(&models.Order{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "confirmed").Count(&tmp); d.ConfirmedOrders = int(tmp)
	h.db.Model(&models.Order{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "fulfilled").Count(&tmp); d.FulfilledOrders = int(tmp)
	h.db.Model(&models.Order{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "cancelled").Count(&tmp); d.CancelledOrders = int(tmp)

	h.db.Model(&models.Booking{}).Where(timeClause, businessID, startTime, endTime).Count(&tmp); d.TotalBookings = int(tmp)
	h.db.Model(&models.Booking{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "pending").Count(&tmp); d.PendingBookings = int(tmp)
	h.db.Model(&models.Booking{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "client_confirmed").Count(&tmp); d.ConfirmedBookings = int(tmp)
	h.db.Model(&models.Booking{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "completed").Count(&tmp); d.CompletedBookings = int(tmp)
	h.db.Model(&models.Booking{}).Where(timeClause+" AND status = ?", businessID, startTime, endTime, "cancelled").Count(&tmp); d.CancelledBookings = int(tmp)

	h.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('confirmed', 'fulfilled')", businessID, startTime, endTime).Scan(&d.OrdersRevenue)
	h.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('client_confirmed', 'completed')", businessID, startTime, endTime).Scan(&d.BookingsRevenue)
	d.TotalRevenue = d.OrdersRevenue + d.BookingsRevenue

	h.db.Raw(`
		SELECT p.name, SUM(oi.total_price) as revenue, SUM(oi.quantity) as count
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		JOIN orders o ON o.id = oi.order_id
		WHERE o.business_id = ? AND o.created_at BETWEEN ? AND ? AND o.status IN ('confirmed', 'fulfilled')
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT 10
	`, businessID, startTime, endTime).Scan(&d.TopProducts)

	h.db.Model(&models.Conversation{}).Where(timeClause, businessID, startTime, endTime).Count(&tmp); d.ActiveClients = int(tmp)

	var orders []models.Order
	h.db.Where(timeClause+" AND status IN ?", businessID, startTime, endTime, []string{"confirmed", "fulfilled"}).Find(&orders)

	var bookings []models.Booking
	h.db.Where(timeClause+" AND status IN ?", businessID, startTime, endTime, []string{"client_confirmed", "completed"}).Find(&bookings)

	monthMap := make(map[string]float64)
	for _, o := range orders {
		month := o.CreatedAt.Format("2006-01")
		monthMap[month] += o.TotalAmount
	}
	for _, b := range bookings {
		month := b.CreatedAt.Format("2006-01")
		monthMap[month] += b.TotalAmount
	}

	for month, revenue := range monthMap {
		d.MonthlyRevenue = append(d.MonthlyRevenue, MonthlyRevenue{Month: month, Revenue: revenue})
	}
	sort.Slice(d.MonthlyRevenue, func(i, j int) bool {
		return d.MonthlyRevenue[i].Month < d.MonthlyRevenue[j].Month
	})
	if len(d.MonthlyRevenue) > 6 {
		d.MonthlyRevenue = d.MonthlyRevenue[len(d.MonthlyRevenue)-6:]
	}

	return d
}

func (h *BusinessHandler) GetAnalyticsStats(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	data := h.computeAnalyticsData(businessID, r)

	c.HTML(http.StatusOK, "analytics_content", gin.H{
		"Business":          currentBusiness,
		"ActivePage":        "analytics",
		"TotalRevenue":      data.TotalRevenue,
		"OrdersRevenue":     data.OrdersRevenue,
		"BookingsRevenue":   data.BookingsRevenue,
		"TotalOrders":       data.TotalOrders,
		"PendingOrders":     data.PendingOrders,
		"ConfirmedOrders":   data.ConfirmedOrders,
		"FulfilledOrders":   data.FulfilledOrders,
		"CancelledOrders":   data.CancelledOrders,
		"TotalBookings":     data.TotalBookings,
		"PendingBookings":   data.PendingBookings,
		"ConfirmedBookings": data.ConfirmedBookings,
		"CompletedBookings": data.CompletedBookings,
		"CancelledBookings": data.CancelledBookings,
		"TopProducts":       data.TopProducts,
		"ActiveClients":     data.ActiveClients,
		"MonthlyRevenue":    data.MonthlyRevenue,
	})
}

func (h *BusinessHandler) GetAnalyticsStatsGrid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	data := h.computeAnalyticsData(businessID, r)

	c.HTML(http.StatusOK, "analytics_stats_grid", gin.H{
		"Business":          currentBusiness,
		"TotalRevenue":      data.TotalRevenue,
		"OrdersRevenue":     data.OrdersRevenue,
		"BookingsRevenue":   data.BookingsRevenue,
		"TotalOrders":       data.TotalOrders,
		"PendingOrders":     data.PendingOrders,
		"FulfilledOrders":   data.FulfilledOrders,
		"TotalBookings":     data.TotalBookings,
		"PendingBookings":   data.PendingBookings,
		"CompletedBookings": data.CompletedBookings,
		"ActiveClients":     data.ActiveClients,
	})
}
