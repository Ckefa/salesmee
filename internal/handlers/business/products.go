package business

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"salesmee/internal/config"
	"salesmee/internal/data"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"
	"salesmee/internal/services/images"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GetProducts for business
func (h *ProductHandler) GetProducts(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	baseWhere := "business_id = ?"
	baseArgs := []interface{}{businessID}
	locID := c.Query("location_id")
	if locID != "" {
		baseWhere += " AND (location_id IS NULL OR location_id = ?)"
		baseArgs = append(baseArgs, locID)
	}

	var totalCount int64
	h.dbc(c).Model(&models.Product{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	var products []models.Product
	h.dbc(c).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where(baseWhere, baseArgs...).
		Order("created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&products)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/products_content", gin.H{
			"Business":   currentBusiness,
			"Products":   products,
			"Page":       float64(page),
			"TotalPages": float64(totalPages),
			"TotalCount": totalCount,
			"Countries":  data.Countries,
			"Currencies": data.Currencies,
			"Onboarding": onboardingData(h.db, businessID),
			"Locations":  locations,
			"AuthType":   c.GetString("auth_type"),
			"Role":       c.GetString("role"),
			"QueryLocationID": locID,
			"ActivePage": "products",
			"IsDev":      config.IsDev(),
		})
		return
	}

	c.HTML(http.StatusOK, "products.html", gin.H{
		"Business":      currentBusiness,
		"Products":      products,
		"Page":          float64(page),
		"TotalPages":    float64(totalPages),
		"TotalCount":    totalCount,
		"ActivePage":    "products",
		"Countries":     data.Countries,
		"Currencies":    data.Currencies,
		"Onboarding":    onboardingData(h.db, businessID),
		"Locations":     locations,
		"AuthType":      c.GetString("auth_type"),
		"Role":          c.GetString("role"),
		"QueryLocationID": locID,
		"AssistEnabled": assist.IsEnabled(),
		"IsDev":         config.IsDev(),
	})
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		SKU         string  `json:"sku"`
		Stock       int     `json:"stock"`
		MinStock    int     `json:"min_stock"`
		LocationID  *uint   `json:"location_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := models.Product{
		BusinessID:  businessID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		SKU:         req.SKU,
		Stock:       req.Stock,
		MinStock:    req.MinStock,
		LocationID:  req.LocationID,
		IsActive:    true,
	}

	if err := h.dbc(c).Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbc(c).Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.dbc(c).Where("product_id = ?", productID).Find(&images)

	for _, img := range images {
		relPath := strings.TrimPrefix(img.ImageURL, "/")
		filePath := filepath.Join("web", relPath)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			// Log but continue
		}
	}
	h.dbc(c).Where("product_id = ?", productID).Delete(&models.ProductImage{})
	h.dbc(c).Delete(&product)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ProductHandler) UploadProductImage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	file, header, err := c.Request.FormFile("image")
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

	uploadDir := filepath.Join("web", "static", "uploads", "products")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	tmpName := fmt.Sprintf("product_%d_%d_tmp%s", productID, time.Now().Unix(), ext)
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

	webpName := fmt.Sprintf("product_%d_%d.webp", productID, time.Now().Unix())
	webpPath := filepath.Join(uploadDir, webpName)
	if err := images.Process(tmpPath, webpPath, images.DefaultConfig); err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image"})
		return
	}

	imageURL := fmt.Sprintf("/static/uploads/products/%s", webpName)

	var count int64
	h.dbc(c).Model(&models.ProductImage{}).Where("product_id = ?", productID).Count(&count)

	productImage := models.ProductImage{
		ProductID: uint(productID),
		ImageURL:  imageURL,
		SortOrder: int(count),
	}
	h.dbc(c).Create(&productImage)

	if product.ImageURL == "" {
		h.dbc(c).Model(&product).Update("ImageURL", imageURL)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": imageURL, "image_id": productImage.ID})
}

func (h *ProductHandler) GetProductImages(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.dbc(c).Where("product_id = ?", productID).Order("sort_order ASC").Find(&images)

	if len(images) == 0 && product.ImageURL != "" {
		images = append(images, models.ProductImage{
			ID:        0,
			ProductID: uint(productID),
			ImageURL:  product.ImageURL,
			SortOrder: 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "images": images})
}

func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	imageID, err := strconv.ParseUint(c.Param("image_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var image models.ProductImage
	if err := h.dbc(c).Where("id = ? AND product_id = ?", imageID, productID).First(&image).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	relPath := strings.TrimPrefix(image.ImageURL, "/")
	filePath := filepath.Join("web", relPath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove product image file %s: %v", filePath, err)
	}

	h.dbc(c).Delete(&image)

	if product.ImageURL == image.ImageURL {
		var firstImage models.ProductImage
		if err := h.dbc(c).Where("product_id = ?", productID).Order("sort_order ASC").First(&firstImage).Error; err == nil {
			h.dbc(c).Model(&product).Update("ImageURL", firstImage.ImageURL)
		} else {
			h.dbc(c).Model(&product).Update("ImageURL", "")
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetBusinessProducts as a struct
func (h *ProductHandler) GetBusinessProducts(c *gin.Context) {
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	var products []models.Product
	if err := h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) ShowClientProductsPage(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Client not authenticated"})
		return
	}

	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client.html", gin.H{"error": "Invalid business ID", "IsDev": config.IsDev()})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "client.html", gin.H{"error": "Business not found", "IsDev": config.IsDev()})
		return
	}

	var products []models.Product
	h.dbc(c).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("business_id = ? AND is_active = ?", businessID, true).Order("created_at DESC").Find(&products)

	var client models.Client
	h.dbc(c).First(&client, clientID)

	c.HTML(http.StatusOK, "client_products.html", gin.H{
		"Business": business,
		"Client":   client,
		"Products": products,
	})
}

func (h *ProductHandler) GetClientProductImages(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Client not authenticated"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.dbc(c).Where("id = ? AND is_active = ?", productID, true).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.dbc(c).Where("product_id = ?", productID).Order("sort_order ASC").Find(&images)

	if len(images) == 0 && product.ImageURL != "" {
		images = append(images, models.ProductImage{
			ID:        0,
			ProductID: uint(productID),
			ImageURL:  product.ImageURL,
			SortOrder: 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "images": images})
}

type quickProduct struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
	Stock    int     `json:"stock"`
	SKU      string  `json:"sku"`
}

func (h *ProductHandler) GetProductsQuickList(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var products []models.Product
	if err := h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Order("name ASC").Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	list := make([]quickProduct, 0, len(products))
	for _, p := range products {
		list = append(list, quickProduct{
			ID:       p.ID,
			Name:     p.Name,
			Price:    p.Price,
			ImageURL: p.ImageURL,
			Stock:    p.Stock,
			SKU:      p.SKU,
		})
	}

	c.JSON(http.StatusOK, list)
}
