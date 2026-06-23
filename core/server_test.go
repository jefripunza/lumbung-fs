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
	"lumbung-fs/core/middlewares"
	"lumbung-fs/core/modules"
	"lumbung-fs/core/modules/auth"
	"lumbung-fs/core/modules/file"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	fileExplorerModel "lumbung-fs/core/modules/file-explorer/model"
	originModel "lumbung-fs/core/modules/origin/model"
	"lumbung-fs/core/modules/rule"
	ruleModel "lumbung-fs/core/modules/rule/model"
	"lumbung-fs/core/modules/upload"
	"lumbung-fs/core/variables"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory database and auto-migrates models
func setupTestDB(t *testing.T) *gorm.DB {
	os.Setenv("WEB_DASHBOARD_ORIGIN", "http://localhost:5173")
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
	if !auth.VerifyMD5("123456", auth.DefaultMD5Hash) {
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
		actual := middlewares.ParseDomain(tc.input)
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
	if actual := middlewares.ResolveDomain(req); actual != "origin-domain.com:8080" {
		t.Errorf("ResolveDomain failed for Origin header: expected 'origin-domain.com:8080', got '%s'", actual)
	}

	// Test X-Forwarded-Host header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("X-Forwarded-Host", "forwarded-domain.com:7070, proxy-domain.com")
	req.Host = "host-domain.com:9090"
	if actual := middlewares.ResolveDomain(req); actual != "forwarded-domain.com:7070" {
		t.Errorf("ResolveDomain failed for X-Forwarded-Host header: expected 'forwarded-domain.com:7070', got '%s'", actual)
	}

	// Test X-Original-Host header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("X-Original-Host", "original-domain.com:6060")
	req.Host = "host-domain.com:9090"
	if actual := middlewares.ResolveDomain(req); actual != "original-domain.com:6060" {
		t.Errorf("ResolveDomain failed for X-Original-Host header: expected 'original-domain.com:6060', got '%s'", actual)
	}

	// Test Referer header
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("Referer", "http://referer-domain.com:5050/some/path")
	req.Host = "host-domain.com:9090"
	if actual := middlewares.ResolveDomain(req); actual != "referer-domain.com:5050" {
		t.Errorf("ResolveDomain failed for Referer header: expected 'referer-domain.com:5050', got '%s'", actual)
	}

	// Test Host header fallback
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Host = "host-domain.com:9090"
	if actual := middlewares.ResolveDomain(req); actual != "host-domain.com:9090" {
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
	token, err := middlewares.GenerateJWT("admin")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	// Create test server wrapper
	handler := middlewares.AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	handler := middlewares.CORSAndOriginHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	allowed, _, _, err := middlewares.EvaluatePathRules(req, origin.ID, "images/pic.png", 1*1024*1024, "png")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true under limits: %v", err)
	}

	// Test Size: Exceed limit
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "images/pic.png", 3*1024*1024, "png")
	if allowed || err == nil {
		t.Error("Expected disallowed due to size limit exceedance")
	}

	// Test Extension: Not allowed
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "images/document.pdf", 500, "pdf")
	if allowed || err == nil {
		t.Error("Expected disallowed due to invalid extension")
	}
}

func TestStrictExternalValidation(t *testing.T) {
	db := setupTestDB(t)

	origin := originModel.Origin{Domain: "test-validation.com", IsBlocked: false}
	db.Create(&origin)

	// Set up a mock external validation server
	var responseStatus int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseStatus)
		_, _ = w.Write([]byte("mock response body"))
	}))
	defer mockServer.Close()

	// Create a rule with validate_url pointing to the mock server
	rule := ruleModel.Rule{
		OriginID:       origin.ID,
		Path:           "secure",
		ValidateMethod: "cache",
		ValidateURL:    mockServer.URL,
	}
	db.Create(&rule)

	req := httptest.NewRequest(http.MethodGet, "/file/secure/document.txt", nil)

	// Test 1: External validation returns 200 OK -> Should allow access
	responseStatus = http.StatusOK
	allowed, _, _, err := middlewares.EvaluatePathRules(req, origin.ID, "secure/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true on 200 OK: %v", err)
	}

	// Test 2: External validation returns 201 Created -> Should allow access (100-399 range)
	responseStatus = http.StatusCreated
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "secure/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true on 201 Created: %v", err)
	}

	// Test 3: External validation returns 401 Unauthorized -> Should block access (400-599 range)
	responseStatus = http.StatusUnauthorized
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "secure/document.txt", 100, "txt")
	if allowed || err == nil {
		t.Error("Expected disallowed on 401 Unauthorized")
	}

	// Test 4: Upload request (POST) with validation rule -> Should bypass ValidateMethod checks (even when validation server returns 401)
	responseStatus = http.StatusUnauthorized
	uploadReq := httptest.NewRequest(http.MethodPost, "/file/secure/document.txt", nil)
	allowed, _, _, err = middlewares.EvaluatePathRules(uploadReq, origin.ID, "secure/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true on upload POST request, bypassing ValidateMethod validation: %v", err)
	}
}

func TestJWTValidationHeaderOrQuery(t *testing.T) {
	db := setupTestDB(t)

	origin := originModel.Origin{Domain: "jwt-test.com", IsBlocked: false}
	db.Create(&origin)

	var lastAuthHeader string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer mockServer.Close()

	// Create a rule with validate_method: jwt and validate_url pointing to mock server
	rule := ruleModel.Rule{
		OriginID:       origin.ID,
		Path:           "secure-jwt",
		ValidateMethod: "jwt",
		ValidateURL:    mockServer.URL,
	}
	db.Create(&rule)

	// Test 1: Neither header nor query param -> Should block access
	req1 := httptest.NewRequest(http.MethodGet, "/file/secure-jwt/document.txt", nil)
	allowed, _, _, err := middlewares.EvaluatePathRules(req1, origin.ID, "secure-jwt/document.txt", 100, "txt")
	if allowed || err == nil {
		t.Error("Expected disallowed when both Authorization header and token query parameter are missing")
	}

	// Test 2: Token passed via Authorization Header -> Should allow access and forward the header
	lastAuthHeader = ""
	req2 := httptest.NewRequest(http.MethodGet, "/file/secure-jwt/document.txt", nil)
	req2.Header.Set("Authorization", "Bearer my-jwt-token-1")
	allowed, _, _, err = middlewares.EvaluatePathRules(req2, origin.ID, "secure-jwt/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true with Authorization header: %v", err)
	}
	if lastAuthHeader != "Bearer my-jwt-token-1" {
		t.Errorf("Expected forwarded Authorization header to be 'Bearer my-jwt-token-1', got: '%s'", lastAuthHeader)
	}

	// Test 3: Token passed via query parameter -> Should allow access and forward as Authorization: Bearer <token>
	lastAuthHeader = ""
	req3 := httptest.NewRequest(http.MethodGet, "/file/secure-jwt/document.txt?token=my-jwt-token-2", nil)
	allowed, _, _, err = middlewares.EvaluatePathRules(req3, origin.ID, "secure-jwt/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true with token query parameter: %v", err)
	}
	if lastAuthHeader != "Bearer my-jwt-token-2" {
		t.Errorf("Expected forwarded Authorization header to be 'Bearer my-jwt-token-2', got: '%s'", lastAuthHeader)
	}

	// Test 4: Token passed via query parameter starting with bearer -> Should preserve the bearer format
	lastAuthHeader = ""
	req4 := httptest.NewRequest(http.MethodGet, "/file/secure-jwt/document.txt?token=bearer%20my-jwt-token-3", nil)
	allowed, _, _, err = middlewares.EvaluatePathRules(req4, origin.ID, "secure-jwt/document.txt", 100, "txt")
	if !allowed || err != nil {
		t.Errorf("Expected allowed=true with bearer token query parameter: %v", err)
	}
	if lastAuthHeader != "bearer my-jwt-token-3" {
		t.Errorf("Expected forwarded Authorization header to be 'bearer my-jwt-token-3', got: '%s'", lastAuthHeader)
	}
}

func TestExternalValidationJsonError(t *testing.T) {
	db := setupTestDB(t)

	origin := originModel.Origin{Domain: "json-err.com", IsBlocked: false}
	db.Create(&origin)

	var mockResponse string
	var mockStatus int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(mockStatus)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	rule := ruleModel.Rule{
		OriginID:       origin.ID,
		Path:           "api-check",
		ValidateMethod: "cache",
		ValidateURL:    mockServer.URL,
	}
	db.Create(&rule)

	req := httptest.NewRequest(http.MethodGet, "/file/api-check/photo.jpg", nil)

	// Case 1: JSON response contains "message"
	mockStatus = http.StatusForbidden
	mockResponse = `{"message": "Custom forbidden message"}`
	allowed, _, _, err := middlewares.EvaluatePathRules(req, origin.ID, "api-check/photo.jpg", 100, "jpg")
	if allowed || err == nil {
		t.Error("Expected access to be denied")
	}
	if err != nil && !strings.Contains(err.Error(), "Custom forbidden message") {
		t.Errorf("Expected error to contain 'Custom forbidden message', got: %v", err)
	}

	// Case 2: JSON response contains "error" as string
	mockStatus = http.StatusInternalServerError
	mockResponse = `{"error": "Database error details"}`
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "api-check/photo.jpg", 100, "jpg")
	if allowed || err == nil {
		t.Error("Expected access to be denied")
	}
	if err != nil && !strings.Contains(err.Error(), "Database error details") {
		t.Errorf("Expected error to contain 'Database error details', got: %v", err)
	}

	// Case 3: JSON response contains "error" as object with "message" (Bun SQL format)
	mockStatus = http.StatusBadRequest
	mockResponse = `{"error": {"message": "Invalid query parameters"}}`
	allowed, _, _, err = middlewares.EvaluatePathRules(req, origin.ID, "api-check/photo.jpg", 100, "jpg")
	if allowed || err == nil {
		t.Error("Expected access to be denied")
	}
	if err != nil && !strings.Contains(err.Error(), "Invalid query parameters") {
		t.Errorf("Expected error to contain 'Invalid query parameters', got: %v", err)
	}
}

func TestFallbackProxy(t *testing.T) {
	db := setupTestDB(t)

	origin := originModel.Origin{Domain: "proxy-test.com", IsBlocked: false}
	db.Create(&origin)

	// Fallback mock server returns custom HTML error content
	mockFallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound) // Custom code
		_, _ = w.Write([]byte("<html><body>Custom Proxy Error Page</body></html>"))
	}))
	defer mockFallbackServer.Close()

	// Validation mock server rejects
	mockValidationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer mockValidationServer.Close()

	// Create rule with validate_url and validate_fallback_url
	rule := ruleModel.Rule{
		OriginID:            origin.ID,
		Path:                "secure-proxy",
		ValidateMethod:      "cache",
		ValidateURL:         mockValidationServer.URL,
		ValidateFallbackURL: mockFallbackServer.URL,
	}
	db.Create(&rule)

	// Setup database directory and a dummy file so Stat succeeds
	variables.EnsureBucketDir()
	originSnake := variables.DomainToSnake(origin.Domain)
	targetDir := filepath.Join(variables.BucketDir, originSnake, "secure-proxy")
	_ = os.MkdirAll(targetDir, 0755)
	filePath := filepath.Join(targetDir, "test.txt")
	_ = os.WriteFile(filePath, []byte("original content"), 0644)
	defer os.RemoveAll(filepath.Join(variables.BucketDir, originSnake))

	mux := http.NewServeMux()
	mux.HandleFunc("/file/", file.ClientFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/file/secure-proxy/test.txt", nil)
	req.Host = "proxy-test.com"
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Verify that we got the fallback server response status code and body instead of a 302 redirect
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d from proxy, got %d", http.StatusNotFound, w.Code)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "Custom Proxy Error Page") {
		t.Errorf("Expected body to contain fallback page content, got: %s", bodyStr)
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
	token, _ := middlewares.GenerateJWT("admin")

	// Create test server wrapper
	handler := middlewares.CORSAndOriginHandler(adminAuthWrapper(mux))

	// 1. Post to create origin
	originBody, _ := json.Marshal(map[string]interface{}{
		"domain":     "newdomain.com",
		"is_blocked": false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/origins", bytes.NewBuffer(originBody))
	req.Header.Set("Origin", "http://localhost:5173")
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
	req.Header.Set("Origin", "http://localhost:5173")
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
	req.Header.Set("Origin", "http://localhost:5173")
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
	mux.HandleFunc("/file/", file.ClientFileHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

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
	mux.HandleFunc("/file/", file.ClientFileHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

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

	// Create a rule for "documents" so uploads are allowed
	rule := ruleModel.Rule{
		OriginID: origin.ID,
		Path:     "documents",
	}
	db.Create(&rule)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload/prepare", upload.PrepareUploadHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

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
	handler := middlewares.CORSAndOriginHandler(mux)

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
		OriginID:  origin.ID,
		Path:      "trans-docs",
		IsEncrypt: false,
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
	mux.HandleFunc("/file/", file.ClientFileHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

	// 2. Perform rule update: transition from IsEncrypt=false to IsEncrypt=true
	payloadJSON := []byte(`{
		"path": "trans-docs",
		"is_encrypt": true,
		"encryption_key": "secret-key-1"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/rules?id="+rRule.ID, bytes.NewReader(payloadJSON))
	req.Header.Set("Origin", "http://localhost:5173")
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

	// 4. Verify downloading/serving via file.ClientFileHandler yields DECRYPTED files
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
	reqDec.Header.Set("Origin", "http://localhost:5173")
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

func TestWebDashboardOriginValidation(t *testing.T) {
	setupTestDB(t)

	// Save original env
	origEnv := os.Getenv("WEB_DASHBOARD_ORIGIN")
	defer os.Setenv("WEB_DASHBOARD_ORIGIN", origEnv)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})
	handler := middlewares.CORSAndOriginHandler(mux)

	// Case 1: WEB_DASHBOARD_ORIGIN is empty
	os.Setenv("WEB_DASHBOARD_ORIGIN", "")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 when WEB_DASHBOARD_ORIGIN is empty, got %d", w.Code)
	}

	// Case 2: WEB_DASHBOARD_ORIGIN is set but request origin does not match
	os.Setenv("WEB_DASHBOARD_ORIGIN", "http://localhost:5173")
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 when origin does not match WEB_DASHBOARD_ORIGIN, got %d", w.Code)
	}

	// Case 3: WEB_DASHBOARD_ORIGIN matches request origin
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when origin matches WEB_DASHBOARD_ORIGIN, got %d", w.Code)
	}
}

func TestEnvCredentialsLogin(t *testing.T) {
	setupTestDB(t)

	// Save original envs
	origUser := os.Getenv("USERNAME")
	origPass := os.Getenv("PASSWORD")
	defer func() {
		os.Setenv("USERNAME", origUser)
		os.Setenv("PASSWORD", origPass)
	}()

	// Case 1: USERNAME and PASSWORD set in env
	os.Setenv("USERNAME", "customuser")
	os.Setenv("PASSWORD", "custompass")

	// Correct login
	payload, _ := json.Marshal(map[string]string{
		"username": "customuser",
		"password": "custompass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	auth.LoginHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for correct env credentials, got %d, body: %s", w.Code, w.Body.String())
	}

	// Incorrect login password
	payload, _ = json.Marshal(map[string]string{
		"username": "customuser",
		"password": "wrongpass",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	w = httptest.NewRecorder()
	auth.LoginHandler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for incorrect env password, got %d", w.Code)
	}

	// Incorrect login username
	payload, _ = json.Marshal(map[string]string{
		"username": "wronguser",
		"password": "custompass",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	w = httptest.NewRecorder()
	auth.LoginHandler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for incorrect env username, got %d", w.Code)
	}

	// Case 2: Only USERNAME set in env
	os.Setenv("USERNAME", "onlyuser")
	os.Setenv("PASSWORD", "")

	auth.InitCredentialsFile()

	payload, _ = json.Marshal(map[string]string{
		"username": "onlyuser",
		"password": "123456",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	w = httptest.NewRecorder()
	auth.LoginHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for custom username + default password, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDualOriginValidation(t *testing.T) {
	setupTestDB(t)

	// Save original env
	origEnv := os.Getenv("WEB_DASHBOARD_ORIGIN")
	defer os.Setenv("WEB_DASHBOARD_ORIGIN", origEnv)

	// Setup a mock server/handler
	mux := http.NewServeMux()
	mux.HandleFunc("/file/", file.ClientFileHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

	// Prepare bucket folder structure
	// We want to test reading a file for the dashboard origin
	dashboardDomain := "localhost:5173"
	dashboardSnake := variables.DomainToSnake(dashboardDomain)
	targetDir := filepath.Join(variables.BucketDir, dashboardSnake)
	os.MkdirAll(targetDir, 0755)
	defer os.RemoveAll(targetDir)

	testFilePath := filepath.Join(targetDir, "test.txt")
	os.WriteFile(testFilePath, []byte("hello dashboard file"), 0644)

	// Case 1: WEB_DASHBOARD_ORIGIN matches dashboardDomain, but the domain is NOT in the database origins.
	os.Setenv("WEB_DASHBOARD_ORIGIN", "http://"+dashboardDomain)
	req := httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("Origin", "http://"+dashboardDomain)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for dashboard origin match via env, got %d, body: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "hello dashboard file" {
		t.Errorf("Expected content 'hello dashboard file', got %q", w.Body.String())
	}

	// Case 2: Origin is NOT in DB and does NOT match WEB_DASHBOARD_ORIGIN
	os.Setenv("WEB_DASHBOARD_ORIGIN", "http://someotherdashboard.com")
	req = httptest.NewRequest(http.MethodGet, "/file/test.txt", nil)
	req.Header.Set("Origin", "http://"+dashboardDomain)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// It should serve forbidden HTML page or status code 403
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for unregistered origin, got %d", w.Code)
	}

	// Case 3: Origin is registered in the DB
	// Add origin to DB
	dbOrigin := originModel.Origin{
		Domain: "registereddomain.com",
	}
	database.DB.Create(&dbOrigin)
	defer database.DB.Delete(&dbOrigin)

	registeredSnake := variables.DomainToSnake(dbOrigin.Domain)
	targetRegisteredDir := filepath.Join(variables.BucketDir, registeredSnake)
	os.MkdirAll(targetRegisteredDir, 0755)
	defer os.RemoveAll(targetRegisteredDir)

	registeredFilePath := filepath.Join(targetRegisteredDir, "reg.txt")
	os.WriteFile(registeredFilePath, []byte("hello registered file"), 0644)

	req = httptest.NewRequest(http.MethodGet, "/file/reg.txt", nil)
	req.Header.Set("Origin", "http://registereddomain.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for registered DB origin, got %d, body: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "hello registered file" {
		t.Errorf("Expected content 'hello registered file', got %q", w.Body.String())
	}
}

func TestApiKeyOriginFallback(t *testing.T) {
	setupTestDB(t)

	// Create registered origin in DB with API key
	dbOrigin := originModel.Origin{
		Domain: "apidomain.com",
		ApiKey: "test-api-key-123",
	}
	database.DB.Create(&dbOrigin)
	defer database.DB.Delete(&dbOrigin)

	// Create a rule for "images" so uploads are allowed
	rule := ruleModel.Rule{
		OriginID: dbOrigin.ID,
		Path:     "images",
	}
	database.DB.Create(&rule)
	defer database.DB.Delete(&rule)

	// Setup mock handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/upload/prepare", upload.PrepareUploadHandler)
	handler := middlewares.CORSAndOriginHandler(mux)

	// Case 1: Send request to /upload/prepare with valid X-API-Key, without matching Host/Origin header
	payload, _ := json.Marshal(map[string]string{
		"path": "images",
	})
	req := httptest.NewRequest(http.MethodPost, "/upload/prepare", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key-123")
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid API key without host matching, got %d, body: %s", w.Code, w.Body.String())
	}

	// Case 2: Send request to /upload/prepare with invalid X-API-Key, without matching Host/Origin header
	req = httptest.NewRequest(http.MethodPost, "/upload/prepare", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "wrong-api-key")
	
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// It should be blocked (unauthorized or forbidden)
	if w.Code == http.StatusOK {
		t.Errorf("Expected status to be blocked (non-200) for invalid API key, got %d", w.Code)
	}
}

