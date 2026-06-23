package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumbung-fs/core/database"
	"lumbung-fs/core/functions"
	"lumbung-fs/core/middlewares"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	explorerModel "lumbung-fs/core/modules/file-explorer/model"
	originModel "lumbung-fs/core/modules/origin/model"
	"lumbung-fs/core/templates"
	"lumbung-fs/core/variables"

	"github.com/google/uuid"
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

// serveExpiredTokenError determines response format based on request headers and serves appropriate response
func serveExpiredTokenError(w http.ResponseWriter, r *http.Request, status int, message string) {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(accept), "application/json") ||
		strings.Contains(strings.ToLower(contentType), "application/json") ||
		strings.Contains(strings.ToLower(contentType), "multipart/") {
		respondWithError(w, status, message)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(templates.ExpiredTokenHTMLTemplate))
}

// checkApiKey validates that the request has an API Key matching the origin's ApiKey
func checkApiKey(r *http.Request, originApiKey string) bool {
	if originApiKey == "" {
		return false
	}
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				apiKey = parts[1]
			}
		}
	}
	return apiKey == originApiKey
}

// UploadHandler handles unified GET/POST for tokenized presigned uploads, and POST for standard API key REST uploads
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// --- Case A: Presigned Token Upload Flow ---
	if token != "" {
		var presigned explorerModel.PresignedURL
		if err := database.DB.Where("token = ?", token).First(&presigned).Error; err != nil {
			serveExpiredTokenError(w, r, http.StatusUnauthorized, "Invalid or expired presigned token")
			return
		}

		if time.Since(presigned.CreatedAt) > 1*time.Minute {
			database.DB.Delete(&presigned)
			serveExpiredTokenError(w, r, http.StatusGone, "Presigned token has expired")
			return
		}

		var origin originModel.Origin
		if err := database.DB.Where("id = ?", presigned.OriginID).First(&origin).Error; err != nil {
			respondWithError(w, http.StatusInternalServerError, "Origin not found for token")
			return
		}

		subpath := strings.Trim(presigned.Path, "/")
		originSnake := variables.DomainToSnake(origin.Domain)
		evalPath := subpath
		prefix := originSnake + "/"
		if strings.HasPrefix(subpath, prefix) {
			evalPath = subpath[len(prefix):]
		} else if subpath == originSnake {
			evalPath = ""
		}

		if r.Method == http.MethodGet {
			// Serve the beautiful uploader HTML page
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			pathSuffix := ""
			if evalPath != "" {
				pathSuffix = "/" + evalPath
			}

			html := strings.ReplaceAll(templates.UploaderHTMLTemplate, "{{.Domain}}", origin.Domain)
			html = strings.ReplaceAll(html, "{{.PathSuffix}}", pathSuffix)
			w.Write([]byte(html))
			return
		}

		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// Handle POST upload with Token
		// Delete token immediately upon start of upload to prevent replay attacks
		database.DB.Delete(&presigned)

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "File parameter is required")
			return
		}
		defer file.Close()

		ext := filepath.Ext(header.Filename)

		allowed, fallbackURL, status, err := middlewares.EvaluatePathRules(r, origin.ID, evalPath, header.Size, ext)
		if !allowed {
			if fallbackURL != "" {
				http.Redirect(w, r, fallbackURL, http.StatusFound)
				return
			}
			errMsg := "Upload blocked by path rules"
			if err != nil {
				errMsg = fmt.Sprintf("Upload blocked: %v", err)
			}
			respondWithError(w, status, errMsg)
			return
		}

		targetDir, err := fileExplorer.SecurePath(filepath.Join(originSnake, evalPath))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid path")
			return
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "ID generation failed")
			return
		}

		uniqueName := id.String() + ext
		targetFilePath := filepath.Join(targetDir, uniqueName)

		// Read file bytes for potential compression/encryption
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Read failed")
			return
		}

		// Check if compression/encryption path rules match
		var compress, encrypt bool
		var compressLevel int
		var encryptionKey string
		if matchedRule, err := middlewares.FindMatchingRule(origin.ID, evalPath); err == nil && matchedRule != nil {
			compress = matchedRule.IsCompress
			compressLevel = matchedRule.CompressLevel
			encrypt = matchedRule.IsEncrypt
			encryptionKey = matchedRule.EncryptionKey
		}

		processedBytes, err := functions.ProcessUploadData(fileBytes, compress, compressLevel, encrypt, encryptionKey)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := os.WriteFile(targetFilePath, processedBytes, 0644); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Write failed")
			return
		}

		bucketAbs, _ := filepath.Abs(variables.BucketDir)
		relPath, _ := filepath.Rel(bucketAbs, targetFilePath)

		var fileURL string
		evalPathClean := strings.Trim(evalPath, "/")
		if evalPathClean != "" {
			fileURL = fmt.Sprintf("https://%s/file/%s/%s", origin.Domain, evalPathClean, uniqueName)
		} else {
			fileURL = fmt.Sprintf("https://%s/file/%s", origin.Domain, uniqueName)
		}

		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"message":  "File uploaded successfully",
			"url":      fileURL,
			"filename": uniqueName,
			"path":     relPath,
			"size":     header.Size,
		})
		return
	}

	// --- Case B: REST Client API Key Upload Flow (no token) ---
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	originPtr, err := getOriginFromRequest(r)
	if err != nil {
		status := http.StatusForbidden
		if strings.HasPrefix(err.Error(), "Unauthorized:") {
			status = http.StatusUnauthorized
		}
		respondWithError(w, status, err.Error())
		return
	}
	origin := *originPtr

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File parameter is required")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)

	// Clean destination subpath relative to origin
	evalPath := strings.Trim(r.FormValue("path"), "/")

	allowed, fallbackURL, status, err := middlewares.EvaluatePathRules(r, origin.ID, evalPath, header.Size, ext)
	if !allowed {
		if fallbackURL != "" {
			http.Redirect(w, r, fallbackURL, http.StatusFound)
			return
		}
		errMsg := "Upload blocked by path rules"
		if err != nil {
			errMsg = fmt.Sprintf("Upload blocked: %v", err)
		}
		respondWithError(w, status, errMsg)
		return
	}

	originSnake := variables.DomainToSnake(origin.Domain)
	targetDir, err := fileExplorer.SecurePath(filepath.Join(originSnake, evalPath))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate unique filename")
		return
	}

	uniqueName := id.String() + ext
	targetFilePath := filepath.Join(targetDir, uniqueName)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Read failed")
		return
	}

	var compress, encrypt bool
	var compressLevel int
	var encryptionKey string
	if matchedRule, err := middlewares.FindMatchingRule(origin.ID, evalPath); err == nil && matchedRule != nil {
		compress = matchedRule.IsCompress
		compressLevel = matchedRule.CompressLevel
		encrypt = matchedRule.IsEncrypt
		encryptionKey = matchedRule.EncryptionKey
	}

	processedBytes, err := functions.ProcessUploadData(fileBytes, compress, compressLevel, encrypt, encryptionKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := os.WriteFile(targetFilePath, processedBytes, 0644); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Write failed")
		return
	}

	bucketAbs, _ := filepath.Abs(variables.BucketDir)
	relPath, _ := filepath.Rel(bucketAbs, targetFilePath)

	var fileURL string
	evalPathClean := strings.Trim(evalPath, "/")
	if evalPathClean != "" {
		fileURL = fmt.Sprintf("https://%s/file/%s/%s", origin.Domain, evalPathClean, uniqueName)
	} else {
		fileURL = fmt.Sprintf("https://%s/file/%s", origin.Domain, uniqueName)
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":           "File uploaded successfully",
		"original_filename": header.Filename,
		"filename":          uniqueName,
		"url":               fileURL,
		"path":              relPath,
		"size":              header.Size,
	})
}

// PrepareUploadHandler handles POST /upload/prepare (replaces REST presigned url generation)
func PrepareUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	originPtr, err := getOriginFromRequest(r)
	if err != nil {
		status := http.StatusForbidden
		if strings.HasPrefix(err.Error(), "Unauthorized:") {
			status = http.StatusUnauthorized
		}
		respondWithError(w, status, err.Error())
		return
	}
	origin := *originPtr

	var input struct {
		Path string `json:"path"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			// path defaults to empty if payload is empty/missing
		}
	} else {
		input.Path = r.FormValue("path")
	}

	// Verify that a matching path rule exists for the target upload path
	matchedRule, err := middlewares.FindMatchingRule(origin.ID, input.Path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if matchedRule == nil {
		respondWithError(w, http.StatusForbidden, "Forbidden: upload path not allowed (no matching path rule found)")
		return
	}

	tokenUUID, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	token := tokenUUID.String()

	presigned := explorerModel.PresignedURL{
		OriginID: origin.ID,
		Path:     input.Path,
		Token:    token,
	}

	if err := database.DB.Create(&presigned).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"presigned_url": fmt.Sprintf("https://%s/upload?token=%s", origin.Domain, token),
		"token":         token,
		"path":          presigned.Path,
		"expires_at":    presigned.CreatedAt.Add(1 * time.Minute),
	})
}

// getOriginFromRequest extracts the origin from the request, checking API key first
func getOriginFromRequest(r *http.Request) (*originModel.Origin, error) {
	var origin originModel.Origin
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				apiKey = parts[1]
			}
		}
	}

	if apiKey != "" {
		if err := database.DB.Where("api_key = ?", apiKey).First(&origin).Error; err == nil {
			return &origin, nil
		}
		return nil, fmt.Errorf("Unauthorized: Invalid API key")
	}

	requestDomain := middlewares.ResolveDomain(r)
	if requestDomain == "" {
		return nil, fmt.Errorf("Forbidden: Missing origin or host header")
	}

	if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
		dashboardOrigin := os.Getenv("WEB_DASHBOARD_ORIGIN")
		if dashboardOrigin != "" && requestDomain == middlewares.ParseDomain(dashboardOrigin) {
			return &originModel.Origin{
				Domain: requestDomain,
			}, nil
		}
		return nil, fmt.Errorf("Forbidden: Invalid origin")
	}

	return &origin, nil
}
