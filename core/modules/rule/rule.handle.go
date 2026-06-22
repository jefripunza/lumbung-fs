package rule

import (
	"encoding/json"
	"net/http"
	"strings"

	"lumbung-fs/core/database"
	ruleModel "lumbung-fs/core/modules/rule/model"
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

// ListRules handles GET /api/rules
func ListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	originID := r.URL.Query().Get("origin_id")
	
	var rules []ruleModel.Rule
	var err error
	
	if originID != "" {
		err = database.DB.Where("origin_id = ?", originID).Find(&rules).Error
	} else {
		err = database.DB.Find(&rules).Error
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, rules)
}

// CreateRule handles POST /api/rules
func CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		OriginID            string `json:"origin_id"`
		Path                string `json:"path"`
		ValidateMethod      string `json:"validate_method"`
		ValidateHeaders     string `json:"validate_headers"`
		ValidateURL         string `json:"validate_url"`
		ValidateFallbackURL string `json:"validate_fallback_url"`
		IsMaxSize           bool   `json:"is_max_size"`
		ValueMaxSize        int    `json:"value_max_size"`
		ValueUnitSize       string `json:"value_unit_size"`
		IsExtensions        bool   `json:"is_extensions"`
		ValueExtensions     string `json:"value_extensions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if input.OriginID == "" {
		respondWithError(w, http.StatusBadRequest, "Origin ID is required")
		return
	}

	pathClean := strings.TrimSpace(strings.Trim(input.Path, "/"))
	pathClean = strings.TrimPrefix(pathClean, "file/")
	pathClean = strings.Trim(pathClean, "/")
	if pathClean == "" {
		respondWithError(w, http.StatusBadRequest, "Path is required")
		return
	}

	unitClean := strings.ToUpper(strings.TrimSpace(input.ValueUnitSize))
	if unitClean == "" {
		unitClean = "MB"
	}

	rule := ruleModel.Rule{
		OriginID:            input.OriginID,
		Path:                pathClean,
		ValidateMethod:      strings.TrimSpace(input.ValidateMethod),
		ValidateHeaders:     strings.TrimSpace(input.ValidateHeaders),
		ValidateURL:         strings.TrimSpace(input.ValidateURL),
		ValidateFallbackURL: strings.TrimSpace(input.ValidateFallbackURL),
		IsMaxSize:           input.IsMaxSize,
		ValueMaxSize:        input.ValueMaxSize,
		ValueUnitSize:       unitClean,
		IsExtensions:        input.IsExtensions,
		ValueExtensions:     strings.TrimSpace(input.ValueExtensions),
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, rule)
}

// UpdateRule handles PUT /api/rules
func UpdateRule(w http.ResponseWriter, r *http.Request) {
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
		Path                string `json:"path"`
		ValidateMethod      string `json:"validate_method"`
		ValidateHeaders     string `json:"validate_headers"`
		ValidateURL         string `json:"validate_url"`
		ValidateFallbackURL string `json:"validate_fallback_url"`
		IsMaxSize           bool   `json:"is_max_size"`
		ValueMaxSize        int    `json:"value_max_size"`
		ValueUnitSize       string `json:"value_unit_size"`
		IsExtensions        bool   `json:"is_extensions"`
		ValueExtensions     string `json:"value_extensions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var rule ruleModel.Rule
	if err := database.DB.Where("id = ?", id).First(&rule).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	// Update fields
	if input.Path != "" {
		pathClean := strings.TrimSpace(strings.Trim(input.Path, "/"))
		pathClean = strings.TrimPrefix(pathClean, "file/")
		pathClean = strings.Trim(pathClean, "/")
		rule.Path = pathClean
	}
	rule.ValidateMethod = strings.TrimSpace(input.ValidateMethod)
	rule.ValidateHeaders = strings.TrimSpace(input.ValidateHeaders)
	rule.ValidateURL = strings.TrimSpace(input.ValidateURL)
	rule.ValidateFallbackURL = strings.TrimSpace(input.ValidateFallbackURL)
	rule.IsMaxSize = input.IsMaxSize
	rule.ValueMaxSize = input.ValueMaxSize
	if input.ValueUnitSize != "" {
		rule.ValueUnitSize = strings.ToUpper(strings.TrimSpace(input.ValueUnitSize))
	}
	rule.IsExtensions = input.IsExtensions
	rule.ValueExtensions = strings.TrimSpace(input.ValueExtensions)

	if err := database.DB.Save(&rule).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, rule)
}

// DeleteRule handles DELETE /api/rules
func DeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	var rule ruleModel.Rule
	if err := database.DB.Where("id = ?", id).First(&rule).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	if err := database.DB.Delete(&rule).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted successfully"})
}
