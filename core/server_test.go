package core

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lumbung-fs/core/database"
	"lumbung-fs/core/middleware"
	"lumbung-fs/core/modules"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	fileExplorerModel "lumbung-fs/core/modules/file-explorer/model"
	originModel "lumbung-fs/core/modules/origin/model"
	ruleModel "lumbung-fs/core/modules/rule/model"
	"lumbung-fs/core/modules/rule"
	"lumbung-fs/core/modules/upload"
	"lumbung-fs/core/variables"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory database and auto-migrates models
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&originModel.Origin{},
		&originModel.UnknownOrigin{},
		&ruleModel.Rule{},
		&fileExplorerModel.PresignedURL{},
	)
	if err != nil {
		t.Fatalf("failed to migrate models: %v", err)
	}

	database.DB = db
	return db
}

func TestVerifyMD5(t *testing.T) {
	if !verifyMD5("123456", defaultMD5Hash) {
		t.Error("verifyMD5 failed to verify default hash")
	}
}

func TestParseDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "example.com"},
		{"https://sub.domain.com:8080/path", "sub.domain.com:8080"},
		{"domain.com", "domain.com"},
		{"localhost:3000", "localhost:3000"},
	}

	for _, tc := range tests {
		actual := middleware.ParseDomain(tc.input)
		if actual != tc.expected {
			t.Errorf("ParseDomain(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestResolveDomain(t *testing.T) {
	// Test Origin header
	req := httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("Origin", "http://origin-domain.com:8080")
	req.Host = "host-domain.com:9090"
	if actual := middleware.ResolveDomain(req); actual != "origin-domain.com:8080" {
		t.Errorf("ResolveDomain failed for Origin header: expected 'origin-domain.com:8080', got '%s'", actual)
	}

	// Test X-Forwarded-Host header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("X-Forwarded-Host", "forwarded-domain.com:7070, proxy-domain.com")
	req.Host = "host-domain.com:9090"
	if actual := middleware.ResolveDomain(req); actual != "forwarded-domain.com:7070" {
		t.Errorf("ResolveDomain failed for X-Forwarded-Host header: expected 'forwarded-domain.com:7070', got '%s'", actual)
	}

	// Test X-Original-Host header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("X-Original-Host", "original-domain.com:6060")
	req.Host = "host-domain.com:9090"
	if actual := middleware.ResolveDomain(req); actual != "original-domain.com:6060" {
		t.Errorf("ResolveDomain failed for X-Original-Host header: expected 'original-domain.com:6060', got '%s'", actual)
	}

	// Test Referer header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("Referer", "http://referer-domain.com:5050/some/path")
	req.Host = "host-domain.com:9090"
	if actual := middleware.ResolveDomain(req); actual != "referer-domain.com:5050" {
		t.Errorf("ResolveDomain failed for Referer header: expected 'referer-domain.com:5050', got '%s'", actual)
	}

	// Test Host header fallback
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Host = "host-domain.com:9090"
	if actual := middleware.ResolveDomain(req); actual != "host-domain.com:9090" {
		t.Errorf("ResolveDomain failed for Host header: expected 'host-domain.com:9090', got '%s'", actual)
	}
}


func TestDomainToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"localhost:5173", "localhost_5173"},
		{"foo...bar-baz", "foo_bar_baz"},
		{"sawang.tech", "sawang_tech"},
		{"--hello_world--", "hello_world"},
	}

	for _, tc := range tests {
		actual := variables.DomainToSnake(tc.input)
		if actual != tc.expected {
			t.Errorf("DomainToSnake(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestJWTGenerationAndAuth(t *testing.T) {
	setupTestDB(t)

	// Generate JWT
	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// Create test server wrapper
	handler := middleware.AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock success handler
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Case 1: Missing header
	req := httptest.NewRequest(http.MethodGet, "/api/origins", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized (401), got status: %d", w.Code)
	}

	// Case 2: Invalid token
	req = httptest.NewRequest(http.MethodGet, "/api/origins", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized (401), got status: %d", w.Code)
	}

	// Case 3: Valid token
	req = httptest.NewRequest(http.MethodGet, "/api/origins", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected OK (200), got status: %d", w.Code)
	}
}

func TestCORSAndOriginMiddleware(t *testing.T) {
	db := setupTestDB(t)

	// Create registered origin
	allowed := originModel.Origin{Domain: "allowed.com", IsBlocked: false}
	blocked := originModel.Origin{Domain: "blocked.com", IsBlocked: true}
	db.Create(&allowed)
	db.Create(&blocked)

	handler := middleware.CORSAndOriginHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))

	// Case 1: Allowed Origin
	req := httptest.NewRequest(http.MethodGet, "/file/test.png", nil)
	req.Header.Set("Origin", "http://allowed.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}

	// Case 2: Blocked Origin
	req = httptest.NewRequest(http.MethodGet, "/file/test.png", nil)
	req.Header.Set("Origin", "http://blocked.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 (Blocked), got: %d", w.Code)
	}

	// Case 3: Unknown Origin
	req = httptest.NewRequest(http.MethodGet, "/file/test.png", nil)
	req.Header.Set("Origin", "http://unknown.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 (Unknown), got: %d", w.Code)
	}

	// Check if UnknownOrigin log exists
	var logRecord originModel.UnknownOrigin
	err := db.Where("domain = ?", "unknown.com").First(&logRecord).Error
	if err != nil {
		t.Errorf("Expected unknown origin to be logged: %v", err)
	}
}

func TestEvaluatePathRules(t *testing.T) {
	db := setupTestDB(t)

	origin := originModel.Origin{Domain: "test.com", IsBlocked: false}
	db.Create(&origin)

	// Create rules
	rule1 := ruleModel.Rule{
		OriginID:        origin.ID,
		Path:            "images",
		IsMaxSize:       true,
		ValueMaxSize:    2,
		ValueUnitSize:   "MB",
		IsExtensions:    true,
		ValueExtensions: "png,jpg,jpeg",
	}
	db.Create(&rule1)

	req := httptest.NewRequest(http.MethodPost, "/file/images/upload", nil)

	// Test Size: Under limit
	allowed, _, _, err := middleware.EvaluatePathRules(req, origin.ID, "images/pic.png", 1*1024*1024, "png")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true under limits: %v", err)
	}

	// Test Size: Exceed limit
	allowed, _, _, err = middleware.EvaluatePathRules(req, origin.ID, "images/pic.png", 3*1024*1024, "png")
	if allowed || err == nil {
		t.Error("Expected disallowed due to size limit exceedance")
	}

	// Test Extension: Not allowed
	allowed, _, _, err = middleware.EvaluatePathRules(req, origin.ID, "images/document.pdf", 500, "pdf")
	if allowed || err == nil {
		t.Error("Expected disallowed due to invalid extension")
	}
}

func TestSecurePathTraversal(t *testing.T) {
	_, err := fileExplorer.SecurePath("../main.go")
	if err == nil {
		t.Error("Expected error for parent directory traversal")
	}

	_, err = fileExplorer.SecurePath("domain_com/../../etc/passwd")
	if err == nil {
		t.Error("Expected error for complex path traversal")
	}

	variables.EnsureBucketDir()
	path, err := fileExplorer.SecurePath("domain_com/user/avatar.png")
	if err != nil {
		t.Errorf("Expected valid path to succeed: %v", err)
	}
	expectedSuffix := filepath.Join("bucket", "domain_com", "user", "avatar.png")
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("Path mismatch: got %s, expected suffix %s", path, expectedSuffix)
	}
}

func TestAPIRoutingAndDashboardFlow(t *testing.T) {
	db := setupTestDB(t)

	// Create test router
	mux := http.NewServeMux()
	modules.RegisterAllRoutes(mux)

	// Create test token
	token, _ := middleware.GenerateJWT("admin")

	// Create test server wrapper
	handler := middleware.CORSAndOriginHandler(adminAuthWrapper(mux))

	// 1. Post to create origin
	originBody, _ := json.Marshal(map[string]interface{}{
		"domain":     "newdomain.com",
		"is_blocked": false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/origins", bytes.NewBuffer(originBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create origin via API: status=%d, body=%s", w.Code, w.Body.String())
	}

	var created originModel.Origin
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.Domain != "newdomain.com" {
		t.Errorf("Domain name mismatch: %s", created.Domain)
	}

	// 2. Query origins list
	req = httptest.NewRequest(http.MethodGet, "/api/origins", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to list origins via API: status=%d", w.Code)
	}

	var list []originModel.Origin
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0].Domain != "newdomain.com" {
		t.Errorf("Unexpected origins list result: %v", list)
	}

	// 3. Post to create rule for the origin
	ruleBody, _ := json.Marshal(map[string]interface{}{
		"origin_id":        created.ID,
		"path":             "secure",
		"is_extensions":    true,
		"value_extensions": "jpg",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/rules", bytes.NewBuffer(ruleBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create rule via API: status=%d, body=%s", w.Code, w.Body.String())
	}

	// Verify rule was saved
	var count int64
	db.Model(&ruleModel.Rule{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 rule in DB, got: %d", count)
	}
}

func TestUploadAndDownloadFileServing(t *testing.T) {
	db := setupTestDB(t)

	// Create test storage directories
	err := os.MkdirAll("bucket/testserve_com", 0755)
	if err != nil {
		t.Fatalf("failed to create test bucket dir: %v", err)
	}
	defer os.RemoveAll("bucket/testserve_com")

	// Register origin
	origin := originModel.Origin{Domain: "testserve.com", IsBlocked: false, ApiKey: "test-key"}
	db.Create(&origin)

	// Create path rule for documents (required for upload)
	rule := ruleModel.Rule{OriginID: origin.ID, Path: "documents"}
	db.Create(&rule)

	mux := http.NewServeMux()
	mux.HandleFunc("/file/", clientFileHandler)
	handler := middleware.CORSAndOriginHandler(mux)

	// 1. Upload File
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "hello.txt")
	part.Write([]byte("Hello LumbungFS"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/file/documents", body)
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("File upload failed: status=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	uploadedFilename := resp["filename"].(string)

	// 2. Download File
	req = httptest.NewRequest(http.MethodGet, "/file/documents/"+uploadedFilename, nil)
	req.Header.Set("Origin", "http://testserve.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("File download failed: status=%d", w.Code)
	}

	if w.Body.String() != "Hello LumbungFS" {
		t.Errorf("Content mismatch: expected 'Hello LumbungFS', got: %s", w.Body.String())
	}
}

func TestUploadAndDownloadFileServingCompressedAndEncrypted(t *testing.T) {
	db := setupTestDB(t)

	// Create test storage directories
	err := os.MkdirAll("bucket/testserve_com", 0755)
	if err != nil {
		t.Fatalf("failed to create test bucket dir: %v", err)
	}
	defer os.RemoveAll("bucket/testserve_com")

	// Register origin
	origin := originModel.Origin{Domain: "testserve.com", IsBlocked: false, ApiKey: "test-key"}
	db.Create(&origin)

	// Create path rule for secure-docs with compress and encrypt enabled
	rule := ruleModel.Rule{
		OriginID:      origin.ID,
		Path:          "secure-docs",
		IsCompress:    true,
		CompressLevel: 5,
		IsEncrypt:     true,
		EncryptionKey: "test-encryption-key",
	}
	db.Create(&rule)

	mux := http.NewServeMux()
	mux.HandleFunc("/file/", clientFileHandler)
	handler := middleware.CORSAndOriginHandler(mux)

	// 1. Upload File
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "secret.txt")
	part.Write([]byte("Secret Data 123"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/file/secure-docs", body)
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("File upload failed: status=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	uploadedFilename := resp["filename"].(string)

	// Check file on disk is NOT plain text "Secret Data 123"
	filePath := filepath.Join("bucket/testserve_com/secure-docs", uploadedFilename)
	diskBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file from disk: %v", err)
	}
	if string(diskBytes) == "Secret Data 123" {
		t.Errorf("Expected file on disk to be compressed/encrypted, but got raw string")
	}

	// 2. Download File
	req = httptest.NewRequest(http.MethodGet, "/file/secure-docs/"+uploadedFilename, nil)
	req.Header.Set("Origin", "http://testserve.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("File download failed: status=%d", w.Code)
	}

	if w.Body.String() != "Secret Data 123" {
		t.Errorf("Content mismatch: expected 'Secret Data 123', got: %s", w.Body.String())
	}
}

func TestGeneratePresignedURLRest(t *testing.T) {
	db := setupTestDB(t)

	// Register origin
	origin := originModel.Origin{Domain: "testserve.com", IsBlocked: false, ApiKey: "test-key"}
	db.Create(&origin)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload/prepare", upload.PrepareUploadHandler)
	handler := middleware.CORSAndOriginHandler(mux)

	// 1. Test JSON request body
	jsonPayload := []byte(`{"path": "documents/nested"}`)
	req := httptest.NewRequest(http.MethodPost, "/upload/prepare", bytes.NewReader(jsonPayload))
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Generate presigned url JSON failed: status=%d, body=%s", w.Code, w.Body.String())
	}

	var respJSON map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &respJSON); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if respJSON["path"] != "documents/nested" {
		t.Errorf("Expected path to be 'documents/nested', got %v", respJSON["path"])
	}

	// 2. Test urlencoded form body
	formPayload := strings.NewReader("path=documents/form-nested")
	req = httptest.NewRequest(http.MethodPost, "/upload/prepare", formPayload)
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Generate presigned url Form failed: status=%d, body=%s", w.Code, w.Body.String())
	}

	var respForm map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &respForm); err != nil {
		t.Fatalf("failed to decode Form response: %v", err)
	}

	if respForm["path"] != "documents/form-nested" {
		t.Errorf("Expected path to be 'documents/form-nested', got %v", respForm["path"])
	}
}

func TestPresignedExpiredTokenMultipartJSONResponse(t *testing.T) {
	db := setupTestDB(t)

	// Register origin
	origin := originModel.Origin{Domain: "testserve.com", IsBlocked: false, ApiKey: "test-key"}
	db.Create(&origin)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", upload.UploadHandler)
	handler := middleware.CORSAndOriginHandler(mux)

	// Hit /upload?token=expired-token with Content-Type multipart/form-data
	req := httptest.NewRequest(http.MethodPost, "/upload?token=expired-token", nil)
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=something")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// It should return JSON error response rather than HTML page
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected response content-type to contain application/json, but got: %s", contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Expected valid JSON body, got error: %v", err)
	}

	if response["error"] == "" {
		t.Errorf("Expected non-empty error message, got empty JSON object")
	}
}

func TestRuleEncryptionTransition(t *testing.T) {
	db := setupTestDB(t)

	// Create test storage directory
	targetPath := "bucket/testserve_com/trans-docs"
	err := os.MkdirAll(targetPath, 0755)
	if err != nil {
		t.Fatalf("failed to create test bucket dir: %v", err)
	}
	defer os.RemoveAll("bucket/testserve_com")

	// Register origin
	origin := originModel.Origin{Domain: "testserve.com", IsBlocked: false, ApiKey: "test-key"}
	db.Create(&origin)

	// Create path rule for trans-docs with encryption DISABLED initially
	rRule := ruleModel.Rule{
		OriginID:   origin.ID,
		Path:       "trans-docs",
		IsEncrypt:  false,
	}
	db.Create(&rRule)

	// Create file1.txt and file2.txt containing raw contents
	file1Path := filepath.Join(targetPath, "file1.txt")
	file2Path := filepath.Join(targetPath, "file2.txt")
	if err := os.WriteFile(file1Path, []byte("ContentA"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2Path, []byte("ContentB"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// 1. Verify files are currently unencrypted on disk
	f1Data, _ := os.ReadFile(file1Path)
	if string(f1Data) != "ContentA" {
		t.Errorf("Expected raw content ContentA on disk, got: %s", string(f1Data))
	}

	// Setup mux & handlers to hit rule endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			rule.CreateRule(w, r)
		} else if r.Method == http.MethodPut {
			rule.UpdateRule(w, r)
		}
	})
	mux.HandleFunc("/file/", clientFileHandler)
	handler := middleware.CORSAndOriginHandler(mux)

	// 2. Perform rule update: transition from IsEncrypt=false to IsEncrypt=true
	payloadJSON := []byte(`{
		"path": "trans-docs",
		"is_encrypt": true,
		"encryption_key": "secret-key-1"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/rules?id="+rRule.ID, bytes.NewReader(payloadJSON))
	req.Header.Set("Origin", "http://testserve.com")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update rule: %d, body: %s", w.Code, w.Body.String())
	}

	// 3. Verify files on disk are now ENCRYPTED (should not match plain text)
	f1Enc, err := os.ReadFile(file1Path)
	if err != nil {
		t.Fatalf("failed to read file1: %v", err)
	}
	if string(f1Enc) == "ContentA" {
		t.Errorf("Expected file1 on disk to be encrypted, but got raw content")
	}

	f2Enc, err := os.ReadFile(file2Path)
	if err != nil {
		t.Fatalf("failed to read file2: %v", err)
	}
	if string(f2Enc) == "ContentB" {
		t.Errorf("Expected file2 on disk to be encrypted, but got raw content")
	}

	// 4. Verify downloading/serving via clientFileHandler yields DECRYPTED files
	reqDownload := httptest.NewRequest(http.MethodGet, "/file/trans-docs/file1.txt", nil)
	reqDownload.Header.Set("Origin", "http://testserve.com")
	wDownload := httptest.NewRecorder()
	handler.ServeHTTP(wDownload, reqDownload)

	if wDownload.Code != http.StatusOK {
		t.Fatalf("Failed to download file1: %d", wDownload.Code)
	}
	if wDownload.Body.String() != "ContentA" {
		t.Errorf("Expected decrypted download content 'ContentA', got: '%s'", wDownload.Body.String())
	}

	// 5. Perform rule update: transition from IsEncrypt=true to IsEncrypt=false
	payloadJSONDec := []byte(`{
		"path": "trans-docs",
		"is_encrypt": false,
		"encryption_key": ""
	}`)
	reqDec := httptest.NewRequest(http.MethodPut, "/api/rules?id="+rRule.ID, bytes.NewReader(payloadJSONDec))
	reqDec.Header.Set("Origin", "http://testserve.com")
	reqDec.Header.Set("Content-Type", "application/json")
	wDec := httptest.NewRecorder()
	handler.ServeHTTP(wDec, reqDec)

	if wDec.Code != http.StatusOK {
		t.Fatalf("Failed to update rule back to unencrypted: %d, body: %s", wDec.Code, wDec.Body.String())
	}

	// 6. Verify files on disk are decrypted back to plain text
	f1Dec, _ := os.ReadFile(file1Path)
	if string(f1Dec) != "ContentA" {
		t.Errorf("Expected file1 on disk to be decrypted back to plain text, got: %s", string(f1Dec))
	}
	f2Dec, _ := os.ReadFile(file2Path)
	if string(f2Dec) != "ContentB" {
		t.Errorf("Expected file2 on disk to be decrypted back to plain text, got: %s", string(f2Dec))
	}
}

