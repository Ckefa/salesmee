package business

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"oneflow/internal/data"
	"oneflow/internal/models"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GetProducts for business
func (h *BusinessHandler) GetProducts(c *gin.Context) {
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

	var products []models.Product
	h.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("business_id = ?", businessID).Order("created_at DESC").Find(&products)

	c.HTML(http.StatusOK, "products.html", gin.H{
		"Business":   currentBusiness,
		"Products":   products,
		"ActivePage": "products",
		"Countries":  data.Countries,
		"Currencies": data.Currencies,
	})
}

func (h *BusinessHandler) CreateProduct(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product.BusinessID = businessID
	product.IsActive = true

	if err := h.db.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *BusinessHandler) GetProduct(c *gin.Context) {
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
	if err := h.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *BusinessHandler) UpdateProduct(c *gin.Context) {
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
	if err := h.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "product": product})
}

func (h *BusinessHandler) DeleteProduct(c *gin.Context) {
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
	if err := h.db.Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.db.Where("product_id = ?", productID).Find(&images)

	for _, img := range images {
		relPath := strings.TrimPrefix(img.ImageURL, "/")
		filePath := filepath.Join("web", relPath)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			// Log but continue
		}
	}
	h.db.Where("product_id = ?", productID).Delete(&models.ProductImage{})
	h.db.Delete(&product)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) UploadProductImage(c *gin.Context) {
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
	if err := h.db.Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
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

	filename := fmt.Sprintf("product_%d_%d%s", productID, time.Now().Unix(), ext)
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

	imageURL := fmt.Sprintf("/static/uploads/products/%s", filename)

	var count int64
	h.db.Model(&models.ProductImage{}).Where("product_id = ?", productID).Count(&count)

	productImage := models.ProductImage{
		ProductID: uint(productID),
		ImageURL:  imageURL,
		SortOrder: int(count),
	}
	h.db.Create(&productImage)

	if product.ImageURL == "" {
		h.db.Model(&product).Update("ImageURL", imageURL)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": imageURL, "image_id": productImage.ID})
}

func (h *BusinessHandler) GetProductImages(c *gin.Context) {
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
	if err := h.db.Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.db.Where("product_id = ?", productID).Order("sort_order ASC").Find(&images)

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

func (h *BusinessHandler) DeleteProductImage(c *gin.Context) {
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
	if err := h.db.Where("id = ? AND business_id = ?", productID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var image models.ProductImage
	if err := h.db.Where("id = ? AND product_id = ?", imageID, productID).First(&image).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	h.db.Delete(&image)

	if product.ImageURL == image.ImageURL {
		var firstImage models.ProductImage
		if err := h.db.Where("product_id = ?", productID).Order("sort_order ASC").First(&firstImage).Error; err == nil {
			h.db.Model(&product).Update("ImageURL", firstImage.ImageURL)
		} else {
			h.db.Model(&product).Update("ImageURL", "")
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetBusinessProducts as a struct
func (h *BusinessHandler) GetBusinessProducts(c *gin.Context) {
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	var products []models.Product
	if err := h.db.Where("business_id = ? AND is_active = ?", businessID, true).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *BusinessHandler) ShowClientProductsPage(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Client not authenticated"})
		return
	}

	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client.html", gin.H{"error": "Invalid business ID"})
		return
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "client.html", gin.H{"error": "Business not found"})
		return
	}

	var products []models.Product
	h.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("business_id = ? AND is_active = ?", businessID, true).Order("created_at DESC").Find(&products)

	var client models.Client
	h.db.First(&client, clientID)

	c.HTML(http.StatusOK, "client_products.html", gin.H{
		"Business": business,
		"Client":   client,
		"Products": products,
	})
}

func (h *BusinessHandler) GetClientProductImages(c *gin.Context) {
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
	if err := h.db.Where("id = ? AND is_active = ?", productID, true).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var images []models.ProductImage
	h.db.Where("product_id = ?", productID).Order("sort_order ASC").Find(&images)

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
