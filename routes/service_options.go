package routes

import (
	"net/http"
	"repair-service-server/database"
	"repair-service-server/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetServiceOptionsByCategory retrieves all service options for a specific category
func GetServiceOptionsByCategory(c *gin.Context) {
	categoryIDStr := c.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid category ID",
		})
		return
	}

	var serviceOptions []models.ServiceOption
	result := database.DB.Where("category_id = ? AND is_active = ?", categoryID, true).
		Order("sort_order ASC, title ASC").
		Preload("Category").
		Find(&serviceOptions)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch service options",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    serviceOptions,
	})
}

// GetAllServiceOptions retrieves all service options (admin only)
func GetAllServiceOptions(c *gin.Context) {
	var serviceOptions []models.ServiceOption
	result := database.DB.Order("category_id ASC, sort_order ASC, title ASC").
		Preload("Category").
		Find(&serviceOptions)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch service options",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    serviceOptions,
	})
}

// CreateServiceOption creates a new service option (admin only)
func CreateServiceOption(c *gin.Context) {
	var serviceOption models.ServiceOption
	if err := c.ShouldBindJSON(&serviceOption); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
		})
		return
	}

	// Validate required fields
	if serviceOption.Title == "" || serviceOption.Description == "" || serviceOption.CategoryID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Title, description, and category are required",
		})
		return
	}

	result := database.DB.Create(&serviceOption)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create service option",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Service option created successfully",
		"data":    serviceOption,
	})
}

// UpdateServiceOption updates an existing service option (admin only)
func UpdateServiceOption(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid ID",
		})
		return
	}

	var serviceOption models.ServiceOption
	if err := database.DB.First(&serviceOption, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Service option not found",
		})
		return
	}

	var updateData models.ServiceOption
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
		})
		return
	}

	result := database.DB.Model(&serviceOption).Updates(updateData)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update service option",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service option updated successfully",
		"data":    serviceOption,
	})
}

// DeleteServiceOption deletes a service option (admin only)
func DeleteServiceOption(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid ID",
		})
		return
	}

	var serviceOption models.ServiceOption
	if err := database.DB.First(&serviceOption, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Service option not found",
		})
		return
	}

	result := database.DB.Delete(&serviceOption)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete service option",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service option deleted successfully",
	})
}

// RegisterServiceOptionRoutes registers all service option routes
func RegisterServiceOptionRoutes(router *gin.RouterGroup) {
	serviceOptions := router.Group("/service-options")
	{
		serviceOptions.GET("/category/:categoryId", GetServiceOptionsByCategory)
		serviceOptions.GET("/", GetAllServiceOptions)
		serviceOptions.POST("/", CreateServiceOption)
		serviceOptions.PUT("/:id", UpdateServiceOption)
		serviceOptions.DELETE("/:id", DeleteServiceOption)
	}
}
