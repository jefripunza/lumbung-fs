package core

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lumbung-fs/core/database"
	"lumbung-fs/core/middleware"
	"lumbung-fs/core/modules"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	originModel "lumbung-fs/core/modules/origin/model"
	ruleModel "lumbung-fs/core/modules/rule/model"
	"lumbung-fs/core/variables"

	"github.com/google/uuid"
)

// Global secret hash constant for fallback
const defaultMD5Hash = "e10adc3949ba59abbe56e057f20f883e" // MD5 of "123456"

// Start initializes the database, checks files, registers routes, and starts the server
func ServerStart() {
	log.Println("Initializing LumbungFS Core Service...")

	// 1. Establish DB Connection
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// 2. Run Database AutoMigrations
	err = db.AutoMigrate(
		&originModel.Origin{},
		&originModel.UnknownOrigin{},
		&ruleModel.Rule{},
	)
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}
	log.Println("Database migration completed successfully.")

	// 3. Initialize credentials file
	initCredentialsFile()

	// 4. Create standard http ServeMux
	mux := http.NewServeMux()

	// 5. Auth endpoint
	mux.HandleFunc("/api/auth/login", loginHandler)

	// 6. Register Module Routes
	modules.RegisterAllRoutes(mux)

	// 7. Register Client File Serving Route
	mux.HandleFunc("/file/", clientFileHandler)

	// 8. Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"lumbung-fs"}`))
	})

	// 9. Wrap administrative and client routes with appropriate middlewares
	// Note: CORSAndOriginHandler handles origin validation & dashboard CORS.
	// AdminAuth is applied individually inside RegisterAllRoutes or via a sub-router pattern.
	// For standard ServeMux, we can wrap the entire multiplexer with CORSAndOriginHandler
	// and then check JWT inside /api/... handlers except login.
	mainHandler := middleware.CORSAndOriginHandler(adminAuthWrapper(mux))

	port := 8080
	log.Printf("LumbungFS server successfully started on port :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mainHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// adminAuthWrapper intercepts /api/ requests (excluding /api/auth/login) and applies JWT checks
func adminAuthWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") && path != "/api/auth/login" {
			// Wrap with AdminAuth middleware logic
			middleware.AdminAuth(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// initCredentialsFile ensures password.txt exists with default hash if missing
func initCredentialsFile() {
	passFile := variables.GetPasswordFilePath()
	if _, err := os.Stat(passFile); os.IsNotExist(err) {
		log.Println("Credentials file password.txt is missing. Creating with default credentials (admin:123456)...")
		err := os.WriteFile(passFile, []byte(defaultMD5Hash), 0644)
		if err != nil {
			log.Printf("Warning: Failed to create default password.txt: %v", err)
		}
	}
}

// verifyMD5 compares a plaintext string to an MD5 hash (case-insensitive)
func verifyMD5(plaintext string, hash string) bool {
	hasher := md5.New()
	hasher.Write([]byte(plaintext))
	sum := hasher.Sum(nil)
	computed := hex.EncodeToString(sum)
	return strings.ToLower(computed) == strings.ToLower(hash)
}

// loginHandler processes admin login credentials and returns JWT
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	passFile := variables.GetPasswordFilePath()
	hashBytes, err := os.ReadFile(passFile)
	if err != nil {
		http.Error(w, "Internal server error: unable to verify credentials", http.StatusInternalServerError)
		return
	}

	storedHash := strings.TrimSpace(string(hashBytes))

	// Allow verification of:
	// 1. Plain MD5 of password
	// 2. MD5 of username:password
	// 3. Special fallback case (admin/admin -> accepts 123456 or admin)
	valid := false
	if verifyMD5(credentials.Password, storedHash) {
		valid = true
	} else if verifyMD5(credentials.Username+":"+credentials.Password, storedHash) {
		valid = true
	} else if storedHash == defaultMD5Hash {
		// Default password.txt contains MD5 of "123456"
		if credentials.Username == "admin" && (credentials.Password == "123456" || credentials.Password == "admin") {
			valid = true
		}
	}

	if !valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid username or password"}`))
		return
	}

	token, err := middleware.GenerateJWT(credentials.Username)
	if err != nil {
		http.Error(w, "Internal server error: token generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

// clientFileHandler processes GET and POST/PUT requests to /file/... for custom domains
func clientFileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract subpath (e.g. /file/user/ktp/photo.png -> user/ktp/photo.png)
	subpath := r.URL.Path[len("/file/"):]
	if subpath == "" {
		http.Error(w, "Bad request: path is required", http.StatusBadRequest)
		return
	}

	// Resolve origin domain from request
	originHeader := r.Header.Get("Origin")
	hostHeader := r.Header.Get("Host")
	requestDomain := ""
	if originHeader != "" {
		requestDomain = middleware.ParseDomain(originHeader)
	} else {
		requestDomain = middleware.ParseDomain(hostHeader)
	}

	// Find origin record in DB to get ID and snake_case name
	var origin originModel.Origin
	if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
		http.Error(w, "Forbidden: Invalid origin", http.StatusForbidden)
		return
	}

	originSnake := strings.ReplaceAll(strings.ReplaceAll(origin.Domain, ".", "_"), "-", "_")

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
		allowed, fallbackURL, status, err := middleware.EvaluatePathRules(r, origin.ID, subpath, info.Size(), ext)
		if !allowed {
			if fallbackURL != "" {
				http.Redirect(w, r, fallbackURL, http.StatusFound)
				return
			}
			errMsg := "Access denied by path rules"
			if err != nil {
				errMsg = fmt.Sprintf("Access denied: %v", err)
			}
			http.Error(w, errMsg, status)
			return
		}

		// 3. Serve File
		http.ServeFile(w, r, targetPath)

	case http.MethodPost, http.MethodPut:
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

		out, err := os.Create(targetFilePath)
		if err != nil {
			http.Error(w, "Internal server error: write failed", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, "Internal server error: copy failed", http.StatusInternalServerError)
			return
		}

		relPath, _ := filepath.Rel(variables.BucketDir, targetFilePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "File uploaded successfully",
			"url":      fmt.Sprintf("/file/%s/%s", strings.Trim(subpath, "/"), uniqueName),
			"filename": uniqueName,
			"path":     relPath,
			"size":     header.Size,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
