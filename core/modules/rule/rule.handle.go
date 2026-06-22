package rule

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lumbung-fs/core/database"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	originModel "lumbung-fs/core/modules/origin/model"
	ruleModel "lumbung-fs/core/modules/rule/model"
	"lumbung-fs/core/variables"
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
		IsCompress          bool   `json:"is_compress"`
		CompressLevel       int    `json:"compress_level"`
		IsEncrypt           bool   `json:"is_encrypt"`
		EncryptionKey       string `json:"encryption_key"`
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

	level := input.CompressLevel
	if level == 0 {
		level = 3
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
		IsCompress:          input.IsCompress,
		CompressLevel:       level,
		IsEncrypt:           input.IsEncrypt,
		EncryptionKey:       strings.TrimSpace(input.EncryptionKey),
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if rule.IsEncrypt {
		if err := ProcessRuleEncryptionTransition(rule.OriginID, rule.Path, false, "", true, rule.EncryptionKey); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Rule created but file encryption transition failed: "+err.Error())
			return
		}
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
		IsCompress          bool   `json:"is_compress"`
		CompressLevel       int    `json:"compress_level"`
		IsEncrypt           bool   `json:"is_encrypt"`
		EncryptionKey       string `json:"encryption_key"`
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

	// Keep copy of old values for transition logic
	oldPath := rule.Path
	oldIsEncrypt := rule.IsEncrypt
	oldKey := rule.EncryptionKey

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
	rule.IsCompress = input.IsCompress
	level := input.CompressLevel
	if level == 0 {
		level = 3
	}
	rule.CompressLevel = level
	rule.IsEncrypt = input.IsEncrypt
	rule.EncryptionKey = strings.TrimSpace(input.EncryptionKey)

	// Apply file transition
	newPath := rule.Path
	newIsEncrypt := rule.IsEncrypt
	newKey := rule.EncryptionKey

	if oldPath != newPath {
		// Decrypt old path if it was encrypted
		if oldIsEncrypt {
			if err := ProcessRuleEncryptionTransition(rule.OriginID, oldPath, true, oldKey, false, ""); err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to decrypt old path: "+err.Error())
				return
			}
		}
		// Encrypt new path if it should be encrypted
		if newIsEncrypt {
			if err := ProcessRuleEncryptionTransition(rule.OriginID, newPath, false, "", true, newKey); err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to encrypt new path: "+err.Error())
				return
			}
		}
	} else {
		// Same path, transition key or encryption state
		if err := ProcessRuleEncryptionTransition(rule.OriginID, rule.Path, oldIsEncrypt, oldKey, newIsEncrypt, newKey); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to apply encryption transition: "+err.Error())
			return
		}
	}

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

// ProcessRuleEncryptionTransition handles transition of encryption (encrypting, decrypting or re-keying)
// recursively for all files under the path specified by the rule.
func ProcessRuleEncryptionTransition(originID string, path string, oldIsEncrypt bool, oldKey string, newIsEncrypt bool, newKey string) error {
	var origin originModel.Origin
	if err := database.DB.Where("id = ?", originID).First(&origin).Error; err != nil {
		return err
	}

	originSnake := variables.DomainToSnake(origin.Domain)
	targetDir, err := fileExplorer.SecurePath(filepath.Join(originSnake, path))
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return nil // No files to process
	}

	var action string // "encrypt", "decrypt", "rekey", "none"
	if !oldIsEncrypt && newIsEncrypt {
		action = "encrypt"
	} else if oldIsEncrypt && !newIsEncrypt {
		action = "decrypt"
	} else if oldIsEncrypt && newIsEncrypt && oldKey != newKey {
		action = "rekey"
	} else {
		return nil
	}

	var oldDerivedKey, newDerivedKey []byte
	if action == "decrypt" || action == "rekey" {
		oldDerivedKey, err = variables.DeriveKey(oldKey)
		if err != nil {
			return err
		}
	}
	if action == "encrypt" || action == "rekey" {
		newDerivedKey, err = variables.DeriveKey(newKey)
		if err != nil {
			return err
		}
	}

	err = filepath.WalkDir(targetDir, func(filePath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		var processed []byte
		switch action {
		case "encrypt":
			processed, err = variables.EncryptAESGCM(data, newDerivedKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt file %s: %w", filePath, err)
			}
		case "decrypt":
			processed, err = variables.DecryptAESGCM(data, oldDerivedKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt file %s: %w", filePath, err)
			}
		case "rekey":
			// Decrypt with old key first
			decrypted, err := variables.DecryptAESGCM(data, oldDerivedKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt file %s during rekey: %w", filePath, err)
			}
			// Encrypt with new key
			processed, err = variables.EncryptAESGCM(decrypted, newDerivedKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt file %s during rekey: %w", filePath, err)
			}
		}

		if err := os.WriteFile(filePath, processed, 0644); err != nil {
			return err
		}

		return nil
	})

	return err
}
