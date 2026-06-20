package business

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"salesmee/internal/config"
	"salesmee/internal/data"
	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/assist"
	"salesmee/internal/services/images"
	"salesmee/internal/services/subscription"
	"salesmee/internal/services/onboarding"
	"salesmee/internal/ws"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BusinessHandler struct {
	dbProvider
}

func NewBusinessHandler(deps *HandlerDeps) *BusinessHandler {
	return &BusinessHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func dbc(c *gin.Context) *gorm.DB {
	return db.DB.WithContext(c.Request.Context())
}

func bizIDStr(c *gin.Context) string {
	return strconv.Itoa(int(c.GetUint("business_id")))
}

func onboardingData(db *gorm.DB, businessID uint) *onboarding.OnboardingData {
	var business models.Business
	if err := db.First(&business, businessID).Error; err != nil {
		return &onboarding.OnboardingData{Step: 6, TotalSteps: 5, Completed: true}
	}
	data, err := onboarding.DetectStep(db, &business)
	if err != nil {
		return &onboarding.OnboardingData{Step: 6, TotalSteps: 5, Completed: true}
	}
	return data
}

// DashboardData structure
type DashboardData struct {
	Title               string
	Business            models.Business
	ProductCount        int64
	ServiceCount        int64
	PendingOrderCount   int64
	PendingBookingCount int64
	TotalRevenue        float64
	TotalOrders         int64
	TotalBookings       int64
	ActiveClients       int64
	CompletedCount      int64
	PendingCount        int64
	ConfirmedCount      int64
	CancelledCount      int64
	PeriodLabel         string
	RecentOrders        []models.Order
	RecentBookings      []models.Booking
	LowStockProducts    []models.Product
	ActivePage          string
	Countries           []data.Country
	Currencies          []data.Currency
	Onboarding          *onboarding.OnboardingData
	Locations           []models.Location
	QueryLocationID     string
	AuthType            string
	Role                string
	AssistEnabled       bool
	IsSilverPlan        bool
	ContentTemplate     string
}

func (h *BusinessHandler) GetSharePage(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "business_share.html", gin.H{"error": "Business not found", "AuthType": c.GetString("auth_type"), "Role": c.GetString("role")})
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
	h.dbc(c).Model(&models.Client{}).Where("business_id = ?", businessID).Count(&totalClients)

	var totalProducts int64
	h.dbc(c).Model(&models.Product{}).Where("business_id = ?", businessID).Count(&totalProducts)

	data := gin.H{
		"Title":          "Share - " + business.Name,
		"Business":       business,
		"ProfileURL":     fullURL,
		"ConnectURL":     connectURL,
		"QRData":         fullURL,
		"TotalClients":   int(totalClients),
		"TotalProducts":  int(totalProducts),
		"ActivePage":     "share",
		"Onboarding":     onboardingData(h.db, businessID),
		"AuthType":       c.GetString("auth_type"),
		"Role":           c.GetString("role"),
		"AssistEnabled":  assist.IsEnabled(),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard/share_content", data)
		return
	}

	c.HTML(http.StatusOK, "business_share.html", data)
}

func (h *BusinessHandler) GetBizHome(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "business.html", gin.H{
			"Title":    "SalesMee",
			"Error":    "Business not found",
			"AuthType": c.GetString("auth_type"),
			"Role":     c.GetString("role"),
		})
		return
	}

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

	if err := h.dbc(c).Raw(query, businessID).Scan(&clientsWithUnread).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "business.html", gin.H{
			"Title":     "SalesMee",
			"Error":     "Failed to load clients",
			"Business":  business,
			"Countries": data.Countries,
			"Currencies": data.Currencies,
			"AuthType":  c.GetString("auth_type"),
			"Role":      c.GetString("role"),
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

	// Count non-completed orders per client and add to unread badge
	type notCompletedCountResult struct {
		ClientID uint
		Count    int
	}
	var orderNotCompleted []notCompletedCountResult
	h.dbc(c).Model(&models.Order{}).
		Select("client_id, COUNT(*) as count").
		Where("business_id = ? AND status NOT IN ('fulfilled', 'completed', 'cancelled')", businessID).
		Group("client_id").
		Find(&orderNotCompleted)
	orderMap := make(map[uint]int)
	for _, o := range orderNotCompleted {
		orderMap[o.ClientID] = o.Count
	}

	var bookingNotCompleted []notCompletedCountResult
	h.dbc(c).Model(&models.Booking{}).
		Select("client_id, COUNT(*) as count").
		Where("business_id = ? AND status NOT IN ('completed', 'cancelled')", businessID).
		Group("client_id").
		Find(&bookingNotCompleted)
	bookingMap := make(map[uint]int)
	for _, b := range bookingNotCompleted {
		bookingMap[b.ClientID] = b.Count
	}

	for i := range clientsWithUnread {
		clientsWithUnread[i].UnreadCount += orderMap[clientsWithUnread[i].ID] + bookingMap[clientsWithUnread[i].ID]
	}

	// Count non-completed orders and bookings (global)
	var pendingOrderCount int64
	h.dbc(c).Model(&models.Order{}).Where("business_id = ? AND status NOT IN ('fulfilled', 'completed', 'cancelled')", businessID).Count(&pendingOrderCount)

	var pendingBookingCount int64
	h.dbc(c).Model(&models.Booking{}).Where("business_id = ? AND status NOT IN ('completed', 'cancelled')", businessID).Count(&pendingBookingCount)

	totalPending := int(pendingOrderCount + pendingBookingCount)

	onlineCount := 0
	for _, c := range clientsWithUnread {
		if c.IsOnline {
			onlineCount++
		}
	}

	var unreadNotifCount int64
	h.dbc(c).Model(&models.InAppNotification{}).Where("business_id = ? AND is_read = false", businessID).Count(&unreadNotifCount)

	var bizProductCount, bizServiceCount int64
	h.dbc(c).Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&bizProductCount)
	h.dbc(c).Model(&models.Service{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&bizServiceCount)

	token, _ := c.Cookie("token")
	if token == "" {
		token, _ = c.Cookie("team_token")
	}

	c.HTML(http.StatusOK, "business.html", gin.H{
		"Title":               "SalesMee",
		"Business":            business,
		"Clients":             clientsWithUnread,
		"PendingOrderCount":   int(pendingOrderCount),
		"PendingBookingCount": int(pendingBookingCount),
		"TotalPending":        totalPending,
		"UnreadNotifCount":    int(unreadNotifCount),
		"OnlineCount":         onlineCount,
		"Countries":           data.Countries,
		"Currencies":          data.Currencies,
		"Onboarding":          onboardingData(h.db, businessID),
		"AuthType":            c.GetString("auth_type"),
		"Role":                c.GetString("role"),
		"AssistEnabled":       assist.IsEnabled(),
		"ProductCount":        bizProductCount,
		"ServiceCount":        bizServiceCount,
		"AuthToken":           token,
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
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "business_login.html", gin.H{"error": "Business not found"})
		return
	}

	locID := c.Query("location_id")

	// Helper to build location clause
	locArgs := []interface{}{}
	if locID != "" {
		locArgs = append(locArgs, locID)
	}

	// Get counts (global — these are sidebar badge counts, not filtered by location)
	var productCount, serviceCount, pendingOrderCount, pendingBookingCount int64

	h.dbc(c).Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&productCount)
	h.dbc(c).Model(&models.Service{}).Where("business_id = ? AND is_active = ?", businessID, true).Count(&serviceCount)
	h.dbc(c).Model(&models.Order{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingOrderCount)
	h.dbc(c).Model(&models.Booking{}).Where("business_id = ? AND status = ?", businessID, "pending").Count(&pendingBookingCount)

	// Get recent orders with client info
	var recentOrders []models.Order
	roQuery := h.dbc(c).Preload("Client").Where("business_id = ?", businessID)
	if locID != "" {
		roQuery = roQuery.Where("location_id = ?", locID)
	}
	roQuery.Order("created_at DESC").Limit(5).Find(&recentOrders)

	// Get recent bookings with client info
	var recentBookings []models.Booking
	rbQuery := h.dbc(c).Preload("Client").Where("business_id = ?", businessID)
	if locID != "" {
		rbQuery = rbQuery.Where("location_id = ?", locID)
	}
	rbQuery.Order("created_at DESC").Limit(5).Find(&recentBookings)

	// Get low stock products (global — not location-scoped)
	var lowStockProducts []models.Product
	h.dbc(c).Where("business_id = ? AND stock <= min_stock AND is_active = ?", businessID, true).Find(&lowStockProducts)

	// Load locations
	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	data := DashboardData{
		Business:            currentBusiness,
		ProductCount:        productCount,
		ServiceCount:        serviceCount,
		PendingOrderCount:   pendingOrderCount,
		PendingBookingCount: pendingBookingCount,
		RecentOrders:        recentOrders,
		RecentBookings:      recentBookings,
		LowStockProducts:    lowStockProducts,
		Countries:           data.Countries,
		Currencies:          data.Currencies,
		Onboarding:          onboardingData(h.db, businessID),
		Locations:           locations,
		QueryLocationID:     locID,
	}

	data.ActivePage = "dashboard"
	data.AuthType = c.GetString("auth_type")
	data.Role = c.GetString("role")
	data.AssistEnabled = assist.IsEnabled()
	data.IsSilverPlan = subscription.IsSilverPlan(businessID)
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard_content", data)
	} else {
		c.HTML(http.StatusOK, "dashboard.html", data)
	}
}

// Helper function to get or create conversation by client and business ID
func getOrCreateConversation(db *gorm.DB, clientID uint, businessID uint) (*models.Conversation, error) {
	var conversation models.Conversation
	err := db.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error
	if err != nil {
		conversation = models.Conversation{
			ClientID:   clientID,
			BusinessID: businessID,
		}
		if err := db.Create(&conversation).Error; err != nil {
			return nil, fmt.Errorf("failed to create conversation: %v", err)
		}
	}
	return &conversation, nil
}

func (h *BusinessHandler) UpdateBusinessProfile(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	name := c.PostForm("name")
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	country := c.PostForm("country")
	currency := c.PostForm("currency")

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
	if country != "" {
		updates["country"] = country
	}
	if currency != "" {
		updates["currency"] = currency
	}
	if password != "" {
		updates["password"] = services.Hash(password)
	}

	if lp := saveBusinessLogo(businessID, c); lp != "" {
		updates["logo"] = lp
	}

	if len(updates) > 0 {
		if err := h.dbc(c).Model(&business).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func saveBusinessLogo(businessID uint, c *gin.Context) string {
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		return ""
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] || header.Size > 5*1024*1024 {
		return ""
	}

	uploadDir := filepath.Join("web", "static", "uploads", "logos")
	os.MkdirAll(uploadDir, 0755)

	tmpName := fmt.Sprintf("business_%d_%d_tmp%s", businessID, time.Now().Unix(), ext)
	tmpPath := filepath.Join(uploadDir, tmpName)
	dst, err := os.Create(tmpPath)
	if err != nil {
		return ""
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return ""
	}
	dst.Close()

	webpName := fmt.Sprintf("business_%d_%d.webp", businessID, time.Now().Unix())
	webpPath := filepath.Join(uploadDir, webpName)
	if err := images.Process(tmpPath, webpPath, images.LogoConfig); err != nil {
		os.Remove(tmpPath)
		return ""
	}

	return filepath.Join("uploads", "logos", webpName)
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
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
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

	tmpName := fmt.Sprintf("business_%d_%d_tmp%s", businessID, time.Now().Unix(), ext)
	tmpPath := filepath.Join(uploadDir, tmpName)
	dst, err := os.Create(tmpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	dst.Close()

	webpName := fmt.Sprintf("business_%d_%d.webp", businessID, time.Now().Unix())
	webpPath := filepath.Join(uploadDir, webpName)
	if err := images.Process(tmpPath, webpPath, images.LogoConfig); err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image"})
		return
	}

	logoPath := filepath.Join("uploads", "logos", webpName)
	if err := h.dbc(c).Model(&models.Business{}).Where("id = ?", businessID).Update("logo", logoPath).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update business logo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logo":    "/static/" + logoPath,
	})
}


func (h *BusinessHandler) RegenerateSlug(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
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
		if h.dbc(c).Where("slug = ? AND id != ?", slug, businessID).First(&existing).Error != nil {
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

	h.dbc(c).Model(&business).Update("slug", slug)

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

func (h *BusinessHandler) InitiateProfileChange(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	name := c.PostForm("name")
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")
	country := c.PostForm("country")
	currency := c.PostForm("currency")

	if name == "" || username == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, username, and email are required"})
		return
	}

	needsOTP := false

	if password != "" {
		if password != confirmPassword {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
			return
		}
		if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
			return
		}
		needsOTP = true
	}

	if email != business.Email {
		needsOTP = true
	}

	change := &PendingProfileChange{
		BusinessID: businessID,
		Name:       name,
		Username:   username,
		Email:      email,
		Country:    country,
		Currency:   currency,
	}

	if password != "" {
		change.Password = services.Hash(password)
	}

	logoPath := saveBusinessLogo(businessID, c)

	if !needsOTP {
		updates := map[string]interface{}{
			"name":     name,
			"username": username,
			"email":    email,
		}
		if country != "" {
			updates["country"] = country
		}
		if currency != "" {
			updates["currency"] = currency
		}
		if password != "" {
			updates["password"] = change.Password
		}
		if logoPath != "" {
			updates["logo"] = logoPath
		}
		if err := h.dbc(c).Model(&business).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	change.Logo = logoPath

	otpCode := services.GenerateOTP()
	change.OTPCode = otpCode
	change.OTPExpiresAt = time.Now().Add(10 * time.Minute)

	token := saveProfileChange(change)

	go func() {
		if err := services.SendOTPEmail(business.Email, otpCode); err != nil {
			log.Printf("Warning: failed to send profile change OTP email to business %d: %v", businessID, err)
		}
	}()

	if config.IsDev() {
		log.Printf("[DEV] Profile change OTP for business %d: %s", businessID, otpCode)
	}

	c.JSON(http.StatusOK, gin.H{
		"requires_otp": true,
		"token":        token,
	})
}

func (h *BusinessHandler) ConfirmProfileChange(c *gin.Context) {
	businessID := c.GetUint("business_id")
	token := c.PostForm("token")
	otp := c.PostForm("otp")

	if token == "" || otp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and OTP are required"})
		return
	}

	pending, ok := getProfileChange(token)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired change request"})
		return
	}

	if pending.BusinessID != businessID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	if time.Now().After(pending.OTPExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP has expired. Please request a new code."})
		return
	}

	if pending.OTPCode != otp {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification code"})
		return
	}

	updates := map[string]interface{}{
		"name":     pending.Name,
		"username": pending.Username,
		"email":    pending.Email,
	}
	if pending.Country != "" {
		updates["country"] = pending.Country
	}
	if pending.Currency != "" {
		updates["currency"] = pending.Currency
	}
	if pending.Password != "" {
		updates["password"] = pending.Password
	}
	if pending.Logo != "" {
		updates["logo"] = pending.Logo
	}

	if err := h.dbc(c).Model(&models.Business{}).Where("id = ?", businessID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	deleteProfileChange(token)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) ResendProfileOTP(c *gin.Context) {
	businessID := c.GetUint("business_id")
	token := c.PostForm("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	pending, ok := getProfileChange(token)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired change request"})
		return
	}

	if pending.BusinessID != businessID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	otpCode := services.GenerateOTP()
	pending.OTPCode = otpCode
	pending.OTPExpiresAt = time.Now().Add(10 * time.Minute)
	saveProfileChange(pending)

	go func() {
		if err := services.SendOTPEmail(business.Email, otpCode); err != nil {
			log.Printf("Warning: failed to resend profile change OTP email: %v", err)
		}
	}()

	if config.IsDev() {
		log.Printf("[DEV] Profile change OTP resent for business %d: %s", businessID, otpCode)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type DashboardStatsData struct {
	Business       models.Business
	PeriodLabel    string
	TotalOrders    int64
	TotalBookings  int64
	CompletedCount int64
	PendingCount   int64
	ConfirmedCount int64
	CancelledCount int64
	TotalRevenue   float64
	ActiveClients  int64
	Locations      []models.Location
	QueryLocationID string
}

type periodStats struct {
	TotalOrders    int64
	TotalBookings  int64
	CompletedCount int64
	PendingCount   int64
	ConfirmedCount int64
	CancelledCount int64
	OrdersRevenue  float64
	BookingsRevenue float64
	ActiveClients  int64
}

type statusCount struct {
	Status string
	Count  int64
}

// queryPeriodStats consolidates 12+ individual dashboard COUNT/SUM queries into 5 grouped queries.
func queryPeriodStats(db *gorm.DB, businessID uint, startTime, endTime time.Time, locID string, locArgs []interface{}) periodStats {
	var ps periodStats

	locClause := ""
	if locID != "" {
		locClause = " AND location_id = ?"
	}

	// 1. Order status counts — GROUP BY replaces 6 COUNTs
	var orderCounts []statusCount
	orderArgs := []interface{}{businessID, startTime, endTime}
	orderArgs = append(orderArgs, locArgs...)
	db.Raw("SELECT status, COUNT(*) AS count FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ?"+locClause+" GROUP BY status", orderArgs...).Scan(&orderCounts)
	for _, sc := range orderCounts {
		switch sc.Status {
		case "pending":
			ps.PendingCount += sc.Count; ps.TotalOrders += sc.Count
		case "confirmed", "client_confirmed":
			ps.ConfirmedCount += sc.Count; ps.TotalOrders += sc.Count
		case "fulfilled", "completed":
			ps.CompletedCount += sc.Count; ps.TotalOrders += sc.Count
		case "cancelled":
			ps.CancelledCount += sc.Count
		default:
			ps.TotalOrders += sc.Count
		}
	}

	// 2. Booking status counts — GROUP BY replaces 6 COUNTs
	var bookingCounts []statusCount
	bookingArgs := []interface{}{businessID, startTime, endTime}
	bookingArgs = append(bookingArgs, locArgs...)
	db.Raw("SELECT status, COUNT(*) AS count FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ?"+locClause+" GROUP BY status", bookingArgs...).Scan(&bookingCounts)
	for _, sc := range bookingCounts {
		switch sc.Status {
		case "pending":
			ps.PendingCount += sc.Count; ps.TotalBookings += sc.Count
		case "client_confirmed", "confirmed":
			ps.ConfirmedCount += sc.Count; ps.TotalBookings += sc.Count
		case "completed":
			ps.CompletedCount += sc.Count; ps.TotalBookings += sc.Count
		case "cancelled":
			ps.CancelledCount += sc.Count
		default:
			ps.TotalBookings += sc.Count
		}
	}

	// 3. Order revenue
	db.Raw("SELECT COALESCE(SUM(paid_amount), 0) FROM orders WHERE business_id = ? AND created_at BETWEEN ? AND ?"+locClause, orderArgs...).Scan(&ps.OrdersRevenue)

	// 4. Booking revenue
	db.Raw("SELECT COALESCE(SUM(paid_amount), 0) FROM bookings WHERE business_id = ? AND created_at BETWEEN ? AND ?"+locClause, bookingArgs...).Scan(&ps.BookingsRevenue)

	// 5. Active clients (conversations created in period)
	db.Model(&models.Conversation{}).Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).Count(&ps.ActiveClients)

	return ps
}

func timeRangeBounds(r string) (start, end time.Time, label string) {
	now := time.Now()
	loc := now.Location()

	switch r {
	case "last_year":
		y := now.Year() - 1
		start = time.Date(y, 1, 1, 0, 0, 0, 0, loc)
		end = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		label = "Last Year"
	case "this_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		end = now
		label = "Year to Date"
	case "last_6_months":
		start = now.AddDate(0, -6, 0)
		end = now
		label = "Last 6 Months"
	case "last_3_months":
		start = now.AddDate(0, -3, 0)
		end = now
		label = "Last 3 Months"
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, loc)
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		label = "Last Month"
	default: // this_month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end = now
		label = "This Month"
	}
	return
}

func (h *BusinessHandler) GetDashboardStats(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	locID := c.Query("location_id")

	startTime, endTime, label := timeRangeBounds(r)

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	locArgs := []interface{}{}
	if locID != "" {
		locArgs = append(locArgs, locID)
	}

	stats := queryPeriodStats(h.db, businessID, startTime, endTime, locID, locArgs)

	// Load locations
	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	data := DashboardStatsData{
		Business:        currentBusiness,
		PeriodLabel:     label,
		TotalOrders:     stats.TotalOrders,
		TotalBookings:   stats.TotalBookings,
		CompletedCount:  stats.CompletedCount,
		PendingCount:    stats.PendingCount,
		ConfirmedCount:  stats.ConfirmedCount,
		CancelledCount:  stats.CancelledCount,
		TotalRevenue:    stats.OrdersRevenue + stats.BookingsRevenue,
		ActiveClients:   stats.ActiveClients,
		Locations:       locations,
		QueryLocationID: locID,
	}

	c.HTML(http.StatusOK, "dashboard_stats", data)
}

// GetNotifications renders the in-app notification list (HTMX fragment)
func (h *BusinessHandler) GetNotifications(c *gin.Context) {
	businessID := c.GetUint("business_id")

	type notifWithTime struct {
		models.InAppNotification
		TimeAgo string
	}

	var notifs []models.InAppNotification
	h.dbc(c).Where("business_id = ?", businessID).Order("created_at DESC").Limit(20).Find(&notifs)

	var unreadCount int64
	h.dbc(c).Model(&models.InAppNotification{}).Where("business_id = ? AND is_read = false", businessID).Count(&unreadCount)

	now := time.Now()
	enriched := make([]notifWithTime, len(notifs))
	for i, n := range notifs {
		enriched[i] = notifWithTime{InAppNotification: n, TimeAgo: timeAgo(now, n.CreatedAt)}
	}

	c.HTML(http.StatusOK, "notifications_list", gin.H{
		"Notifications": enriched,
		"UnreadCount":   int(unreadCount),
	})
}

func timeAgo(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		dd := int(d.Hours() / 24)
		if dd == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", dd)
	default:
		return t.Format("Jan 2")
	}
}

func broadcastBizPendingCounts(db *gorm.DB, hub *ws.Hub, businessID uint) {
	if hub == nil {
		return
	}
	var orderCount, bookingCount, notifCount int64
	db.Model(&models.Order{}).Where("business_id = ? AND status NOT IN ('fulfilled', 'completed', 'cancelled')", businessID).Count(&orderCount)
	db.Model(&models.Booking{}).Where("business_id = ? AND status NOT IN ('completed', 'cancelled')", businessID).Count(&bookingCount)
	db.Model(&models.InAppNotification{}).Where("business_id = ? AND is_read = false", businessID).Count(&notifCount)
	ws.BroadcastPendingCount(hub, strconv.Itoa(int(businessID)), int(orderCount), int(bookingCount), int(notifCount))
}

// GetNotificationCount returns the unread notification count as JSON
func (h *BusinessHandler) GetNotificationCount(c *gin.Context) {
	businessID := c.GetUint("business_id")

	cacheKey := fmt.Sprintf("notif_count:%d", businessID)
	if cached, ok := h.fcache.Get(cacheKey); ok {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	var count int64
	h.dbc(c).Model(&models.InAppNotification{}).Where("business_id = ? AND is_read = false", businessID).Count(&count)

	resp := fmt.Sprintf(`{"count":%d}`, int(count))
	h.fcache.Set(cacheKey, resp)
	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(resp))
}

// MarkNotificationRead marks a single notification as read
func (h *BusinessHandler) MarkNotificationRead(c *gin.Context) {
	businessID := c.GetUint("business_id")
	id := c.Param("id")

	h.dbc(c).Model(&models.InAppNotification{}).
		Where("id = ? AND business_id = ?", id, businessID).
		Update("is_read", true)

	broadcastBizPendingCounts(h.db, h.hub, businessID)
	h.fcache.Delete(fmt.Sprintf("notif_count:%d", businessID))

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkAllNotificationsRead marks all notifications as read for the business
func (h *BusinessHandler) MarkAllNotificationsRead(c *gin.Context) {
	businessID := c.GetUint("business_id")

	h.dbc(c).Model(&models.InAppNotification{}).
		Where("business_id = ? AND is_read = false", businessID).
		Update("is_read", true)

	broadcastBizPendingCounts(h.db, h.hub, businessID)
	h.fcache.Delete(fmt.Sprintf("notif_count:%d", businessID))

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNotification deletes a single notification
func (h *BusinessHandler) DeleteNotification(c *gin.Context) {
	businessID := c.GetUint("business_id")
	id := c.Param("id")

	h.dbc(c).Where("id = ? AND business_id = ?", id, businessID).Delete(&models.InAppNotification{})

	broadcastBizPendingCounts(h.db, h.hub, businessID)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

