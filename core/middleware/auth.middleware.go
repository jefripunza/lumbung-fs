package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"lumbung-fs/core/database"
	ruleModel "lumbung-fs/core/modules/rule/model"
	
	"github.com/golang-jwt/jwt/v5"
)

var JWTSecretKey []byte

type contextKey string

const (
	AdminUserContextKey contextKey = "admin_user"
	defaultSecret                  = "lumbungfs-super-secret-key-change-me-12345"
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = defaultSecret
	}
	JWTSecretKey = []byte(secret)
}

// GenerateJWT generates a signed HS256 JWT token for the admin
func GenerateJWT(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecretKey)
}

// AdminAuth middleware validates JWT token for admin routes
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return JWTSecretKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
			return
		}

		username, _ := claims["sub"].(string)
		
		ctx := context.WithValue(r.Context(), AdminUserContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ParseSizeInBytes converts value + unit (KB, MB, GB) to bytes
func ParseSizeInBytes(value int, unit string) int64 {
	multiplier := int64(1)
	switch strings.ToUpper(unit) {
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	}
	return int64(value) * multiplier
}

// EvaluatePathRules checks all path-based rules for a given origin
// Returns (isAllowed, redirectURL, statusCode, err)
func EvaluatePathRules(r *http.Request, originID string, path string, fileSize int64, fileExtension string) (bool, string, int, error) {
	// Clean path and separate segments
	cleanPath := strings.Trim(path, "/")
	segments := strings.Split(cleanPath, "/")

	// Fetch rules for this origin
	var rules []ruleModel.Rule
	if err := database.DB.Where("origin_id = ?", originID).Find(&rules).Error; err != nil {
		return false, "", http.StatusInternalServerError, err
	}

	if len(rules) == 0 {
		return true, "", http.StatusOK, nil
	}

	// Determine if any rule path matches segments of the cleanPath
	// E.g., rule.Path = "ktp" matches "user/ktp/photo.png"
	var matchedRule *ruleModel.Rule
	for _, rule := range rules {
		rulePathClean := strings.Trim(rule.Path, "/")
		
		// Exact match or partial match on segments
		matchFound := false
		if rulePathClean == cleanPath {
			matchFound = true
		} else {
			// Check if rule.Path is a segment in the request path
			ruleSegments := strings.Split(rulePathClean, "/")
			if len(ruleSegments) <= len(segments) {
				// Simple matching: check if ruleSegments is a prefix or sub-slice
				for i := 0; i <= len(segments)-len(ruleSegments); i++ {
					match := true
					for j := 0; j < len(ruleSegments); j++ {
						if segments[i+j] != ruleSegments[j] {
							match = false
							break
						}
					}
					if match {
						matchFound = true
						break
					}
				}
			}
		}

		if matchFound {
			matchedRule = &rule
			break
		}
	}

	// If no rules matched the path, allow request
	if matchedRule == nil {
		return true, "", http.StatusOK, nil
	}

	// 1. File Size Verification
	if matchedRule.IsMaxSize && fileSize > 0 {
		maxBytes := ParseSizeInBytes(matchedRule.ValueMaxSize, matchedRule.ValueUnitSize)
		if fileSize > maxBytes {
			return false, "", http.StatusBadRequest, fmt.Errorf("file size %d exceeds allowed max size %d %s (%d bytes)", fileSize, matchedRule.ValueMaxSize, matchedRule.ValueUnitSize, maxBytes)
		}
	}

	// 2. File Extension Verification
	if matchedRule.IsExtensions && fileExtension != "" {
		allowedExts := strings.Split(strings.ToLower(matchedRule.ValueExtensions), ",")
		extClean := strings.TrimPrefix(strings.ToLower(fileExtension), ".")
		
		extMatched := false
		for _, allowed := range allowedExts {
			if strings.TrimSpace(allowed) == extClean {
				extMatched = true
				break
			}
		}
		
		if !extMatched {
			return false, "", http.StatusBadRequest, fmt.Errorf("file extension .%s is not allowed", extClean)
		}
	}

	// 3. Authentication Verification (External Endpoint check)
	if matchedRule.ValidateMethod != "" && matchedRule.ValidateURL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, matchedRule.ValidateURL, nil)
		if err != nil {
			return false, matchedRule.ValidateFallbackURL, http.StatusUnauthorized, err
		}

		// Forward specific credentials headers
		if auth := r.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if cookie := r.Header.Get("Cookie"); cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		
		// Copy any headers that start with custom prefixes or validation methods
		for key, vals := range r.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-") {
				for _, val := range vals {
					req.Header.Add(key, val)
				}
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return false, matchedRule.ValidateFallbackURL, http.StatusUnauthorized, err
		}
		defer resp.Body.Close()

		// Read small body for debug logging if necessary
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return false, matchedRule.ValidateFallbackURL, http.StatusUnauthorized, fmt.Errorf("external validation returned status %d: %s", resp.StatusCode, string(bodyBytes))
		}
	}

	return true, "", http.StatusOK, nil
}
