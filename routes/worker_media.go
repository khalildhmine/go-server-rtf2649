package routes

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
)

// validateImageFile validates mimetype and size (<= 5MB)
func validateImageFile(h *multipart.FileHeader) bool {
    if h == nil || h.Size <= 0 || h.Size > 5*1024*1024 {
        return false
    }
    ext := strings.ToLower(filepath.Ext(h.Filename))
    switch ext {
    case ".jpg", ".jpeg", ".png", ".webp":
        return true
    default:
        return false
    }
}

// RegisterWorkerMediaRoutes adds media upload endpoints under protected group
func RegisterWorkerMediaRoutes(rg *gin.RouterGroup) {
    rg.POST("/workers/profile/photos", func(c *gin.Context) {
        userID := c.GetUint("user_id")

        // Multipart form
        if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid form data"})
            return
        }

        profileHeader, _ := c.FormFile("profile_photo")
        idHeader, _ := c.FormFile("id_card_photo")
        idBackHeader, _ := c.FormFile("id_card_photo_back")

        log.Printf("📸 Received files - Profile: %v, ID Front: %v, ID Back: %v", 
            profileHeader != nil, idHeader != nil, idBackHeader != nil)
        
        if profileHeader != nil {
            log.Printf("📸 Profile file: %s, size: %d", profileHeader.Filename, profileHeader.Size)
        }
        if idHeader != nil {
            log.Printf("📸 ID front file: %s, size: %d", idHeader.Filename, idHeader.Size)
        }
        if idBackHeader != nil {
            log.Printf("📸 ID back file: %s, size: %d", idBackHeader.Filename, idBackHeader.Size)
        }

        if profileHeader == nil && idHeader == nil && idBackHeader == nil {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "No files provided"})
            return
        }

        if profileHeader != nil && !validateImageFile(profileHeader) {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid profile photo"})
            return
        }
        if idHeader != nil && !validateImageFile(idHeader) {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID card photo"})
            return
        }
        if idBackHeader != nil && !validateImageFile(idBackHeader) {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid ID card back photo"})
            return
        }

        // Ensure worker profile exists
        var wp models.WorkerProfile
        if err := database.DB.Where("user_id = ?", userID).First(&wp).Error; err != nil {
            c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Worker profile not found"})
            return
        }

        // Get Cloudinary configuration from environment
        cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
        apiKey := os.Getenv("CLOUDINARY_API_KEY")
        apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
        
        if cloudName == "" || apiKey == "" || apiSecret == "" {
            log.Printf("❌ Cloudinary environment variables not set: cloudName=%s, apiKey=%s, apiSecret=%s", cloudName, apiKey, apiSecret)
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Cloudinary not configured"})
            return
        }
        
        // Construct Cloudinary URL
        cloudinaryURL := fmt.Sprintf("cloudinary://%s:%s@%s", apiKey, apiSecret, cloudName)
        log.Printf("🔧 Using Cloudinary URL: cloudinary://%s:***@%s", apiKey, cloudName)
        
        cld, err := cloudinary.NewFromURL(cloudinaryURL)
        if err != nil {
            log.Printf("❌ Failed to initialize Cloudinary: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Cloudinary initialization failed"})
            return
        }

        ctx := context.Background()
        data := gin.H{}

        // Upload helper
        upload := func(field string, header *multipart.FileHeader, folder string) (string, error) {
            if header == nil { return "", nil }
            file, err := header.Open()
            if err != nil { return "", err }
            defer file.Close()
            ow := true
            uf := true
            up, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
                Folder:   folder,
                PublicID: strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)),
                Overwrite: &ow,
                UniqueFilename: &uf,
                ResourceType: "image",
            })
            if err != nil { return "", err }
            return up.SecureURL, nil
        }

        // Build folders
        base := "workers"
        profileFolder := base + "/profile_photos/" +  strconv.Itoa(int(userID))
        idFolder := base + "/id_cards/" +  strconv.Itoa(int(userID)) + "/front"
        idBackFolder := base + "/id_cards/" +  strconv.Itoa(int(userID)) + "/back"

        // Perform uploads
        if profileHeader != nil {
            log.Printf("📸 Uploading profile photo to folder: %s", profileFolder)
            if url, err := upload("profile_photo", profileHeader, profileFolder); err == nil {
                wp.ProfilePhoto = &url
                data["profile_photo_url"] = url
                log.Printf("✅ Profile photo uploaded successfully: %s", url)
            } else {
                log.Printf("❌ Profile photo upload failed: %v", err)
                c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Profile upload failed"})
                return
            }
        }
        if idHeader != nil {
            log.Printf("📸 Uploading ID card photo to folder: %s", idFolder)
            if url, err := upload("id_card_photo", idHeader, idFolder); err == nil {
                wp.IDCardPhoto = &url
                data["id_card_photo_url"] = url
                log.Printf("✅ ID card photo uploaded successfully: %s", url)
            } else {
                log.Printf("❌ ID card photo upload failed: %v", err)
                c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID card upload failed"})
                return
            }
        }
        if idBackHeader != nil {
            log.Printf("📸 Uploading ID card back photo to folder: %s", idBackFolder)
            if url, err := upload("id_card_photo_back", idBackHeader, idBackFolder); err == nil {
                wp.IDCardBackPhoto = &url
                data["id_card_photo_back_url"] = url
                log.Printf("✅ ID card back photo uploaded successfully: %s", url)
            } else {
                log.Printf("❌ ID card back photo upload failed: %v", err)
                c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID card back upload failed"})
                return
            }
        }

        wp.UpdatedAt = time.Now()
        if err := database.DB.Save(&wp).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to save profile"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
    })
}


