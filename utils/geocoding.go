package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GeocodingResult represents the result of a geocoding operation
type GeocodingResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
}

// GeocodeAddress converts a text address to coordinates using OpenStreetMap Nominatim
// This is a free service, but for production use, consider using Google Maps API or similar
func GeocodeAddress(addressText string) (*GeocodingResult, error) {
	// Clean and format the address
	cleanAddress := strings.TrimSpace(addressText)
	if cleanAddress == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Add city context for better accuracy in Mauritania
	if !strings.Contains(strings.ToLower(cleanAddress), "nouakchott") {
		cleanAddress = cleanAddress + ", Nouakchott, Mauritania"
	}

	// Encode the address for URL
	encodedAddress := url.QueryEscape(cleanAddress)

	// Build the Nominatim API URL
	apiURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1&countrycodes=MR", encodedAddress)

	// Make the HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make geocoding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding service returned status: %d", resp.StatusCode)
	}

	// Parse the response
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	// Check if we got any results
	if len(results) == 0 {
		// Return default Nouakchott coordinates if no results found
		return &GeocodingResult{
			Latitude:  18.0799,
			Longitude: -15.9653,
			City:      "Nouakchott",
		}, nil
	}

	// Parse the coordinates
	result := results[0]
	lat, err := parseFloat(result.Lat)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude in response: %w", err)
	}

	lon, err := parseFloat(result.Lon)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude in response: %w", err)
	}

	// Extract city from display name
	city := extractCity(result.DisplayName)

	return &GeocodingResult{
		Latitude:  lat,
		Longitude: lon,
		City:      city,
	}, nil
}

// parseFloat is a helper function to parse string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// extractCity extracts the city name from the display name
func extractCity(displayName string) string {
	parts := strings.Split(displayName, ",")
	if len(parts) > 0 {
		// Return the first part as city, or default to Nouakchott
		city := strings.TrimSpace(parts[0])
		if city != "" {
			return city
		}
	}
	return "Nouakchott"
}

// GetDefaultCoordinates returns the default coordinates for Nouakchott
func GetDefaultCoordinates() *GeocodingResult {
	return &GeocodingResult{
		Latitude:  18.0799,
		Longitude: -15.9653,
		City:      "Nouakchott",
	}
}
