package utils

import (
	"math"
	"repair-service-server/models"
	"time"

	"gorm.io/gorm"
)

// Location represents a geographical coordinate
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HaversineDistance calculates the distance between two points on Earth using the Haversine formula
// Returns distance in kilometers
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Differences in coordinates
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// FindNearbyWorkers finds workers within a specified radius of a location
func FindNearbyWorkers(db *gorm.DB, location Location, radius float64, category models.WorkerCategory) ([]models.WorkerProfile, error) {
	var workers []models.WorkerProfile

	// Query workers in the specified category who are available
	err := db.Preload("User").
		Where("category = ? AND is_available = ? AND current_lat IS NOT NULL AND current_lng IS NOT NULL", 
			category, true).
		Find(&workers).Error

	if err != nil {
		return nil, err
	}

	// Filter workers by distance
	var nearbyWorkers []models.WorkerProfile
	for _, worker := range workers {
		if worker.CurrentLat != nil && worker.CurrentLng != nil {
			distance := HaversineDistance(
				location.Latitude, location.Longitude,
				*worker.CurrentLat, *worker.CurrentLng,
			)
			
			if distance <= radius {
				// Add distance to worker profile for response
				// Note: This is a temporary solution. In production, consider using PostGIS for better performance
				nearbyWorkers = append(nearbyWorkers, worker)
			}
		}
	}

	return nearbyWorkers, nil
}

// CalculateETA estimates the time of arrival for a worker
// This is a simplified calculation - in production, you might want to use Google Maps API
func CalculateETA(workerLocation, requestLocation Location, averageSpeed float64) time.Duration {
	distance := HaversineDistance(
		workerLocation.Latitude, workerLocation.Longitude,
		requestLocation.Latitude, requestLocation.Longitude,
	)
	
	// Convert distance to time (distance in km, speed in km/h)
	timeHours := distance / averageSpeed
	timeMinutes := int(timeHours * 60)
	
	return time.Duration(timeMinutes) * time.Minute
}

// IsLocationValid checks if the provided coordinates are valid
func IsLocationValid(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// IsLocationRecent checks if the location was updated recently (within last 30 minutes)
func IsLocationRecent(lastUpdate *time.Time) bool {
	if lastUpdate == nil {
		return false
	}
	
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
	return lastUpdate.After(thirtyMinutesAgo)
}

// GetDefaultBroadcastRadius returns the default broadcast radius in kilometers
func GetDefaultBroadcastRadius() float64 {
	return 10.0 // 10 kilometers
}

// GetMaxBroadcastRadius returns the maximum allowed broadcast radius in kilometers
func GetMaxBroadcastRadius() float64 {
	return 50.0 // 50 kilometers
}

// ValidateBroadcastRadius checks if the broadcast radius is within acceptable limits
func ValidateBroadcastRadius(radius float64) bool {
	return radius > 0 && radius <= GetMaxBroadcastRadius()
}
