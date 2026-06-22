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

// GeneratePresignedURLRest handles POST /presigned-url (for external backend clients via API Key)
func GeneratePresignedURLRest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	requestDomain := middleware.ResolveDomain(r)

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

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			// path defaults to empty if payload is empty/missing
		}
	} else {
		input.Path = r.FormValue("path")
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

// UploadHandler handles both GET/POST for tokenized presigned uploads, and POST for standard API key REST uploads
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// --- Case A: Presigned Token Upload Flow ---
	if token != "" {
		var presigned explorerModel.PresignedURL
		if err := database.DB.Where("token = ?", token).First(&presigned).Error; err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired presigned token")
			return
		}

		if time.Since(presigned.CreatedAt) > 1*time.Minute {
			database.DB.Delete(&presigned)
			respondWithError(w, http.StatusGone, "Presigned token has expired")
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

			html := strings.ReplaceAll(uploaderHTMLTemplate, "{{.Domain}}", origin.Domain)
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

	requestDomain := middleware.ResolveDomain(r)

	var origin originModel.Origin
	if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
		respondWithError(w, http.StatusForbidden, "Forbidden: Invalid origin")
		return
	}

	if !checkApiKey(r, origin.ApiKey) {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing API key")
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

	subpath := strings.Trim(r.FormValue("path"), "/")
	ext := filepath.Ext(header.Filename)

	allowed, fallbackURL, status, err := middleware.EvaluatePathRules(r, origin.ID, subpath, header.Size, ext)
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
	targetDir, err := SecurePath(filepath.Join(originSnake, subpath))
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

	bucketAbs, _ := filepath.Abs(variables.BucketDir)
	relPath, _ := filepath.Rel(bucketAbs, targetFilePath)

	var fileURL string
	subpathClean := strings.Trim(subpath, "/")
	if subpathClean != "" {
		fileURL = fmt.Sprintf("https://%s/file/%s/%s", origin.Domain, subpathClean, uniqueName)
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

const uploaderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Upload File - LumbungFS</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Outfit:wght@500;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --color-deep-forest: #0c322c;
            --color-bone-white: #faf9f5;
            --color-forest-ink: #0e2a22;
            --color-slate-smoke: #62756f;
            --color-lichen: #cad3d2;
            --color-moss: #5c8e75;
            --color-deep-fern: #2d4f45;
            --radius-xl: 16px;
            --radius-md: 8px;
        }
        
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', -apple-system, sans-serif;
            background: var(--color-deep-forest);
            color: var(--color-forest-ink);
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 20px;
        }

        .upload-card {
            background: var(--color-bone-white);
            border-radius: var(--radius-xl);
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
            width: 100%;
            max-width: 480px;
            padding: 40px 32px;
            text-align: center;
            animation: slideUp 0.3s ease-out;
        }

        @keyframes slideUp {
            from { opacity: 0; transform: translateY(12px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .header {
            margin-bottom: 28px;
        }

        .title {
            font-family: 'Outfit', sans-serif;
            font-size: 24px;
            font-weight: 600;
            color: var(--color-deep-fern);
            margin-bottom: 8px;
        }

        .subtitle {
            font-size: 14px;
            color: var(--color-slate-smoke);
            line-height: 1.5;
        }

        .origin-badge {
            display: inline-block;
            background: rgba(92, 142, 117, 0.12);
            color: var(--color-deep-fern);
            padding: 4px 8px;
            border-radius: var(--radius-md);
            font-size: 12px;
            font-weight: 500;
            margin-top: 10px;
            border: 0.5px solid var(--color-lichen);
        }

        .dropzone {
            border: 2px dashed var(--color-lichen);
            background: #fdfdfb;
            border-radius: var(--radius-xl);
            padding: 40px 20px;
            cursor: pointer;
            transition: all 0.2s ease;
            position: relative;
            margin-bottom: 24px;
        }

        .dropzone:hover, .dropzone.dragover {
            border-color: var(--color-moss);
            background: #f7f9f7;
        }

        .dropzone-icon {
            font-size: 40px;
            margin-bottom: 16px;
            display: inline-block;
        }

        .dropzone-text {
            font-size: 14px;
            font-weight: 500;
            color: var(--color-forest-ink);
            margin-bottom: 6px;
        }

        .dropzone-subtext {
            font-size: 12px;
            color: var(--color-slate-smoke);
        }

        .file-input {
            display: none;
        }

        .progress-container {
            display: none;
            margin-bottom: 24px;
            text-align: left;
        }

        .progress-label {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            font-weight: 500;
            color: var(--color-forest-ink);
            margin-bottom: 6px;
        }

        .progress-track {
            height: 8px;
            background: var(--color-lichen);
            border-radius: 4px;
            overflow: hidden;
        }

        .progress-bar {
            height: 100%;
            width: 0%;
            background: linear-gradient(90deg, var(--color-moss), var(--color-deep-fern));
            transition: width 0.1s ease;
            border-radius: 4px;
        }

        .error-message {
            display: none;
            background: #fdf2f2;
            border: 0.5px solid #f8b4b4;
            color: #9b1c1c;
            padding: 12px;
            border-radius: var(--radius-md);
            font-size: 13px;
            margin-bottom: 24px;
            text-align: left;
            line-height: 1.4;
        }

        .success-container {
            display: none;
            animation: fadeIn 0.2s ease-out;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        .success-icon {
            font-size: 48px;
            margin-bottom: 16px;
            display: inline-block;
        }

        .success-title {
            font-family: 'Outfit', sans-serif;
            font-size: 20px;
            font-weight: 600;
            color: #1e3a2f;
            margin-bottom: 8px;
        }

        .success-desc {
            font-size: 13.5px;
            color: var(--color-slate-smoke);
            margin-bottom: 20px;
        }

        .url-box {
            display: flex;
            background: #f3f2ee;
            border: 0.5px solid var(--color-lichen);
            border-radius: var(--radius-md);
            padding: 10px 12px;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 24px;
            text-align: left;
        }

        .url-text {
            font-size: 13px;
            color: var(--color-forest-ink);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-right: 12px;
            flex: 1;
        }

        .copy-btn {
            background: var(--color-moss);
            border: none;
            border-radius: 6px;
            color: white;
            padding: 6px 12px;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.15s ease;
            flex-shrink: 0;
        }

        .copy-btn:hover {
            background: var(--color-deep-fern);
        }

        .btn-upload-another {
            display: inline-block;
            font-size: 13px;
            color: var(--color-deep-fern);
            text-decoration: underline;
            background: none;
            border: none;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <div class="upload-card">
        <div id="main-flow">
            <div class="header">
                <h1 class="title">Upload File</h1>
                <p class="subtitle">Upload a file directly to the storage bucket using your temporary 1-minute presigned token</p>
                <div class="origin-badge">🌐 {{.Domain}}{{.PathSuffix}}</div>
            </div>

            <div id="error-box" class="error-message"></div>

            <div class="dropzone" id="dropzone">
                <span class="dropzone-icon">📥</span>
                <p class="dropzone-text">Drag & drop your file here</p>
                <p class="dropzone-subtext">or click to browse from device</p>
                <input type="file" id="file-input" class="file-input">
            </div>

            <div class="progress-container" id="progress-container">
                <div class="progress-label">
                    <span id="progress-filename">Uploading...</span>
                    <span id="progress-percent">0%</span>
                </div>
                <div class="progress-track">
                    <div class="progress-bar" id="progress-bar"></div>
                </div>
            </div>
        </div>

        <div id="success-flow" class="success-container">
            <span class="success-icon">🎉</span>
            <h2 class="success-title">Upload Successful!</h2>
            <p class="success-desc">Your file has been uploaded and stored securely. You can copy the public access URL below:</p>
            
            <div class="url-box">
                <span class="url-text" id="url-text">https://example.com/file/photo.jpg</span>
                <button type="button" class="copy-btn" id="copy-btn">Copy Link</button>
            </div>
        </div>
    </div>

    <script>
        const dropzone = document.getElementById('dropzone');
        const fileInput = document.getElementById('file-input');
        const progressContainer = document.getElementById('progress-container');
        const progressBar = document.getElementById('progress-bar');
        const progressPercent = document.getElementById('progress-percent');
        const progressFilename = document.getElementById('progress-filename');
        const errorBox = document.getElementById('error-box');
        const mainFlow = document.getElementById('main-flow');
        const successFlow = document.getElementById('success-flow');
        const urlText = document.getElementById('url-text');
        const copyBtn = document.getElementById('copy-btn');

        // Extract token from query params
        const params = new URLSearchParams(window.location.search);
        const token = params.get('token');

        if (!token) {
            showError("Access Denied: Missing presigned upload token.");
            dropzone.style.pointerEvents = 'none';
            dropzone.style.opacity = '0.5';
        }

        // Drag events
        ['dragenter', 'dragover'].forEach(eventName => {
            dropzone.addEventListener(eventName, e => {
                e.preventDefault();
                dropzone.classList.add('dragover');
            }, false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            dropzone.addEventListener(eventName, e => {
                e.preventDefault();
                dropzone.classList.remove('dragover');
            }, false);
        });

        dropzone.addEventListener('drop', e => {
            const dt = e.dataTransfer;
            const files = dt.files;
            if (files.length) {
                handleFile(files[0]);
            }
        });

        dropzone.addEventListener('click', () => {
            fileInput.click();
        });

        fileInput.addEventListener('change', () => {
            if (fileInput.files.length) {
                handleFile(fileInput.files[0]);
            }
        });

        function showError(msg) {
            errorBox.innerText = msg;
            errorBox.style.display = 'block';
        }

        function handleFile(file) {
            if (!token) return;
            
            errorBox.style.display = 'none';
            dropzone.style.display = 'none';
            progressContainer.style.display = 'block';
            progressFilename.innerText = file.name;

            const formData = new FormData();
            formData.append('file', file);

            const xhr = new XMLHttpRequest();
            xhr.open('POST', "/upload?token=" + encodeURIComponent(token), true);

            xhr.upload.addEventListener('progress', e => {
                if (e.lengthComputable) {
                    const percent = Math.round((e.loaded * 100) / e.total);
                    progressBar.style.width = percent + '%';
                    progressPercent.innerText = percent + '%';
                }
            });

            xhr.onload = () => {
                if (xhr.status === 201) {
                    try {
                        const res = JSON.parse(xhr.responseText);
                        showSuccess(res.url);
                    } catch (err) {
                        showError("Failed to parse server response.");
                        resetFlow();
                    }
                } else {
                    let errMsg = "Upload failed.";
                    try {
                        const res = JSON.parse(xhr.responseText);
                        if (res.error) errMsg = res.error;
                    } catch (e) {}
                    showError(errMsg);
                    resetFlow();
                }
            };

            xhr.onerror = () => {
                showError("Network error occurred.");
                resetFlow();
            };

            xhr.send(formData);
        }

        function resetFlow() {
            progressContainer.style.display = 'none';
            dropzone.style.display = 'block';
        }

        function showSuccess(url) {
            mainFlow.style.display = 'none';
            successFlow.style.display = 'block';
            urlText.innerText = url;
        }

        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(urlText.innerText);
                copyBtn.innerText = "Copied!";
                setTimeout(() => {
                    copyBtn.innerText = "Copy Link";
                }, 2000);
            } catch (err) {
                console.error("Failed to copy", err);
            }
        });
    </script>
</body>
</html>`
