package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
)

// GetAllServices returns all services
func GetAllServices(c *gin.Context) {
	var services []models.Service
	if err := database.DB.Preload("Category").Find(&services).Error; err != nil {
		log.Printf("❌ Failed to fetch services: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    services,
	})
}

// CreateService creates a new service
func CreateService(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Description   string  `json:"description"`
		CategoryID    uint    `json:"category_id" binding:"required"`
		Price         float64 `json:"price"`
		ImageURL      string  `json:"image_url"`
		IsActive      bool    `json:"is_active"`
		NameAr        string  `json:"name_ar"`
		DescriptionAr string  `json:"description_ar"`
		BasePrice     float64 `json:"base_price"`
		PriceUnit     string  `json:"price_unit"`
		Guarantee     string  `json:"guarantee"`
		Policies      string  `json:"policies"`
		Duration      int     `json:"duration"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	service := models.Service{
		Name:          req.Name,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		Price:         req.Price,
		ImageURL:      req.ImageURL,
		IsActive:      req.IsActive,
		NameAr:        req.NameAr,
		DescriptionAr: req.DescriptionAr,
		BasePrice:     req.BasePrice,
		PriceUnit:     req.PriceUnit,
		Guarantee:     req.Guarantee,
		Policies:      req.Policies,
		Duration:      req.Duration,
	}

	if err := database.DB.Create(&service).Error; err != nil {
		log.Printf("❌ Failed to create service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	// Preload related data
	database.DB.Preload("Category").First(&service, service.ID)

	log.Printf("✅ Service created: %s (ID: %d)", service.Name, service.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Service created successfully",
		"data":    service,
	})
}

// UpdateService updates an existing service
func UpdateService(c *gin.Context) {
	serviceID := c.Param("id")
	
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Description   string  `json:"description"`
		CategoryID    uint    `json:"category_id" binding:"required"`
		Price         float64 `json:"price"`
		ImageURL      string  `json:"image_url"`
		IsActive      bool    `json:"is_active"`
		NameAr        string  `json:"name_ar"`
		DescriptionAr string  `json:"description_ar"`
		BasePrice     float64 `json:"base_price"`
		PriceUnit     string  `json:"price_unit"`
		Guarantee     string  `json:"guarantee"`
		Policies      string  `json:"policies"`
		Duration      int     `json:"duration"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var service models.Service
	if err := database.DB.First(&service, serviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	service.Name = req.Name
	service.Description = req.Description
	service.CategoryID = req.CategoryID
	service.Price = req.Price
	service.ImageURL = req.ImageURL
	service.IsActive = req.IsActive
	service.NameAr = req.NameAr
	service.DescriptionAr = req.DescriptionAr
	service.BasePrice = req.BasePrice
	service.PriceUnit = req.PriceUnit
	service.Guarantee = req.Guarantee
	service.Policies = req.Policies
	service.Duration = req.Duration

	if err := database.DB.Save(&service).Error; err != nil {
		log.Printf("❌ Failed to update service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	// Preload related data
	database.DB.Preload("Category").First(&service, service.ID)

	log.Printf("✅ Service updated: %s (ID: %d)", service.Name, service.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service updated successfully",
		"data":    service,
	})
}

// DeleteService deletes a service
func DeleteService(c *gin.Context) {
	serviceID := c.Param("id")

	var service models.Service
	if err := database.DB.First(&service, serviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	if err := database.DB.Delete(&service).Error; err != nil {
		log.Printf("❌ Failed to delete service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	log.Printf("✅ Service deleted: %s (ID: %d)", service.Name, service.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service deleted successfully",
	})
}

// GetAllServiceOptionsForAdmin returns all service options for admin
func GetAllServiceOptionsForAdmin(c *gin.Context) {
	var options []models.ServiceOption
	if err := database.DB.Preload("Category").Find(&options).Error; err != nil {
		log.Printf("❌ Failed to fetch service options: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service options"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    options,
	})
}

// CreateServiceOptionForAdmin creates a new service option for admin
func CreateServiceOptionForAdmin(c *gin.Context) {
	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Price       float64 `json:"price" binding:"required"`
		Duration    int     `json:"duration" binding:"required"`
		CategoryID  uint    `json:"category_id" binding:"required"`
		ImageURL    string  `json:"image_url"`
		Features    []string `json:"features"`
		IsActive    bool    `json:"is_active"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	option := models.ServiceOption{
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Duration:    req.Duration,
		CategoryID:  req.CategoryID,
		ImageURL:    req.ImageURL,
		Features:    req.Features,
		IsActive:    req.IsActive,
		SortOrder:   req.SortOrder,
	}

	if err := database.DB.Create(&option).Error; err != nil {
		log.Printf("❌ Failed to create service option: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service option"})
		return
	}

	// Preload related data
	database.DB.Preload("Category").First(&option, option.ID)

	log.Printf("✅ Service option created: %s (ID: %d)", option.Title, option.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Service option created successfully",
		"data":    option,
	})
}

// UpdateServiceOptionForAdmin updates an existing service option for admin
func UpdateServiceOptionForAdmin(c *gin.Context) {
	optionID := c.Param("id")
	
	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Price       float64 `json:"price" binding:"required"`
		Duration    int     `json:"duration" binding:"required"`
		CategoryID  uint    `json:"category_id" binding:"required"`
		ImageURL    string  `json:"image_url"`
		Features    []string `json:"features"`
		IsActive    bool    `json:"is_active"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var option models.ServiceOption
	if err := database.DB.First(&option, optionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service option not found"})
		return
	}

	option.Title = req.Title
	option.Description = req.Description
	option.Price = req.Price
	option.Duration = req.Duration
	option.CategoryID = req.CategoryID
	option.ImageURL = req.ImageURL
	option.Features = req.Features
	option.IsActive = req.IsActive
	option.SortOrder = req.SortOrder

	if err := database.DB.Save(&option).Error; err != nil {
		log.Printf("❌ Failed to update service option: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service option"})
		return
	}

	// Preload related data
	database.DB.Preload("Category").First(&option, option.ID)

	log.Printf("✅ Service option updated: %s (ID: %d)", option.Title, option.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service option updated successfully",
		"data":    option,
	})
}

// DeleteServiceOptionForAdmin deletes a service option for admin
func DeleteServiceOptionForAdmin(c *gin.Context) {
	optionID := c.Param("id")

	var option models.ServiceOption
	if err := database.DB.First(&option, optionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service option not found"})
		return
	}

	if err := database.DB.Delete(&option).Error; err != nil {
		log.Printf("❌ Failed to delete service option: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service option"})
		return
	}

	log.Printf("✅ Service option deleted: %s (ID: %d)", option.Title, option.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service option deleted successfully",
	})
}
