package file_explorer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumbung-fs/core/variables"
	"github.com/google/uuid"
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

		relPath, _ := filepath.Rel(variables.BucketDir, filepath.Join(targetDir, entry.Name()))

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

	relPath, _ := filepath.Rel(variables.BucketDir, targetDir)
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

	relPath, _ := filepath.Rel(variables.BucketDir, targetFilePath)
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
