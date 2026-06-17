package business

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"salesmee/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type DailyRevenue struct {
	Date           string
	OrdersRevenue  float64
	BookingsRevenue float64
	Total          float64
}

type SalesRow struct {
	Name       string
	Type       string
	Quantity   int64
	Revenue    float64
	Percentage float64
}

type TaxRow struct {
	Month   string
	Revenue float64
}

func (h *ReportHandler) GetReportsPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/reports_page_content", gin.H{
			"Business":   currentBusiness,
			"ActivePage": "reports",
			"ActiveTab":  "revenue",
			"Onboarding": onboardingData(h.db, businessID),
			"AuthType":   c.GetString("auth_type"),
			"Role":       c.GetString("role"),
		})
		return
	}

	c.HTML(http.StatusOK, "reports.html", gin.H{
		"Business":   currentBusiness,
		"ActivePage": "reports",
		"ActiveTab":  "revenue",
		"Onboarding": onboardingData(h.db, businessID),
		"AuthType":   c.GetString("auth_type"),
		"Role":       c.GetString("role"),
	})
}

func (h *ReportHandler) GetRevenueReport(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Status(http.StatusUnauthorized)
		return
	}

	start, end, label := resolveDateRange(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	var allDaily []DailyRevenue
	h.db.Raw(`
		SELECT
		  d.date,
		  COALESCE(o.amount, 0) as orders_revenue,
		  COALESCE(b.amount, 0) as bookings_revenue,
		  COALESCE(o.amount, 0) + COALESCE(b.amount, 0) as total
		FROM (
		  SELECT generate_series(
		    DATE(?),
		    DATE(?),
		    '1 day'::interval
		  )::date AS date
		) d
		LEFT JOIN (
		  SELECT DATE(created_at) as date, SUM(total_amount) as amount
		  FROM orders
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('confirmed', 'fulfilled')
		  GROUP BY DATE(created_at)
		) o ON o.date = d.date
		LEFT JOIN (
		  SELECT DATE(created_at) as date, SUM(total_amount) as amount
		  FROM bookings
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('client_confirmed', 'completed')
		  GROUP BY DATE(created_at)
		) b ON b.date = d.date
		ORDER BY d.date
	`, start, end, businessID, start, end, businessID, start, end).Scan(&allDaily)

	totalItems := len(allDaily)
	totalPages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		totalPages++
	}

	var totalOrdersRev, totalBookingsRev float64
	for _, d := range allDaily {
		totalOrdersRev += d.OrdersRevenue
		totalBookingsRev += d.BookingsRevenue
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if startIdx > totalItems {
		startIdx = totalItems
	}
	if endIdx > totalItems {
		endIdx = totalItems
	}
	var daily []DailyRevenue
	if startIdx < totalItems {
		daily = allDaily[startIdx:endIdx]
	} else {
		daily = []DailyRevenue{}
	}

	c.HTML(http.StatusOK, "dashboard/reports_content", gin.H{
		"ActiveTab":         "revenue",
		"RangeLabel":        label,
		"DailyRevenue":      daily,
		"TotalOrdersRev":    totalOrdersRev,
		"TotalBookingsRev":  totalBookingsRev,
		"TotalRevenue":      totalOrdersRev + totalBookingsRev,
		"StartDate":         start.Format("2006-01-02"),
		"EndDate":           end.Format("2006-01-02"),
		"Page":              float64(page),
		"TotalPages":        float64(totalPages),
		"TotalItems":        totalItems,
	})
}

func (h *ReportHandler) GetSalesReport(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Status(http.StatusUnauthorized)
		return
	}

	start, end, label := resolveDateRange(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	type productSales struct {
		Name     string
		Quantity int64
		Revenue  float64
	}
	var products []productSales
	h.db.Raw(`
		SELECT p.name, SUM(oi.quantity) as quantity, SUM(oi.total_price) as revenue
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		JOIN orders o ON o.id = oi.order_id
		WHERE o.business_id = ? AND o.created_at BETWEEN ? AND ?
		  AND o.status IN ('confirmed', 'fulfilled')
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
	`, businessID, start, end).Scan(&products)

	type serviceSales struct {
		Name     string
		Quantity int64
		Revenue  float64
	}
	var services []serviceSales
	h.db.Raw(`
		SELECT s.name, SUM(bi.quantity) as quantity, SUM(bi.total_price) as revenue
		FROM booking_items bi
		JOIN services s ON s.id = bi.service_id
		JOIN bookings b ON b.id = bi.booking_id
		WHERE b.business_id = ? AND b.created_at BETWEEN ? AND ?
		  AND b.status IN ('client_confirmed', 'completed')
		GROUP BY s.id, s.name
		ORDER BY revenue DESC
	`, businessID, start, end).Scan(&services)

	var totalRevenue float64
	var salesRows []SalesRow
	for _, p := range products {
		totalRevenue += p.Revenue
	}
	for _, s := range services {
		totalRevenue += s.Revenue
	}
	for _, p := range products {
		pct := 0.0
		if totalRevenue > 0 {
			pct = math.Round(p.Revenue/totalRevenue*100*10) / 10
		}
		salesRows = append(salesRows, SalesRow{
			Name: p.Name, Type: "Product", Quantity: p.Quantity,
			Revenue: p.Revenue, Percentage: pct,
		})
	}
	for _, s := range services {
		pct := 0.0
		if totalRevenue > 0 {
			pct = math.Round(s.Revenue/totalRevenue*100*10) / 10
		}
		salesRows = append(salesRows, SalesRow{
			Name: s.Name, Type: "Service", Quantity: s.Quantity,
			Revenue: s.Revenue, Percentage: pct,
		})
	}

	totalItems := len(salesRows)
	totalPages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		totalPages++
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if startIdx > totalItems {
		startIdx = totalItems
	}
	if endIdx > totalItems {
		endIdx = totalItems
	}
	if startIdx < totalItems {
		salesRows = salesRows[startIdx:endIdx]
	} else {
		salesRows = []SalesRow{}
	}

	c.HTML(http.StatusOK, "dashboard/reports_content", gin.H{
		"ActiveTab":    "sales",
		"RangeLabel":   label,
		"SalesRows":    salesRows,
		"TotalRevenue": totalRevenue,
		"StartDate":    start.Format("2006-01-02"),
		"EndDate":      end.Format("2006-01-02"),
		"Page":         float64(page),
		"TotalPages":   float64(totalPages),
		"TotalItems":   totalItems,
	})
}

func (h *ReportHandler) GetClientReport(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Status(http.StatusUnauthorized)
		return
	}

	start, end, label := resolveDateRange(c)

	var newClients []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	h.db.Raw(`
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM clients
		WHERE business_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, businessID, start, end).Scan(&newClients)

	var totalClients int64
	h.db.Model(&models.Client{}).Where("business_id = ?", businessID).Count(&totalClients)

	var newTotal int64
	for _, c := range newClients {
		newTotal += c.Count
	}

	var recentClients []models.Client
	h.db.Where("business_id = ?", businessID).Order("created_at DESC").Limit(20).Find(&recentClients)

	c.HTML(http.StatusOK, "dashboard/reports_content", gin.H{
		"ActiveTab":        "clients",
		"RangeLabel":       label,
		"NewClients":       newClients,
		"NewClientsTotal":  newTotal,
		"TotalClients":     totalClients,
		"RecentClients":    recentClients,
		"StartDate":        start.Format("2006-01-02"),
		"EndDate":          end.Format("2006-01-02"),
	})
}

func (h *ReportHandler) GetTaxReport(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Status(http.StatusUnauthorized)
		return
	}

	start, end, label := resolveDateRange(c)

	type monthRev struct {
		Month   string
		Revenue float64
	}
	var monthly []monthRev
	h.db.Raw(`
		SELECT TO_CHAR(date_trunc('month', created_at), 'YYYY-MM') as month,
		       SUM(total_amount) as revenue
		FROM (
		  SELECT created_at, total_amount FROM orders
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('confirmed', 'fulfilled')
		  UNION ALL
		  SELECT created_at, total_amount FROM bookings
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('client_confirmed', 'completed')
		) combined
		GROUP BY month
		ORDER BY month
	`, businessID, start, end, businessID, start, end).Scan(&monthly)

	var totalRevenue float64
	for _, m := range monthly {
		totalRevenue += m.Revenue
	}

	var totalOrders, totalBookings int64
	h.db.Model(&models.Order{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('confirmed', 'fulfilled')", businessID, start, end).Count(&totalOrders)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND created_at BETWEEN ? AND ? AND status IN ('client_confirmed', 'completed')", businessID, start, end).Count(&totalBookings)

	c.HTML(http.StatusOK, "dashboard/reports_content", gin.H{
		"ActiveTab":      "tax",
		"RangeLabel":     label,
		"MonthlyRevenue": monthly,
		"TotalRevenue":   totalRevenue,
		"TotalOrders":    totalOrders,
		"TotalBookings":  totalBookings,
		"StartDate":      start.Format("2006-01-02"),
		"EndDate":        end.Format("2006-01-02"),
	})
}

func (h *ReportHandler) ExportOrdersCSV(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}

	var orders []models.Order
	h.db.Where("business_id = ?", businessID).Preload("Client").Preload("OrderItems.Product").Order("created_at DESC").Find(&orders)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=orders.csv")
	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"Order #", "Status", "Client Name", "Client Email", "Items", "Total Amount", "Paid Amount", "Created At"})
	for _, o := range orders {
		var itemNames []string
		for _, it := range o.OrderItems {
			itemNames = append(itemNames, fmt.Sprintf("%s x%d", it.Product.Name, it.Quantity))
		}
		wr.Write([]string{
			o.OrderNumber, o.Status, o.Client.Name, o.Client.Email,
			strings.Join(itemNames, "; "),
			fmt.Sprintf("%.2f", o.TotalAmount),
			fmt.Sprintf("%.2f", o.PaidAmount),
			o.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

func (h *ReportHandler) ExportBookingsCSV(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}

	var bookings []models.Booking
	h.db.Where("business_id = ?", businessID).Preload("Client").Preload("BookingItems.Service").Order("created_at DESC").Find(&bookings)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=bookings.csv")
	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"Booking #", "Status", "Client Name", "Client Email", "Service", "Scheduled Date", "Total Amount", "Paid Amount", "Created At"})
	for _, b := range bookings {
		var svcNames []string
		for _, it := range b.BookingItems {
			svcNames = append(svcNames, it.Service.Name)
		}
		wr.Write([]string{
			b.BookingNumber, b.Status, b.Client.Name, b.Client.Email,
			strings.Join(svcNames, "; "),
			b.ScheduledDate.Format("2006-01-02 15:04"),
			fmt.Sprintf("%.2f", b.TotalAmount),
			fmt.Sprintf("%.2f", b.PaidAmount),
			b.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

func (h *ReportHandler) ExportPaymentsCSV(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}

	var payments []models.Payment
	h.db.Where("order_id IN (SELECT id FROM orders WHERE business_id = ?) OR booking_id IN (SELECT id FROM bookings WHERE business_id = ?)", businessID, businessID).
		Preload("Client").Order("created_at DESC").Find(&payments)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=payments.csv")
	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"Payment #", "Type", "Client Name", "Client Email", "Amount", "Method", "Status", "Reference", "Created At"})
	for _, p := range payments {
		refType := "Order"
		if p.BookingID != nil {
			refType = "Booking"
		}
		wr.Write([]string{
			fmt.Sprintf("%d", p.ID), refType, p.Client.Name, p.Client.Email,
			fmt.Sprintf("%.2f", p.Amount), p.Method, p.Status, p.Reference,
			p.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

func (h *ReportHandler) ExportRevenueCSV(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}

	start, end, _ := resolveDateRange(c)

	var daily []DailyRevenue
	h.db.Raw(`
		SELECT
		  d.date,
		  COALESCE(o.amount, 0) as orders_revenue,
		  COALESCE(b.amount, 0) as bookings_revenue,
		  COALESCE(o.amount, 0) + COALESCE(b.amount, 0) as total
		FROM (
		  SELECT generate_series(
		    DATE(?),
		    DATE(?),
		    '1 day'::interval
		  )::date AS date
		) d
		LEFT JOIN (
		  SELECT DATE(created_at) as date, SUM(total_amount) as amount
		  FROM orders
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('confirmed', 'fulfilled')
		  GROUP BY DATE(created_at)
		) o ON o.date = d.date
		LEFT JOIN (
		  SELECT DATE(created_at) as date, SUM(total_amount) as amount
		  FROM bookings
		  WHERE business_id = ? AND created_at BETWEEN ? AND ?
		    AND status IN ('client_confirmed', 'completed')
		  GROUP BY DATE(created_at)
		) b ON b.date = d.date
		ORDER BY d.date
	`, start, end, businessID, start, end, businessID, start, end).Scan(&daily)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=revenue.csv")
	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"Date", "Orders Revenue", "Bookings Revenue", "Total Revenue"})
	for _, d := range daily {
		wr.Write([]string{
			d.Date,
			fmt.Sprintf("%.2f", d.OrdersRevenue),
			fmt.Sprintf("%.2f", d.BookingsRevenue),
			fmt.Sprintf("%.2f", d.Total),
		})
	}
	wr.Flush()
}

func (h *ReportHandler) ExportClientsCSV(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}

	var clients []models.Client
	h.db.Where("business_id = ?", businessID).Order("created_at DESC").Find(&clients)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=clients.csv")
	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"Name", "Email", "Phone", "Status", "Created At"})
	for _, cl := range clients {
		wr.Write([]string{
			cl.Name, cl.Email, cl.Phone, string(cl.Status),
			cl.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

func resolveDateRange(c *gin.Context) (start, end time.Time, label string) {
	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr != "" && endStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", startStr)
		if err == nil {
			end, err = time.Parse("2006-01-02", endStr)
			if err == nil {
				end = end.Add(24*time.Hour - time.Second)
				label = startStr + " - " + endStr
				return
		}
	}
	}
	now := time.Now()
	loc := now.Location()
	switch c.DefaultQuery("range", "this_month") {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = now
		label = "Today"
	case "this_week":
		weekday := now.Weekday()
		start = now.AddDate(0, 0, -int(weekday))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end = now
		label = "This Week"
	default:
		start, end, label = timeRangeBounds(c.DefaultQuery("range", "this_month"))
	}
	return
}
