package file

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lumbung-fs/core/database"
	"lumbung-fs/core/functions"
	"lumbung-fs/core/middlewares"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	originModel "lumbung-fs/core/modules/origin/model"
	"lumbung-fs/core/templates"
	"lumbung-fs/core/variables"

	"github.com/google/uuid"
)

// ClientFileHandler processes GET and POST/PUT requests to /file/... for custom domains
func ClientFileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract subpath (e.g. /file/user/ktp/photo.png -> user/ktp/photo.png)
	subpath := r.URL.Path[len("/file/"):]
	if subpath == "" {
		http.Error(w, "Bad request: path is required", http.StatusBadRequest)
		return
	}

	// Try lookup by API Key first if present
	var origin originModel.Origin
	var resolved bool
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
			resolved = true
		}
	}

	if !resolved {
		requestDomain := middlewares.ResolveDomain(r)
		if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
			dashboardOrigin := os.Getenv("WEB_DASHBOARD_ORIGIN")
			if dashboardOrigin != "" && requestDomain == middlewares.ParseDomain(dashboardOrigin) {
				origin = originModel.Origin{
					Domain: requestDomain,
				}
			} else {
				http.Error(w, "Forbidden: Invalid origin", http.StatusForbidden)
				return
			}
		}
	}

	originSnake := variables.DomainToSnake(origin.Domain)

	switch r.Method {
	case http.MethodGet:
		// 1. Locate file on disk
		targetPath, err := fileExplorer.SecurePath(filepath.Join(originSnake, subpath))
		if err != nil {
			http.Error(w, "Bad request: invalid path", http.StatusBadRequest)
			return
		}

		info, err := os.Stat(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			http.Error(w, "Forbidden: Cannot read directory", http.StatusForbidden)
			return
		}

		// 2. Evaluate rules (Extension and External Auth checks)
		ext := filepath.Ext(targetPath)
		allowed, fallbackURL, status, err := middlewares.EvaluatePathRules(r, origin.ID, subpath, info.Size(), ext)
		if !allowed {
			if fallbackURL != "" {
				http.Redirect(w, r, fallbackURL, http.StatusFound)
				return
			}
			title := "Access Denied"
			errMsg := "You do not have permission to access this file."
			var details string
			if err != nil {
				details = err.Error()
				if strings.Contains(strings.ToLower(details), "dial tcp") ||
					strings.Contains(strings.ToLower(details), "connect: connection refused") ||
					strings.Contains(strings.ToLower(details), "no such host") {
					title = "Connection Failed"
					errMsg = "LumbungFS was unable to contact the external validation service."
				}
			}
			renderAccessDenied(w, r, title, errMsg, details, status)
			return
		}

		// 3. Serve File
		var compress, encrypt bool
		var compressLevel int
		var encryptionKey string
		if matchedRule, err := middlewares.FindMatchingRule(origin.ID, subpath); err == nil && matchedRule != nil {
			compress = matchedRule.IsCompress
			compressLevel = matchedRule.CompressLevel
			encrypt = matchedRule.IsEncrypt
			encryptionKey = matchedRule.EncryptionKey
		}

		if compress || encrypt {
			rawBytes, err := os.ReadFile(targetPath)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			processedBytes, err := functions.ProcessDownloadData(rawBytes, compress, compressLevel, encrypt, encryptionKey)
			if err != nil {
				http.Error(w, "Decompression/Decryption failed: "+err.Error(), http.StatusInternalServerError)
				return
			}

			contentType := http.DetectContentType(processedBytes)
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(processedBytes)))
			w.Write(processedBytes)
		} else {
			http.ServeFile(w, r, targetPath)
		}

	case http.MethodPost, http.MethodPut:
		// Secure backend uploads by verifying the origin API key
		if !checkApiKey(r, origin.ApiKey) {
			http.Error(w, "Unauthorized: Invalid or missing API key", http.StatusUnauthorized)
			return
		}

		// Handle direct file uploads from custom clients/domains
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "File parameter is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := filepath.Ext(header.Filename)

		// Evaluate path rules (including size, extension, and auth checks)
		allowed, fallbackURL, status, err := middlewares.EvaluatePathRules(r, origin.ID, subpath, header.Size, ext)
		if !allowed {
			if fallbackURL != "" {
				http.Redirect(w, r, fallbackURL, http.StatusFound)
				return
			}
			errMsg := "Upload blocked by path rules"
			if err != nil {
				errMsg = fmt.Sprintf("Upload blocked: %v", err)
			}
			http.Error(w, errMsg, status)
			return
		}

		// Save file under bucket/origin_snakecase/subpath/uuid.ext
		targetDir, err := fileExplorer.SecurePath(filepath.Join(originSnake, subpath))
		if err != nil {
			http.Error(w, "Bad request: invalid path", http.StatusBadRequest)
			return
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			http.Error(w, "Internal server error: ID generation failed", http.StatusInternalServerError)
			return
		}

		uniqueName := id.String() + ext
		targetFilePath := filepath.Join(targetDir, uniqueName)

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read upload file", http.StatusInternalServerError)
			return
		}

		var compress, encrypt bool
		var compressLevel int
		var encryptionKey string
		if matchedRule, err := middlewares.FindMatchingRule(origin.ID, subpath); err == nil && matchedRule != nil {
			compress = matchedRule.IsCompress
			compressLevel = matchedRule.CompressLevel
			encrypt = matchedRule.IsEncrypt
			encryptionKey = matchedRule.EncryptionKey
		}

		processedBytes, err := functions.ProcessUploadData(fileBytes, compress, compressLevel, encrypt, encryptionKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(targetFilePath, processedBytes, 0644); err != nil {
			http.Error(w, "Internal server error: write failed", http.StatusInternalServerError)
			return
		}

		bucketAbs, _ := filepath.Abs(variables.BucketDir)
		relPath, _ := filepath.Rel(bucketAbs, targetFilePath)

		var fileURL string
		subpathClean := strings.Trim(subpath, "/")
		if subpathClean != "" {
			fileURL = fmt.Sprintf("/file/%s/%s", subpathClean, uniqueName)
		} else {
			fileURL = fmt.Sprintf("/file/%s", uniqueName)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "File uploaded successfully",
			"url":      fileURL,
			"filename": uniqueName,
			"path":     relPath,
			"size":     header.Size,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

func renderAccessDenied(w http.ResponseWriter, r *http.Request, title, errorMessage, details string, status int) {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(accept), "application/json") ||
		strings.Contains(strings.ToLower(contentType), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   errorMessage,
			"details": details,
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	tmpl, err := template.New("access_denied").Parse(templates.AccessDeniedTemplate)
	if err != nil {
		http.Error(w, errorMessage, status)
		return
	}

	data := struct {
		Title        string
		ErrorMessage string
		Details      string
	}{
		Title:        title,
		ErrorMessage: errorMessage,
		Details:      details,
	}

	_ = tmpl.Execute(w, data)
}
