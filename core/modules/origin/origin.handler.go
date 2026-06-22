package origin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lumbung-fs/core/database"
	originModel "lumbung-fs/core/modules/origin/model"
	ruleModel "lumbung-fs/core/modules/rule/model"
	"lumbung-fs/core/variables"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Helper to write JSON responses
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Helper to write error responses
func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, map[string]string{"error": message})
}

// ListOrigins handles GET /api/origins
func ListOrigins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var origins []originModel.Origin
	if err := database.DB.Find(&origins).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, origins)
}

// CreateOrigin handles POST /api/origins
func CreateOrigin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		Domain    string `json:"domain"`
		IsBlocked bool   `json:"is_blocked"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	domainClean := strings.TrimSpace(strings.ToLower(input.Domain))
	domainClean = strings.TrimPrefix(domainClean, "https://")
	domainClean = strings.TrimPrefix(domainClean, "http://")
	domainClean = strings.Split(domainClean, "/")[0]
	
	if domainClean == "" {
		respondWithError(w, http.StatusBadRequest, "Domain is required")
		return
	}

	keyUUID, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}
	apiKey := "lf_" + strings.ReplaceAll(keyUUID.String(), "-", "")

	origin := originModel.Origin{
		Domain:    domainClean,
		IsBlocked: input.IsBlocked,
		ApiKey:    apiKey,
	}

	if err := database.DB.Create(&origin).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			respondWithError(w, http.StatusConflict, "Origin domain already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, origin)
}

// UpdateOrigin handles PUT /api/origins
func UpdateOrigin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	var input struct {
		Domain    string `json:"domain"`
		IsBlocked bool   `json:"is_blocked"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var origin originModel.Origin
	if err := database.DB.Where("id = ?", id).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Origin not found")
		return
	}

	oldDomain := origin.Domain

	// Update fields
	if input.Domain != "" {
		domainClean := strings.TrimSpace(strings.ToLower(input.Domain))
		domainClean = strings.TrimPrefix(domainClean, "https://")
		domainClean = strings.TrimPrefix(domainClean, "http://")
		domainClean = strings.Split(domainClean, "/")[0]
		origin.Domain = domainClean
	}
	origin.IsBlocked = input.IsBlocked

	// Perform folder rename if domain changed
	if input.Domain != "" && origin.Domain != oldDomain {
		oldFolder := filepath.Join(variables.BucketDir, variables.DomainToSnake(oldDomain))
		newFolder := filepath.Join(variables.BucketDir, variables.DomainToSnake(origin.Domain))

		// Check if old folder exists, then rename
		if _, err := os.Stat(oldFolder); err == nil {
			if err := os.Rename(oldFolder, newFolder); err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to rename storage folder: "+err.Error())
				return
			}
		}
	}

	if err := database.DB.Save(&origin).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, origin)
}

// DeleteOrigin handles DELETE /api/origins
func DeleteOrigin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	var origin originModel.Origin
	if err := database.DB.Where("id = ?", id).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Origin not found")
		return
	}

	tx := database.DB.Begin()
	// Explicitly delete rules associated with this origin_id to ensure cascade delete
	if err := tx.Where("origin_id = ?", id).Delete(&ruleModel.Rule{}).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Delete(&origin).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tx.Commit()

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Origin deleted successfully"})
}

// GenerateApiKey handles POST /api/origins/apikey
func GenerateApiKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	var origin originModel.Origin
	if err := database.DB.Where("id = ?", id).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Origin not found")
		return
	}

	keyUUID, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}
	// Format: lf_<uuid_without_dashes>
	apiKey := "lf_" + strings.ReplaceAll(keyUUID.String(), "-", "")

	origin.ApiKey = apiKey
	if err := database.DB.Save(&origin).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, origin)
}

// ListUnknownOrigins handles GET /api/unknown-origins
func ListUnknownOrigins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var unknowns []originModel.UnknownOrigin
	if err := database.DB.Order("access_at desc").Find(&unknowns).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, unknowns)
}

// DeleteUnknownOrigin handles DELETE /api/unknown-origins
func DeleteUnknownOrigin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		// If no ID is passed, delete all logs
		if err := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&originModel.UnknownOrigin{}).Error; err != nil {
			// fallback check
		}
		
		// To be safe with GORM:
		if err := database.DB.Exec("DELETE FROM unknown_origins").Error; err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "All unknown origins cleared"})
		return
	}

	var unknown originModel.UnknownOrigin
	if err := database.DB.Where("id = ?", id).First(&unknown).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Unknown origin log not found")
		return
	}

	if err := database.DB.Delete(&unknown).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Unknown origin log deleted"})
}

// PromoteUnknownOrigin handles POST /api/unknown-origins/promote
func PromoteUnknownOrigin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	// 1. Find the unknown origin log
	var unknown originModel.UnknownOrigin
	if err := database.DB.Where("id = ?", id).First(&unknown).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Unknown origin log not found")
		return
	}

	keyUUID, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}
	apiKey := "lf_" + strings.ReplaceAll(keyUUID.String(), "-", "")

	// 2. Insert into origins table
	origin := originModel.Origin{
		Domain:    unknown.Domain,
		IsBlocked: false,
		ApiKey:    apiKey,
	}

	// Begin transaction to ensure consistency
	tx := database.DB.Begin()
	if err := tx.Create(&origin).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "UNIQUE") {
			respondWithError(w, http.StatusConflict, "Origin domain already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 3. Delete the log
	if err := tx.Delete(&unknown).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()

	respondWithJSON(w, http.StatusCreated, origin)
}
