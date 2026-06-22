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
	"lumbung-fs/core/functions"
	"lumbung-fs/core/middlewares"
	"lumbung-fs/core/modules"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	fileExplorerModel "lumbung-fs/core/modules/file-explorer/model"
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
		&fileExplorerModel.PresignedURL{},
	)
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}
	log.Println("Database migration completed successfully.")

	// Start presigned URL cleanup worker
	go fileExplorer.StartPresignedURLCleanupWorker(db)

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
	mainHandler := middlewares.CORSAndOriginHandler(adminAuthWrapper(mux))

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
			// Wrap with AdminAuth middlewares logic
			middlewares.AdminAuth(next).ServeHTTP(w, r)
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

	token, err := middlewares.GenerateJWT(credentials.Username)
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
	requestDomain := middlewares.ResolveDomain(r)

	// Find origin record in DB to get ID and snake_case name
	var origin originModel.Origin
	if err := database.DB.Where("domain = ?", requestDomain).First(&origin).Error; err != nil {
		http.Error(w, "Forbidden: Invalid origin", http.StatusForbidden)
		return
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
			errMsg := "Access denied by path rules"
			if err != nil {
				errMsg = fmt.Sprintf("Access denied: %v", err)
			}
			http.Error(w, errMsg, status)
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
