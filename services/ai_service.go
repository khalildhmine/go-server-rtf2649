package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"repair-service-server/database"
	"repair-service-server/models"
	"time"
)

type AIService struct {
	apiKey string
	client *http.Client
}

type GeminiRequest struct {
	Contents []Content `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK           int     `json:"topK"`
	TopP           float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

type AIResponse struct {
	Text string `json:"text"`
	Card *AICard `json:"card,omitempty"`
}

type AICard struct {
	Worker *WorkerCard `json:"worker,omitempty"`
	Task   *TaskCard   `json:"task,omitempty"`
	Buttons []string   `json:"buttons,omitempty"`
}

type WorkerCard struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	PhotoURL string  `json:"photo_url"`
	Rating   float64 `json:"rating"`
	Distance float64 `json:"distance"`
	Category string  `json:"category"`
	Price    int     `json:"price"`
	Time     string  `json:"time"`
}

type TaskCard struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
	Time        string `json:"time"`
}

func NewAIService() *AIService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Printf("⚠️ GEMINI_API_KEY not set, AI features will be disabled")
	}

	return &AIService{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (ai *AIService) ProcessUserInput(userInput string, messageType string, imageData string, voiceData string, userID uint, language string, conversationHistory []map[string]interface{}) (*AIResponse, error) {
	if ai.apiKey == "" {
		return &AIResponse{
			Text: "AI service is currently unavailable. Please contact support.",
		}, nil
	}

	// Get user location for worker matching
	userLocation, err := ai.getUserLocation(userID)
	if err != nil {
		log.Printf("⚠️ Failed to get user location: %v", err)
	}

	// Get available workers near user
	workers, err := ai.getAvailableWorkers(userLocation)
	if err != nil {
		log.Printf("⚠️ Failed to get available workers: %v", err)
	} else {
		log.Printf("🔍 AI Service found %d available workers", len(workers))
		for i, worker := range workers {
			log.Printf("👷 Worker %d: %s (%s) - Rating: %.1f, Price: %d", i+1, worker.Name, worker.Category, worker.Rating, worker.Price)
		}
	}

	// Get service categories for context
	categories, err := ai.getServiceCategories()
	if err != nil {
		log.Printf("⚠️ Failed to get service categories: %v", err)
	}

	// Build conversation context
	context := ai.buildConversationContext(conversationHistory, workers, categories, language)
	log.Printf("🔍 AI Context built with %d workers: %s", len(workers), context)

	// Create prompt based on input type
	var prompt string
	if messageType == "image" && imageData != "" {
		prompt = ai.buildImagePrompt(userInput, imageData, context, language)
	} else if messageType == "voice" && voiceData != "" {
		prompt = ai.buildVoicePrompt(userInput, voiceData, context, language)
	} else {
		prompt = ai.buildTextPrompt(userInput, context, language)
	}

	// Call Gemini API
	response, err := ai.callGeminiAPI(prompt, imageData, voiceData)
	if err != nil {
		return nil, fmt.Errorf("failed to call gemini API: %v", err)
	}

	// Parse response and create worker card if applicable
	aiResponse, err := ai.parseAIResponse(response, workers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ai response: %v", err)
	}

	return aiResponse, nil
}

func (ai *AIService) buildConversationContext(history []map[string]interface{}, workers []WorkerCard, categories []models.ServiceCategory, language string) string {
	context := fmt.Sprintf(`
Language: %s
Available Workers: %d
Service Categories: %d

Workers Data:
`, language, len(workers), len(categories))

	for _, worker := range workers {
		context += fmt.Sprintf("- %s (%s): Rating %.1f, %dkm away, %d OMR\n", 
			worker.Name, worker.Category, worker.Rating, int(worker.Distance), worker.Price)
	}

	context += "\nService Categories:\n"
	for _, category := range categories {
		context += fmt.Sprintf("- %s: %s\n", category.Name, category.Description)
	}

	context += "\nConversation History:\n"
	for i, msg := range history {
		msgType := "user"
		if msg["type"] == "ai" {
			msgType = "assistant"
		}
		context += fmt.Sprintf("%d. %s: %s\n", i+1, msgType, msg["content"])
	}

	return context
}

func (ai *AIService) buildTextPrompt(userInput, context, language string) string {
	basePrompt := `
You are a professional home repair assistant for a service platform in Mauritania. 
Your role is to help customers with home repair issues and connect them with the best available workers.

IMPORTANT RULES:
1. ONLY respond to home repair related queries
2. Keep responses under 50 words
3. Always suggest a specific worker if the issue is repair-related
4. If off-topic, politely redirect to repair issues only
5. Be professional, helpful, and concise
6. Respond in the user's language: %s
7. Use ONLY the real worker data provided in the context below

Context:
%s

User Input: %s

Respond with JSON format using REAL worker data from context:
{
  "text": "Your response here",
  "card": {
    "worker": {
      "id": "use_real_worker_id_from_context",
      "name": "use_real_worker_name_from_context", 
      "photo_url": "use_real_worker_photo_from_context",
      "rating": "use_real_worker_rating_from_context",
      "distance": "use_real_worker_distance_from_context",
      "category": "use_real_worker_category_from_context",
      "price": "use_real_worker_price_from_context",
      "time": "now"
    },
    "task": {
      "description": "Task description based on user input",
      "price": "use_real_worker_price_from_context",
      "time": "now"
    },
    "buttons": ["Accept", "Decline"]
  }
}

If off-topic or not repair-related, respond with:
{
  "text": "Sorry, I can only help with home repair issues. Please describe your problem.",
  "card": null
}
`

	return fmt.Sprintf(basePrompt, language, context, userInput)
}

func (ai *AIService) buildImagePrompt(userInput, imageData, context, language string) string {
	// For image analysis, we'll use a simpler text prompt since Gemini 1.5 Flash
	// doesn't support direct image input in this implementation
	return ai.buildTextPrompt(fmt.Sprintf("User sent an image with description: %s", userInput), context, language)
}

func (ai *AIService) buildVoicePrompt(userInput, voiceData, context, language string) string {
	// For voice analysis, we'll use a simpler text prompt since we need to
	// implement voice-to-text conversion first
	return ai.buildTextPrompt(fmt.Sprintf("User sent a voice message: %s", userInput), context, language)
}

func (ai *AIService) callGeminiAPI(prompt, imageData, voiceData string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", ai.apiKey)

	var parts []Part
	parts = append(parts, Part{Text: prompt})

	// Add image data if present
	if imageData != "" {
		parts = append(parts, Part{
			InlineData: &InlineData{
				MimeType: "image/jpeg",
				Data:     imageData,
			},
		})
	}

	request := GeminiRequest{
		Contents: []Content{
			{Parts: parts},
		},
		GenerationConfig: GenerationConfig{
			Temperature:     0.7,
			TopK:           40,
			TopP:           0.95,
			MaxOutputTokens: 1024,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	resp, err := ai.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: %s", string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (ai *AIService) parseAIResponse(response string, workers []WorkerCard) (*AIResponse, error) {
	log.Printf("🔍 Parsing AI response with %d workers available", len(workers))
	log.Printf("🔍 Raw AI response: %s", response)
	
	// Try to parse as JSON first
	var aiResp AIResponse
	if err := json.Unmarshal([]byte(response), &aiResp); err == nil {
		log.Printf("🔍 AI response parsed successfully, has card: %v", aiResp.Card != nil)
		if aiResp.Card != nil {
			log.Printf("🔍 AI card before injection: %+v", aiResp.Card)
		}
		
		// If we have workers and the AI wants to show a card, use real worker data
		if aiResp.Card != nil && aiResp.Card.Worker != nil && len(workers) > 0 {
			log.Printf("🔍 Injecting real worker data: %s", workers[0].Name)
			log.Printf("🔍 Real worker data: %+v", workers[0])
			
			// Use the first available worker's real data
			realWorker := workers[0]
			aiResp.Card.Worker.ID = realWorker.ID
			aiResp.Card.Worker.Name = realWorker.Name
			aiResp.Card.Worker.PhotoURL = realWorker.PhotoURL
			aiResp.Card.Worker.Rating = realWorker.Rating
			aiResp.Card.Worker.Distance = realWorker.Distance
			aiResp.Card.Worker.Category = realWorker.Category
			aiResp.Card.Worker.Price = realWorker.Price
			
			// Update task price to match worker price
			if aiResp.Card.Task != nil {
				aiResp.Card.Task.Price = realWorker.Price
			}
			
			log.Printf("🔍 Final worker card after injection: %+v", aiResp.Card.Worker)
		} else {
			log.Printf("🔍 No workers available or no card requested")
		}
		return &aiResp, nil
	}

	// If not JSON, return as plain text
	return &AIResponse{
		Text: response,
	}, nil
}

// calculateDistance calculates the distance between two points using the Haversine formula
func (ai *AIService) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadius * c
	return math.Round(distance*10) / 10 // Round to 1 decimal place
}

func (ai *AIService) getUserLocation(userID uint) (*models.Address, error) {
	var address models.Address
	err := database.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&address).Error
	return &address, err
}

func (ai *AIService) getAvailableWorkers(userLocation *models.Address) ([]WorkerCard, error) {
	var workers []WorkerCard

	query := database.DB.Table("worker_profiles").
		Select("worker_profiles.id, users.full_name, worker_profiles.profile_photo, worker_profiles.rating, worker_profiles.hourly_rate, service_categories.name as category_name, worker_profiles.current_lat, worker_profiles.current_lng").
		Joins("JOIN users ON worker_profiles.user_id = users.id").
		Joins("JOIN service_categories ON worker_profiles.category_id = service_categories.id").
		Where("worker_profiles.is_available = ?", true)

	if userLocation != nil {
		// Add distance calculation (simplified)
		query = query.Where("worker_profiles.current_lat IS NOT NULL AND worker_profiles.current_lng IS NOT NULL")
	}

	var results []struct {
		ID       uint     `gorm:"column:id"`
		Name     string   `gorm:"column:full_name"`
		PhotoURL *string  `gorm:"column:profile_photo"`
		Rating   float64  `gorm:"column:rating"`
		Price    int      `gorm:"column:hourly_rate"`
		Category string   `gorm:"column:category_name"`
		Lat      *float64 `gorm:"column:current_lat"`
		Lng      *float64 `gorm:"column:current_lng"`
	}

	if err := query.Limit(5).Find(&results).Error; err != nil {
		log.Printf("⚠️ Failed to get available workers: %v", err)
		return workers, err
	}

	log.Printf("🔍 Found %d available workers for AI matching", len(results))
	for i, result := range results {
		log.Printf("👷 Worker %d: %s (%s) - Rating: %.1f, Rate: %d", i+1, result.Name, result.Category, result.Rating, result.Price)
		log.Printf("🔍 Raw result %d: %+v", i+1, result)
	}

	for _, result := range results {
		photoURL := ""
		if result.PhotoURL != nil {
			photoURL = *result.PhotoURL
			log.Printf("🔍 Worker %s profile photo: %s", result.Name, photoURL)
		} else {
			log.Printf("🔍 Worker %s has no profile photo", result.Name)
		}

		// Calculate distance
		distance := 2.5 // Default distance
		if userLocation != nil && result.Lat != nil && result.Lng != nil {
			distance = ai.calculateDistance(
				userLocation.Latitude, userLocation.Longitude,
				*result.Lat, *result.Lng,
			)
			log.Printf("🔍 Calculated distance: %.1fkm (User: %.6f,%.6f -> Worker: %.6f,%.6f)", 
				distance, userLocation.Latitude, userLocation.Longitude, *result.Lat, *result.Lng)
		} else {
			log.Printf("🔍 Using default distance: %.1fkm (User: %v, Worker Lat: %v, Lng: %v)", 
				distance, userLocation != nil, result.Lat != nil, result.Lng != nil)
		}

		workerCard := WorkerCard{
			ID:       int(result.ID),
			Name:     result.Name,
			PhotoURL: photoURL,
			Rating:   result.Rating,
			Distance: distance,
			Category: result.Category,
			Price:    result.Price,
			Time:     "now",
		}
		
		log.Printf("🔍 Created worker card: %+v", workerCard)
		workers = append(workers, workerCard)
	}

	return workers, nil
}

func (ai *AIService) getServiceCategories() ([]models.ServiceCategory, error) {
	var categories []models.ServiceCategory
	err := database.DB.Where("is_active = ?", true).Find(&categories).Error
	return categories, err
}
