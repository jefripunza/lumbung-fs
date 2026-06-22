package file_explorer

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumbung-fs/core/database"
	"lumbung-fs/core/functions"
	"lumbung-fs/core/middlewares"
	explorerModel "lumbung-fs/core/modules/file-explorer/model"
	originModel "lumbung-fs/core/modules/origin/model"
	"lumbung-fs/core/variables"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileItem represents a file or folder in the explorer
type FileItem struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"` // Relative to bucket root
	IsDir      bool      `json:"is_dir"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

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

// SecurePath validates and returns the absolute target path within the bucket directory
func SecurePath(subpath string) (string, error) {
	// Clean the subpath
	cleanedSub := filepath.Clean(subpath)
	if cleanedSub == "." || cleanedSub == "/" {
		cleanedSub = ""
	}

	// Prevent path traversal
	if strings.HasPrefix(cleanedSub, "..") || strings.Contains(cleanedSub, "../") || strings.Contains(cleanedSub, "..\\") {
		return "", fmt.Errorf("invalid path traversal attempt")
	}

	bucketAbs, err := filepath.Abs(variables.BucketDir)
	if err != nil {
		return "", err
	}

	targetAbs, err := filepath.Abs(filepath.Join(variables.BucketDir, cleanedSub))
	if err != nil {
		return "", err
	}

	// Verify the target path remains inside the bucket directory
	if !strings.HasPrefix(targetAbs, bucketAbs) {
		return "", fmt.Errorf("access denied: path outside bucket directory")
	}

	return targetAbs, nil
}

// ListItems handles GET /api/explorer/list
func ListItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	subpath := r.URL.Query().Get("path")
	targetDir, err := SecurePath(subpath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure the directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := []FileItem{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Don't list database and password file in root directory list
		if subpath == "" || subpath == "/" {
			if entry.Name() == variables.DatabaseName || entry.Name() == variables.PasswordFile {
				continue
			}
		}

		bucketAbs, _ := filepath.Abs(variables.BucketDir)
		relPath, _ := filepath.Rel(bucketAbs, filepath.Join(targetDir, entry.Name()))

		items = append(items, FileItem{
			Name:       entry.Name(),
			Path:       relPath,
			IsDir:      entry.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
		})
	}

	respondWithJSON(w, http.StatusOK, items)
}

// CreateFolder handles POST /api/explorer/folder
func CreateFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		Path string `json:"path"` // Parent folder path relative to bucket
		Name string `json:"name"` // Folder name to create
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	folderName := strings.TrimSpace(input.Name)
	if folderName == "" {
		respondWithError(w, http.StatusBadRequest, "Folder name is required")
		return
	}

	targetParent, err := SecurePath(input.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	targetDir := filepath.Join(targetParent, folderName)

	// Double check final directory path is secure
	if _, err := SecurePath(filepath.Join(input.Path, folderName)); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid folder path")
		return
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	bucketAbs, _ := filepath.Abs(variables.BucketDir)
	relPath, _ := filepath.Rel(bucketAbs, targetDir)
	respondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "Folder created successfully",
		"path":    relPath,
	})
}

// UploadFile handles POST /api/explorer/upload
func UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse multipart form (32MB max memory)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	targetSub := r.FormValue("path") // Destination directory relative to bucket
	targetDir, err := SecurePath(targetSub)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File parameter is required")
		return
	}
	defer file.Close()

	// Ensure the parent directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate UUIDv7 name, preserving extension
	id, err := uuid.NewV7()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate unique filename")
		return
	}

	ext := filepath.Ext(header.Filename)
	uniqueName := id.String() + ext
	targetFilePath := filepath.Join(targetDir, uniqueName)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read upload file")
		return
	}

	var compress, encrypt bool
	var compressLevel int
	var encryptionKey string
	if origin, evalPath := ResolveOriginFromSubpath(targetSub); origin != nil {
		if matchedRule, err := middlewares.FindMatchingRule(origin.ID, evalPath); err == nil && matchedRule != nil {
			compress = matchedRule.IsCompress
			compressLevel = matchedRule.CompressLevel
			encrypt = matchedRule.IsEncrypt
			encryptionKey = matchedRule.EncryptionKey
		}
	}

	processedBytes, err := functions.ProcessUploadData(fileBytes, compress, compressLevel, encrypt, encryptionKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := os.WriteFile(targetFilePath, processedBytes, 0644); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	bucketAbs, _ := filepath.Abs(variables.BucketDir)
	relPath, _ := filepath.Rel(bucketAbs, targetFilePath)
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":           "File uploaded successfully",
		"original_filename": header.Filename,
		"filename":          uniqueName,
		"path":              relPath,
		"size":              header.Size,
	})
}

// DownloadFile handles GET /api/explorer/download
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	subpath := r.URL.Query().Get("path")
	if subpath == "" {
		respondWithError(w, http.StatusBadRequest, "Path parameter is required")
		return
	}

	filePath, err := SecurePath(subpath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			respondWithError(w, http.StatusNotFound, "File not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if info.IsDir() {
		respondWithError(w, http.StatusBadRequest, "Cannot download a directory")
		return
	}

	// Check if there is a matching rule for this path to apply decryption/decompression
	var compress, encrypt bool
	var compressLevel int
	var encryptionKey string
	if origin, evalPath := ResolveOriginFromSubpath(subpath); origin != nil {
		if matchedRule, err := middlewares.FindMatchingRule(origin.ID, evalPath); err == nil && matchedRule != nil {
			compress = matchedRule.IsCompress
			compressLevel = matchedRule.CompressLevel
			encrypt = matchedRule.IsEncrypt
			encryptionKey = matchedRule.EncryptionKey
		}
	}

	if compress || encrypt {
		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "File not found")
			return
		}

		processedBytes, err := functions.ProcessDownloadData(rawBytes, compress, compressLevel, encrypt, encryptionKey)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Decompression/Decryption failed: "+err.Error())
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(processedBytes)))
		w.Write(processedBytes)
	} else {
		// Set headers for download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

		http.ServeFile(w, r, filePath)
	}
}

// DeleteItem handles DELETE /api/explorer/delete
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	subpath := r.URL.Query().Get("path")
	if subpath == "" {
		respondWithError(w, http.StatusBadRequest, "Path parameter is required")
		return
	}

	// Enforce that they cannot delete database or password file
	cleanedSub := filepath.Clean(subpath)
	if cleanedSub == variables.DatabaseName || cleanedSub == variables.PasswordFile || cleanedSub == "." || cleanedSub == "/" {
		respondWithError(w, http.StatusForbidden, "Cannot delete system files or bucket root")
		return
	}

	targetPath, err := SecurePath(subpath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify path exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		respondWithError(w, http.StatusNotFound, "Path not found")
		return
	}

	if err := os.RemoveAll(targetPath); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}

// StartPresignedURLCleanupWorker periodically deletes presigned URLs older than 1 minute
func StartPresignedURLCleanupWorker(db *gorm.DB) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		cutoff := time.Now().Add(-1 * time.Minute)
		if err := db.Where("created_at < ?", cutoff).Delete(&explorerModel.PresignedURL{}).Error; err != nil {
			log.Printf("Error cleaning up expired presigned URLs: %v", err)
		}
	}
}

// GeneratePresignedURLAdmin handles POST /api/explorer/presigned-url (for admin dashboard UI)
func GeneratePresignedURLAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		OriginID string `json:"origin_id"`
		Path     string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if input.OriginID == "" {
		respondWithError(w, http.StatusBadRequest, "Origin ID is required")
		return
	}

	var origin originModel.Origin
	if err := database.DB.Where("id = ?", input.OriginID).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "Origin not found")
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

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	urlStr := fmt.Sprintf("%s://%s/upload?token=%s", scheme, r.Host, token)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"url":        urlStr,
		"token":      token,
		"path":       presigned.Path,
		"expires_at": presigned.CreatedAt.Add(1 * time.Minute),
	})
}

// ResolveOriginFromSubpath helper extracts origin and path relative to the origin folder
func ResolveOriginFromSubpath(subpath string) (*originModel.Origin, string) {
	cleanSub := strings.Trim(filepath.Clean(subpath), "/")
	parts := strings.Split(cleanSub, "/")
	if len(parts) == 0 || parts[0] == "" {
		return nil, ""
	}

	originSnake := parts[0]
	evalPath := strings.Join(parts[1:], "/")

	var origins []originModel.Origin
	if err := database.DB.Find(&origins).Error; err == nil {
		for _, o := range origins {
			if variables.DomainToSnake(o.Domain) == originSnake {
				return &o, evalPath
			}
		}
	}

	// Fallback to checking WEB_DASHBOARD_ORIGIN env
	dashboardOrigin := os.Getenv("WEB_DASHBOARD_ORIGIN")
	if dashboardOrigin != "" {
		parsed := middlewares.ParseDomain(dashboardOrigin)
		if variables.DomainToSnake(parsed) == originSnake {
			return &originModel.Origin{
				Domain: parsed,
			}, evalPath
		}
	}

	return nil, ""
}
