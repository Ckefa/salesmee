package business

import (
	"net/http"
	dataPkg "salesmee/internal/data"
	"salesmee/internal/models"

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

func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
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

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/analytics_content", gin.H{
			"Business":          currentBusiness,
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
			"Onboarding":        onboardingData(h.db, businessID),
			"AuthType":          c.GetString("auth_type"),
			"Role":              c.GetString("role"),
			"ActivePage":        "analytics",
		})
		return
	}

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
		"Onboarding":        onboardingData(h.db, businessID),
		"AuthType":          c.GetString("auth_type"),
		"Role":              c.GetString("role"),
	})
}

type analyticsData struct {
	TotalOrders, PendingOrders, ConfirmedOrders, FulfilledOrders, CancelledOrders int
	TotalBookings, PendingBookings, ConfirmedBookings, CompletedBookings, CancelledBookings int
	OrdersRevenue, BookingsRevenue, TotalRevenue float64
	TopProducts []TopProduct
	ActiveClients int
	MonthlyRevenue []MonthlyRevenue
	AverageRating float64
	ReviewCount int
}

func (h *AnalyticsHandler) computeAnalyticsData(businessID uint, rangeKey string) analyticsData {
	startTime, endTime, _ := timeRangeBounds(rangeKey)

	var d analyticsData

	// Order counts by status (1 query instead of 5)
	var orderCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Order{}).
		Select("status, COUNT(*) as count").
		Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).
		Group("status").
		Scan(&orderCounts)
	for _, sc := range orderCounts {
		switch sc.Status {
		case models.OrderPending:
			d.PendingOrders = int(sc.Count)
		case models.OrderConfirmed:
			d.ConfirmedOrders = int(sc.Count)
		case models.OrderFulfilled:
			d.FulfilledOrders = int(sc.Count)
		case models.OrderCancelled:
			d.CancelledOrders = int(sc.Count)
		}
		d.TotalOrders += int(sc.Count)
	}

	// Booking counts by status (1 query instead of 5)
	var bookingCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Booking{}).
		Select("status, COUNT(*) as count").
		Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).
		Group("status").
		Scan(&bookingCounts)
	for _, sc := range bookingCounts {
		switch sc.Status {
		case models.OrderPending:
			d.PendingBookings = int(sc.Count)
		case models.OrderClientConfirmed:
			d.ConfirmedBookings = int(sc.Count)
		case models.OrderCompleted:
			d.CompletedBookings = int(sc.Count)
		case models.OrderCancelled:
			d.CancelledBookings = int(sc.Count)
		}
		d.TotalBookings += int(sc.Count)
	}

	// Revenue sums
	h.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('confirmed', 'fulfilled')", businessID, startTime, endTime).Scan(&d.OrdersRevenue)
	h.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('client_confirmed', 'completed')", businessID, startTime, endTime).Scan(&d.BookingsRevenue)
	d.TotalRevenue = d.OrdersRevenue + d.BookingsRevenue

	// Top products
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

	// Active clients
	var tmp int64
	h.db.Model(&models.Conversation{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).Count(&tmp)
	d.ActiveClients = int(tmp)

	// Review stats (1 query instead of 2)
	var rs struct {
		AvgRating   float64
		ReviewCount int64
	}
	h.db.Model(&models.Review{}).
		Where("business_id = ?", businessID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as review_count").
		Scan(&rs)
	d.AverageRating = rs.AvgRating
	d.ReviewCount = int(rs.ReviewCount)

	// Monthly revenue via SQL date_trunc (eliminates in-memory loading)
	type monthRow struct {
		Month   string
		Revenue float64
	}
	var monthly []monthRow
	h.db.Raw(`
		SELECT TO_CHAR(date_trunc('month', created_at), 'YYYY-MM') as month,
		       SUM(total_amount) as revenue
		FROM (
			SELECT created_at, total_amount FROM orders
			WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('confirmed', 'fulfilled')
			UNION ALL
			SELECT created_at, total_amount FROM bookings
			WHERE business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('client_confirmed', 'completed')
		) combined
		GROUP BY month
		ORDER BY month
	`, businessID, startTime, endTime, businessID, startTime, endTime).Scan(&monthly)

	for _, mr := range monthly {
		d.MonthlyRevenue = append(d.MonthlyRevenue, MonthlyRevenue{Month: mr.Month, Revenue: mr.Revenue})
	}
	if len(d.MonthlyRevenue) > 6 {
		d.MonthlyRevenue = d.MonthlyRevenue[len(d.MonthlyRevenue)-6:]
	}

	return d
}

func (h *AnalyticsHandler) GetAnalyticsStats(c *gin.Context) {
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

	c.HTML(http.StatusOK, "dashboard/analytics_content", gin.H{
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
		"AverageRating":     data.AverageRating,
		"ReviewCount":       data.ReviewCount,
	})
}

func (h *AnalyticsHandler) GetAnalyticsStatsGrid(c *gin.Context) {
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
