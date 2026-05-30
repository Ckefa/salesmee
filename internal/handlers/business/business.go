package business

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"oneflow/internal/models"
	"oneflow/internal/services"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BusinessHandler struct {
	db *gorm.DB
}

func NewBusinessHandler(db *gorm.DB) *BusinessHandler {
	return &BusinessHandler{db: db}
}

// DashboardData structure
type DashboardData struct {
	Business            models.Business
	ProductCount        int64
	ServiceCount        int64
	PendingOrderCount   int64
	PendingBookingCount int64
	TotalRevenue        float64
	TotalOrders         int64
	TotalBookings       int64
	ActiveClients       int64
	RecentOrders        []models.Order
	RecentBookings      []models.Booking
	LowStockProducts    []models.Product
	ActivePage          string
}

func (h *BusinessHandler) GetSharePage(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}

	profileURL := fmt.Sprintf("%s/b/%s", c.Request.Host, business.Slug)
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s", scheme, profileURL)
	connectURL := fmt.Sprintf("%s://%s/api/connect/%s", scheme, c.Request.Host, business.Slug)

	// Count total clients and products/services for share analytics
	var totalClients int64
	h.db.Model(&models.Client{}).Where("business_id = ?", businessID).Count(&totalClients)

	var totalProducts int64
	h.db.Model(&models.Product{}).Where("business_id = ?", businessID).Count(&totalProducts)

	c.HTML(http.StatusOK, "business_share.html", gin.H{
		"Title":          "Share - " + business.Name,
		"Business":       business,
		"ProfileURL":     fullURL,
		"ConnectURL":     connectURL,
		"QRData":         fullURL,
		"TotalClients":   int(totalClients),
		"TotalProducts":  int(totalProducts),
		"ActivePage":     "share",
	})
}

func (h *BusinessHandler) GetBizHome(c *gin.Context) {
	businessID := c.GetUint("business_id")

	// Client with unread count struct
	type ClientWithUnread struct {
		models.Client
		ConversationID uint       `json:"conversation_id"`
		UnreadCount    int        `json:"unread_count"`
		LastMessageAt  *time.Time `json:"last_message_at"`
		LastMessage    *string    `json:"last_message"`
		OnlineStatus   string     `json:"online_status"`
	}

	var clientsWithUnread []ClientWithUnread

	// Query: join clients with their conversations, count unread messages
	query := `
		SELECT 
			clients.*, 
			conversations.id as conversation_id,
			COUNT(CASE WHEN messages.sender = 'client' AND messages.created_at > COALESCE(conversations.last_read_by_business_at, '1970-01-01') THEN 1 END) as unread_count,
			MAX(messages.created_at) as last_message_at,
			(SELECT content FROM messages WHERE conversation_id = conversations.id ORDER BY created_at DESC LIMIT 1) as last_message
		FROM clients 
		JOIN conversations ON conversations.client_id = clients.id AND conversations.business_id = ?
		LEFT JOIN messages ON messages.conversation_id = conversations.id
		GROUP BY clients.id, conversations.id
		ORDER BY unread_count DESC, last_message_at DESC
	`

	if err := h.db.Raw(query, businessID).Scan(&clientsWithUnread).Error; err != nil {
		c.HTML(500, "business.html", gin.H{
			"Title": "OneFlow",
			"Error": "Failed to load clients",
		})
		return
	}

	// Set online status for each client
	for i := range clientsWithUnread {
		if clientsWithUnread[i].IsOnline {
			clientsWithUnread[i].OnlineStatus = "online"
		} else {
			clientsWithUnread[i].OnlineStatus = "offline"
		}
	}

	// Count pending orders and bookings
	var pendingOrderCount int64
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingOrderCount)

	var pendingBookingCount int64
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingBookingCount)

	totalPending := int(pendingOrderCount + pendingBookingCount)

	onlineCount := 0
	for _, c := range clientsWithUnread {
		if c.IsOnline {
			onlineCount++
		}
	}

	var business models.Business
	h.db.First(&business, businessID)

	c.HTML(200, "business.html", gin.H{
		"Title":               "OneFlow",
		"Business":            business,
		"Clients":             clientsWithUnread,
		"PendingOrderCount":   int(pendingOrderCount),
		"PendingBookingCount": int(pendingBookingCount),
		"TotalPending":        totalPending,
		"OnlineCount":         onlineCount,
	})
}

func (h *BusinessHandler) GetDashboard(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "business_login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	// Get user from database
	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "business_login.html", gin.H{"error": "Business not found"})
		return
	}

	// Get counts
	var productCount, serviceCount, pendingOrderCount, pendingBookingCount int64
	var totalRevenue float64
	var totalOrders, totalBookings, activeClients int64

	h.db.Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&productCount)
	h.db.Model(&models.Service{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&serviceCount)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingOrderCount)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingBookingCount)
	h.db.Model(&models.Order{}).Where("business_id = ?", businessID).Count(&totalOrders)
	h.db.Model(&models.Booking{}).Where("business_id = ?", businessID).Count(&totalBookings)
	h.db.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&activeClients)

	// Calculate total revenue from completed orders and bookings
	var ordersRevenue, bookingsRevenue float64
	h.db.Model(&models.Order{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "fulfilled"}).Scan(&ordersRevenue)
	h.db.Model(&models.Booking{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "completed"}).Scan(&bookingsRevenue)
	totalRevenue = ordersRevenue + bookingsRevenue

	// Get recent orders with client info
	var recentOrders []models.Order
	h.db.Preload("Client").Where("business_id = ?", businessID).Order("created_at DESC").Limit(5).Find(&recentOrders)

	// Get recent bookings with client info
	var recentBookings []models.Booking
	h.db.Preload("Client").Where("business_id = ?", businessID).Order("created_at DESC").Limit(5).Find(&recentBookings)

	// Get low stock products
	var lowStockProducts []models.Product
	h.db.Where("business_id = ? AND stock <= min_stock AND is_active = ?", businessID, true).Find(&lowStockProducts)

	data := DashboardData{
		Business:            currentBusiness,
		ProductCount:        productCount,
		ServiceCount:        serviceCount,
		PendingOrderCount:   pendingOrderCount,
		PendingBookingCount: pendingBookingCount,
		TotalRevenue:        totalRevenue,
		TotalOrders:         totalOrders,
		TotalBookings:       totalBookings,
		ActiveClients:       activeClients,
		RecentOrders:        recentOrders,
		RecentBookings:      recentBookings,
		LowStockProducts:    lowStockProducts,
	}

	data.ActivePage = "dashboard"
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard_content", data)
	} else {
		c.HTML(http.StatusOK, "dashboard.html", data)
	}
}

// Helper function to get or create conversation by client and business ID
func (h *BusinessHandler) getOrCreateConversation(clientID uint, businessID uint) (*models.Conversation, error) {
	var conversation models.Conversation
	err := h.db.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error
	if err != nil {
		conversation = models.Conversation{
			ClientID:   clientID,
			BusinessID: businessID,
		}
		if err := h.db.Create(&conversation).Error; err != nil {
			return nil, fmt.Errorf("failed to create conversation: %v", err)
		}
	}
	return &conversation, nil
}

func (h *BusinessHandler) UpdateBusinessProfile(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	name := c.PostForm("name")
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if username != "" {
		updates["username"] = username
	}
	if email != "" {
		updates["email"] = email
	}
	if password != "" {
		updates["password"] = services.Hash(password)
	}

	file, header, err := c.Request.FormFile("logo")
	if err == nil {
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
			if header.Size <= 5*1024*1024 {
				uploadDir := filepath.Join("web", "static", "uploads", "logos")
				os.MkdirAll(uploadDir, 0755)
				filename := fmt.Sprintf("business_%d_%d%s", businessID, time.Now().Unix(), ext)
				dst, err := os.Create(filepath.Join(uploadDir, filename))
				if err == nil {
					defer dst.Close()
					if _, err := io.Copy(dst, file); err == nil {
						logoPath := filepath.Join("uploads", "logos", filename)
						updates["logo"] = logoPath
					}
				}
			}
		}
	}

	if len(updates) > 0 {
		if err := h.db.Model(&business).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) UploadBusinessLogo(c *gin.Context) {
	businessID := c.GetUint("business_id")

	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files (jpg, jpeg, png, gif, webp) are allowed"})
		return
	}

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size must be less than 5MB"})
		return
	}

	uploadDir := filepath.Join("web", "static", "uploads", "logos")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	filename := fmt.Sprintf("business_%d_%d%s", businessID, time.Now().Unix(), ext)
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	logoPath := filepath.Join("uploads", "logos", filename)
	if err := h.db.Model(&models.Business{}).Where("id = ?", businessID).Update("logo", logoPath).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update business logo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logo":    "/static/" + logoPath,
	})
}

func (h *BusinessHandler) GetLogoUploadPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}

	// Get counts
	var productCount, serviceCount, pendingOrderCount, pendingBookingCount int64
	var totalRevenue float64
	var totalOrders, totalBookings, activeClients int64

	h.db.Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&productCount)
	h.db.Model(&models.Service{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&serviceCount)
	h.db.Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingOrderCount)
	h.db.Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingBookingCount)
	h.db.Model(&models.Order{}).Where("business_id = ?", businessID).Count(&totalOrders)
	h.db.Model(&models.Booking{}).Where("business_id = ?", businessID).Count(&totalBookings)
	h.db.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&activeClients)

	var ordersRevenue, bookingsRevenue float64
	h.db.Model(&models.Order{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "fulfilled"}).Scan(&ordersRevenue)
	h.db.Model(&models.Booking{}).Select("COALESCE(SUM(total_amount), 0)").Where("business_id = ? AND status IN ?", businessID, []string{"confirmed", "completed"}).Scan(&bookingsRevenue)
	totalRevenue = ordersRevenue + bookingsRevenue

	var recentOrders []models.Order
	h.db.Preload("Client").Where("business_id = ?", businessID).Order("created_at DESC").Limit(5).Find(&recentOrders)

	var recentBookings []models.Booking
	h.db.Preload("Client").Where("business_id = ?", businessID).Order("created_at DESC").Limit(5).Find(&recentBookings)

	var lowStockProducts []models.Product
	h.db.Where("business_id = ? AND stock <= min_stock AND is_active = ?", businessID, true).Find(&lowStockProducts)

	data := DashboardData{
		Business:            business,
		ProductCount:        productCount,
		ServiceCount:        serviceCount,
		PendingOrderCount:   pendingOrderCount,
		PendingBookingCount: pendingBookingCount,
		TotalRevenue:        totalRevenue,
		TotalOrders:         totalOrders,
		TotalBookings:       totalBookings,
		ActiveClients:       activeClients,
		RecentOrders:        recentOrders,
		RecentBookings:      recentBookings,
		LowStockProducts:    lowStockProducts,
	}

	c.HTML(http.StatusOK, "dashboard.html", data)
}

func (h *BusinessHandler) GetPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")
	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}
	c.HTML(http.StatusOK, "payments.html", gin.H{
		"Business":   business,
		"ActivePage": "payments",
	})
}

func (h *BusinessHandler) RegenerateSlug(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	// Generate new unique slug with random suffix
	base := strings.ToLower(business.Name)
	base = strings.TrimSpace(base)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "&", "and")
	var result []rune
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	base = string(result)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "business"
	}

	// Find unique slug
	slug := base
	counter := 1
	for {
		var existing models.Business
		if h.db.Where("slug = ? AND id != ?", slug, businessID).First(&existing).Error != nil {
			break
		}
		n, _ := rand.Int(rand.Reader, big.NewInt(9000))
		slug = fmt.Sprintf("%s-%d", base, 1000+int(n.Int64()))
		counter++
		if counter > 100 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique slug"})
			return
		}
	}

	h.db.Model(&business).Update("slug", slug)

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s/b/%s", scheme, c.Request.Host, slug)
	connectURL := fmt.Sprintf("%s://%s/api/connect/%s", scheme, c.Request.Host, slug)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"slug":        slug,
		"profileURL":  fullURL,
		"connectURL":  connectURL,
	})
}

