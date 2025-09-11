package routes

import (
	"log"
	"net/http"
	"strconv"

	"repair-service-server/database"
	"repair-service-server/middleware"
	"repair-service-server/models"

	"github.com/gin-gonic/gin"
)

// RegisterServiceRoutes registers all service-related routes
func RegisterServiceRoutes(router *gin.RouterGroup) {
	// Public routes
	router.GET("", getAllServicesUpdated)
	router.GET("/:id", getService)
	router.GET("/category/:category", getServicesByCategory)
	router.POST("/seed", seedServicesPublic) // Public seed endpoint

	// Protected routes (admin only)
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware())
	{
		admin.POST("", createService)
		admin.PUT("/:id", updateService)
		admin.DELETE("/:id", deleteService)
	}
}

// getAllServicesUpdated returns all active services with all fields
func getAllServicesUpdated(c *gin.Context) {
	var services []models.Service
	result := database.DB.Where("is_active = ?", true).Preload("Category").Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	// Debug logging
	log.Printf("🔍 Found %d services in database", len(services))
	for i, service := range services {
		log.Printf("Service %d: ID=%d, Name=%s, CategoryID=%d, CategoryName=%s, ImageURL=%s", 
			i+1, service.ID, service.Name, service.CategoryID, service.Category.Name, service.ImageURL)
	}

	var responses []models.ServiceResponse
	for _, service := range services {
		responses = append(responses, models.ServiceResponse{
			ID:            service.ID,
			CategoryID:    service.CategoryID,
			Category:      service.Category,
			Name:          service.Name,
			Description:   service.Description,
			Price:         service.Price,
			ImageURL:      service.ImageURL,
			Duration:      service.Duration,
			IsActive:      service.IsActive,
			CreatedAt:     service.CreatedAt,
			NameAr:        service.NameAr,
			DescriptionAr: service.DescriptionAr,
			BasePrice:     service.BasePrice,
			PriceUnit:     service.PriceUnit,
			Guarantee:     service.Guarantee,
			Policies:      service.Policies,
		})
	}

	c.JSON(http.StatusOK, gin.H{"services": responses})
}

// getService returns a specific service by ID
func getService(c *gin.Context) {
	id := c.Param("id")
	serviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var service models.Service
	result := database.DB.Preload("Category").First(&service, serviceID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	response := models.ServiceResponse{
		ID:            service.ID,
		CategoryID:    service.CategoryID,
		Category:      service.Category,
		Name:          service.Name,
		Description:   service.Description,
		Price:         service.Price,
		ImageURL:      service.ImageURL,
		Duration:      service.Duration,
		IsActive:      service.IsActive,
		CreatedAt:     service.CreatedAt,
		NameAr:        service.NameAr,
		DescriptionAr: service.DescriptionAr,
		BasePrice:     service.BasePrice,
		PriceUnit:     service.PriceUnit,
		Guarantee:     service.Guarantee,
		Policies:      service.Policies,
	}

	c.JSON(http.StatusOK, gin.H{"service": response})
}

// getServicesByCategory returns services filtered by category
func getServicesByCategory(c *gin.Context) {
	categoryID := c.Param("category")
	categoryIDUint, err := strconv.ParseUint(categoryID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}
	
	var services []models.Service
	result := database.DB.Where("category_id = ? AND is_active = ?", categoryIDUint, true).Preload("Category").Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	var responses []models.ServiceResponse
	for _, service := range services {
		responses = append(responses, models.ServiceResponse{
			ID:          service.ID,
			CategoryID:  service.CategoryID,
			Category:    service.Category,
			Name:        service.Name,
			Description: service.Description,
			Price:       service.Price,
			Duration:    service.Duration,
			IsActive:    service.IsActive,
			CreatedAt:   service.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"services": responses})
}

// createService creates a new service (admin only)
func createService(c *gin.Context) {
	var request models.ServiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := models.Service{
		CategoryID:  request.CategoryID,
		Name:        request.Name,
		Description: request.Description,
		Price:       request.Price,
		Duration:    request.Duration,
		IsActive:    true,
	}

	result := database.DB.Create(&service)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Service created successfully", "service_id": service.ID})
}

// updateService updates an existing service (admin only)
func updateService(c *gin.Context) {
	id := c.Param("id")
	serviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var request models.ServiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var service models.Service
	result := database.DB.First(&service, serviceID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Update fields
	service.CategoryID = request.CategoryID
	service.Name = request.Name
	service.Description = request.Description
	service.Price = request.Price
	service.Duration = request.Duration

	database.DB.Save(&service)
	c.JSON(http.StatusOK, gin.H{"message": "Service updated successfully"})
}

// deleteService deletes a service (admin only)
func deleteService(c *gin.Context) {
	id := c.Param("id")
	serviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var service models.Service
	result := database.DB.First(&service, serviceID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Soft delete
	database.DB.Delete(&service)
	c.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}

// seedServicesPublic seeds the database with initial services (public endpoint)
func seedServicesPublic(c *gin.Context) {
	// Check if services already exist
	var count int64
	database.DB.Model(&models.Service{}).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Services already seeded"})
		return
	}

	// Get category IDs first
	var categories []models.ServiceCategory
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	// Create a map of category names to IDs
	categoryMap := make(map[string]uint)
	for _, cat := range categories {
		categoryMap[cat.Name] = cat.ID
	}

	services := []models.Service{
		{
			CategoryID:  categoryMap["Plomberie"],
			Name:        "Réparation de fuites",
			Description: "Services de plomberie professionnels incluant réparation de fuites, installation de robinets, réparation de chauffe-eau et maintenance des systèmes d'égout.",
			Price:       1500.0,
			Duration:   180, // 3 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Électricité"],
			Name:        "Installation électrique",
			Description: "Services électriques complets : installation électrique, réparation de panneaux, éclairage LED, sécurité électrique et maintenance préventive.",
			Price:       2000.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Peinture"],
			Name:        "Peinture intérieure",
			Description: "Services de peinture intérieure et extérieure, préparation des surfaces, finitions décoratives et rénovation complète des murs et plafonds.",
			Price:       800.0,
			Duration:   1440, // 24 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Climatisation"],
			Name:        "Installation climatiseur",
			Description: "Installation, réparation et maintenance de systèmes de climatisation et chauffage, nettoyage des filtres et optimisation énergétique.",
			Price:       3000.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Menuiserie & Serrurerie"],
			Name:        "Réparation de portes",
			Description: "Fabrication et réparation de meubles sur mesure, portes, fenêtres, escaliers et aménagements intérieurs en bois de qualité.",
			Price:       2500.0,
			Duration:   7200, // 5 days in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Nettoyage à la demande"],
			Name:        "Nettoyage complet",
			Description: "Services de nettoyage professionnel : nettoyage résidentiel, commercial, après rénovation et entretien régulier des locaux.",
			Price:       500.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Chauffe-eau"],
			Name:        "Installation chauffe-eau",
			Description: "Installation et réparation de chauffe-eau et systèmes solaires thermiques.",
			Price:       1800.0,
			Duration:   120, // 2 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Appareils électroménagers"],
			Name:        "Réparation frigo",
			Description: "Réparation de frigos, machines à laver et autres appareils électroménagers.",
			Price:       1200.0,
			Duration:   90, // 1.5 hours in minutes
			IsActive:   true,
		},
	}

	successCount := 0
	for _, service := range services {
		// Skip if category not found
		if service.CategoryID == 0 {
			log.Printf("Warning: Category not found for service %s", service.Name)
			continue
		}

		result := database.DB.Create(&service)
		if result.Error != nil {
			log.Printf("Error seeding service %s: %v", service.Name, result.Error)
		} else {
			log.Printf("✅ Seeded service: %s", service.Name)
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Services seeded successfully", "count": successCount})
}

// seedServices seeds the database with initial services (admin only)
func seedServices(c *gin.Context) {
	// Check if services already exist
	var count int64
	database.DB.Model(&models.Service{}).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Services already seeded"})
		return
	}

	// Get category IDs first
	var categories []models.ServiceCategory
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	// Create a map of category names to IDs
	categoryMap := make(map[string]uint)
	for _, cat := range categories {
		categoryMap[cat.Name] = cat.ID
	}

	services := []models.Service{
		{
			CategoryID:  categoryMap["Plomberie"],
			Name:        "Réparation de fuites",
			Description: "Services de plomberie professionnels incluant réparation de fuites, installation de robinets, réparation de chauffe-eau et maintenance des systèmes d'égout.",
			Price:       1500.0,
			Duration:   180, // 3 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Électricité"],
			Name:        "Installation électrique",
			Description: "Services électriques complets : installation électrique, réparation de panneaux, éclairage LED, sécurité électrique et maintenance préventive.",
			Price:       2000.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Peinture"],
			Name:        "Peinture intérieure",
			Description: "Services de peinture intérieure et extérieure, préparation des surfaces, finitions décoratives et rénovation complète des murs et plafonds.",
			Price:       800.0,
			Duration:   1440, // 24 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Climatisation"],
			Name:        "Installation climatiseur",
			Description: "Installation, réparation et maintenance de systèmes de climatisation et chauffage, nettoyage des filtres et optimisation énergétique.",
			Price:       3000.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Menuiserie & Serrurerie"],
			Name:        "Réparation de portes",
			Description: "Fabrication et réparation de meubles sur mesure, portes, fenêtres, escaliers et aménagements intérieurs en bois de qualité.",
			Price:       2500.0,
			Duration:   7200, // 5 days in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Nettoyage à la demande"],
			Name:        "Nettoyage complet",
			Description: "Services de nettoyage professionnel : nettoyage résidentiel, commercial, après rénovation et entretien régulier des locaux.",
			Price:       500.0,
			Duration:   240, // 4 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Chauffe-eau"],
			Name:        "Installation chauffe-eau",
			Description: "Installation et réparation de chauffe-eau et systèmes solaires thermiques.",
			Price:       1800.0,
			Duration:   120, // 2 hours in minutes
			IsActive:   true,
		},
		{
			CategoryID:  categoryMap["Appareils électroménagers"],
			Name:        "Réparation frigo",
			Description: "Réparation de frigos, machines à laver et autres appareils électroménagers.",
			Price:       1200.0,
			Duration:   90, // 1.5 hours in minutes
			IsActive:   true,
		},
	}

	for _, service := range services {
		// Skip if category not found
		if service.CategoryID == 0 {
			log.Printf("Warning: Category not found for service %s", service.Name)
			continue
		}

		result := database.DB.Create(&service)
		if result.Error != nil {
			log.Printf("Error seeding service %s: %v", service.Name, result.Error)
		} else {
			log.Printf("✅ Seeded service: %s", service.Name)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Services seeded successfully", "count": len(services)})
}
