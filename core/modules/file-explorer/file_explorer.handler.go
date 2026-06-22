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
	"lumbung-fs/core/middleware"
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
		respondWithError(w, http.StatusNotFound, "Directory not found")
		return
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

	out, err := os.Create(targetFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
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

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	http.ServeFile(w, r, filePath)
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

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"url":        fmt.Sprintf("/upload/presigned?token=%s", token),
		"token":      token,
		"expires_at": presigned.CreatedAt.Add(1 * time.Minute),
	})
}

// GeneratePresignedURLRest handles POST /presigned-url (for external backend clients via API Key)
func GeneratePresignedURLRest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	originHeader := r.Header.Get("Origin")
	hostHeader := r.Host
	if hostHeader == "" {
		hostHeader = r.Header.Get("Host")
	}

	requestDomain := ""
	if originHeader != "" {
		requestDomain = middleware.ParseDomain(originHeader)
	} else {
		requestDomain = middleware.ParseDomain(hostHeader)
	}

	var origin originModel.Origin
	if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusForbidden, "Forbidden: Invalid origin")
		return
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

	if apiKey != origin.ApiKey || origin.ApiKey == "" {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing API key")
		return
	}

	var input struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// path defaults to empty if payload is empty/missing
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
		"presigned_url": fmt.Sprintf("https://%s/upload/presigned?token=%s", origin.Domain, token),
		"token":         token,
		"expires_at":    presigned.CreatedAt.Add(1 * time.Minute),
	})
}

// PresignedUploadHandler handles file uploads via public presigned URLs
func PresignedUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondWithError(w, http.StatusBadRequest, "Token parameter is required")
		return
	}

	var presigned explorerModel.PresignedURL
	if err := database.DB.Where("token = ?", token).First(&presigned).Error; err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired presigned token")
		return
	}

	if time.Since(presigned.CreatedAt) > 1 * time.Minute {
		database.DB.Delete(&presigned)
		respondWithError(w, http.StatusGone, "Presigned token has expired")
		return
	}

	var origin originModel.Origin
	if err := database.DB.Where("id = ?", presigned.OriginID).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Origin not found for token")
		return
	}

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

	subpath := strings.Trim(presigned.Path, "/")
	originSnake := strings.ReplaceAll(strings.ReplaceAll(origin.Domain, ".", "_"), "-", "_")

	evalPath := subpath
	prefix := originSnake + "/"
	if strings.HasPrefix(subpath, prefix) {
		evalPath = subpath[len(prefix):]
	} else if subpath == originSnake {
		evalPath = ""
	}

	allowed, fallbackURL, status, err := middleware.EvaluatePathRules(r, origin.ID, evalPath, header.Size, ext)
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

	targetDir, err := SecurePath(filepath.Join(originSnake, evalPath))
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

	out, err := os.Create(targetFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Write failed")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Copy failed")
		return
	}

	database.DB.Delete(&presigned)

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
}
