package routes

import (
	"log"
	"net/http"

	"repair-service-server/database"
	"repair-service-server/models"

	"github.com/gin-gonic/gin"
)

// RegisterCategoryRoutes registers category-related routes
func RegisterCategoryRoutes(router *gin.RouterGroup) {
	categories := router.Group("/categories")
	{
		categories.GET("", GetServiceCategories)
	}
}

// GetServiceCategories returns all active service categories
func GetServiceCategories(c *gin.Context) {
	db := database.GetDB()

	var categories []models.ServiceCategory
	if err := db.Where("is_active = ?", true).Order("sort_order ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch service categories",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"categories": categories,
	})
}

// CreateCategory creates a new service category
func CreateCategory(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	category := models.ServiceCategory{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		SortOrder:   0,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		log.Printf("❌ Failed to create category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	log.Printf("✅ Category created: %s (ID: %d)", category.Name, category.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Category created successfully",
		"data":    category,
	})
}

// UpdateCategory updates an existing service category
func UpdateCategory(c *gin.Context) {
	categoryID := c.Param("id")
	
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var category models.ServiceCategory
	if err := database.DB.First(&category, categoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	category.Name = req.Name
	category.Description = req.Description

	if err := database.DB.Save(&category).Error; err != nil {
		log.Printf("❌ Failed to update category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	log.Printf("✅ Category updated: %s (ID: %d)", category.Name, category.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category updated successfully",
		"data":    category,
	})
}

// DeleteCategory deletes a service category
func DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")

	var category models.ServiceCategory
	if err := database.DB.First(&category, categoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	if err := database.DB.Delete(&category).Error; err != nil {
		log.Printf("❌ Failed to delete category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	log.Printf("✅ Category deleted: %s (ID: %d)", category.Name, category.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category deleted successfully",
	})
}
