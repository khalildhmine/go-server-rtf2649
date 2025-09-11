package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/utils"
)

// RegisterAddressRoutes registers address-related routes
func RegisterAddressRoutes(router *gin.RouterGroup) {
	router.GET("/", getUserAddresses)
	router.POST("/", createAddress)
	router.GET("/:id", getAddress)
	router.PUT("/:id", updateAddress)
	router.DELETE("/:id", deleteAddress)
	router.PUT("/:id/default", setDefaultAddress)
}

// getUserAddresses gets all addresses for the current user
func getUserAddresses(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var addresses []models.Address
	if err := database.DB.Where("user_id = ?", userID).Order("is_default DESC, created_at DESC").Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get addresses",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Addresses retrieved successfully",
		"data":    addresses,
	})
}

// createAddress creates a new address for the current user
func createAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Debug logging
	log.Printf("🔍 createAddress: Extracted user_id from context: %d", userID)
	log.Printf("🔍 createAddress: All context keys: %v", c.Keys)
	
	if userID == 0 {
		log.Printf("❌ createAddress: user_id is 0, authentication failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication required",
			"message": "User ID not found in context",
		})
		return
	}

	var req models.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	// Log received coordinates for debugging
	log.Printf("🔍 Received coordinates: lat=%f, lng=%f", req.Latitude, req.Longitude)
	
	// Use geocoding to get coordinates if not provided
	if req.Latitude == 0 && req.Longitude == 0 {
		log.Printf("🔍 Using geocoding for coordinates")
		geocodedAddress := req.AddressDetails
		if req.City != "" {
			geocodedAddress = geocodedAddress + ", " + req.City
		}
		
		geocodingResult, err := utils.GeocodeAddress(geocodedAddress)
		if err != nil {
			// Use default coordinates if geocoding fails
			geocodingResult = utils.GetDefaultCoordinates()
		}
		
		req.Latitude = geocodingResult.Latitude
		req.Longitude = geocodingResult.Longitude
		if req.City == "" {
			req.City = geocodingResult.City
		}
	} else {
		log.Printf("🔍 Using provided GPS coordinates: lat=%f, lng=%f", req.Latitude, req.Longitude)
	}

	// If this is the first address or marked as default, set it as default
	if req.IsDefault {
		// Remove default from other addresses
		if err := database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update existing addresses",
				"message": err.Error(),
			})
			return
		}
	}

	address := models.Address{
		UserID:         userID,
		Label:          req.Label,
		AddressDetails: req.AddressDetails,
		City:           req.City,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		IsDefault:      req.IsDefault,
	}

	if err := database.DB.Create(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create address",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Address created successfully",
		"data":    address,
	})
}

// getAddress gets a specific address for the current user
func getAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	addressID := c.Param("id")
	
	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Address not found",
			"message": "The requested address does not exist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address retrieved successfully",
		"data":    address,
	})
}

// updateAddress updates an existing address for the current user
func updateAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	addressID := c.Param("id")
	
	var req models.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	// Check if address exists and belongs to user
	var existingAddress models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addressID, userID).First(&existingAddress).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Address not found",
			"message": "The requested address does not exist",
		})
		return
	}

	// If setting this address as default, remove default from others
	if req.IsDefault && !existingAddress.IsDefault {
		if err := database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update existing addresses",
				"message": err.Error(),
			})
			return
		}
	}

	// Use geocoding to get coordinates if not provided
	if req.Latitude == 0 && req.Longitude == 0 {
		geocodedAddress := req.AddressDetails
		if req.City != "" {
			geocodedAddress = geocodedAddress + ", " + req.City
		}
		
		geocodingResult, err := utils.GeocodeAddress(geocodedAddress)
		if err != nil {
			// Use default coordinates if geocoding fails
			geocodingResult = utils.GetDefaultCoordinates()
		}
		
		req.Latitude = geocodingResult.Latitude
		req.Longitude = geocodingResult.Longitude
		if req.City == "" {
			req.City = geocodingResult.City
		}
	}

	// Update address
	updates := map[string]interface{}{
		"label":           req.Label,
		"address_details": req.AddressDetails,
		"city":            req.City,
		"latitude":        req.Latitude,
		"longitude":       req.Longitude,
		"is_default":      req.IsDefault,
	}

	if err := database.DB.Model(&existingAddress).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update address",
			"message": err.Error(),
		})
		return
	}

	// Get updated address
	var updatedAddress models.Address
	database.DB.First(&updatedAddress, addressID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Address updated successfully",
		"data":    updatedAddress,
	})
}

// deleteAddress deletes an address for the current user
func deleteAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	addressID := c.Param("id")
	
	// Check if address exists and belongs to user
	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Address not found",
			"message": "The requested address does not exist",
		})
		return
	}

	// If deleting default address, set another address as default
	if address.IsDefault {
		var otherAddress models.Address
		if err := database.DB.Where("user_id = ? AND id != ?", userID, addressID).First(&otherAddress).Error; err == nil {
			// Set another address as default
			database.DB.Model(&otherAddress).Update("is_default", true)
		}
	}

	// Delete the address
	if err := database.DB.Delete(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete address",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address deleted successfully",
	})
}

// setDefaultAddress sets an address as the default for the current user
func setDefaultAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	addressID := c.Param("id")
	
	// Check if address exists and belongs to user
	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Address not found",
			"message": "The requested address does not exist",
		})
		return
	}

	// Remove default from all other addresses
	if err := database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update existing addresses",
			"message": err.Error(),
		})
		return
	}

	// Set this address as default
	if err := database.DB.Model(&address).Update("is_default", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to set address as default",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address set as default successfully",
		"data":    address,
	})
}
