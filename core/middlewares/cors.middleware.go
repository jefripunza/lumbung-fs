package middlewares

import (
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"lumbung-fs/core/database"
	originModel "lumbung-fs/core/modules/origin/model"
	"lumbung-fs/core/templates"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ParseDomain strips protocol and path, and port if it is 80 or 443
func ParseDomain(raw string) string {
	if raw == "" {
		return ""
	}

	// Strip protocols
	if strings.HasPrefix(raw, "https://") {
		raw = raw[8:]
	} else if strings.HasPrefix(raw, "http://") {
		raw = raw[7:]
	}

	// Strip path (if any)
	if idx := strings.Index(raw, "/"); idx != -1 {
		raw = raw[:idx]
	}

	// Check for port
	host, port, err := net.SplitHostPort(raw)
	if err == nil {
		if port != "80" && port != "443" && port != "" {
			raw = host + ":" + port
		} else {
			raw = host
		}
	}

	return strings.ToLower(raw)
}

// ResolveDomain extracts the registered domain from Origin, X-Forwarded-Host, X-Original-Host, or Host headers
func ResolveDomain(r *http.Request) string {
	originHeader := r.Header.Get("Origin")
	if originHeader != "" {
		return ParseDomain(originHeader)
	}

	forwardedHost := r.Header.Get("X-Forwarded-Host")
	if forwardedHost != "" {
		if idx := strings.Index(forwardedHost, ","); idx != -1 {
			forwardedHost = forwardedHost[:idx]
		}
		return ParseDomain(strings.TrimSpace(forwardedHost))
	}

	originalHost := r.Header.Get("X-Original-Host")
	if originalHost != "" {
		if idx := strings.Index(originalHost, ","); idx != -1 {
			originalHost = originalHost[:idx]
		}
		return ParseDomain(strings.TrimSpace(originalHost))
	}

	referer := r.Header.Get("Referer")
	if referer != "" {
		return ParseDomain(referer)
	}

	hostHeader := r.Host
	if hostHeader == "" {
		hostHeader = r.Header.Get("Host")
	}
	return ParseDomain(hostHeader)
}

// GetClientIP retrieves the real client IP address
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// CORSAndOriginHandler handles CORS headers and validates request origins for file requests
func CORSAndOriginHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Determine origin domain
		requestDomain := ResolveDomain(r)

		// Check if request has an API Key first
		var apiKey string
		apiKey = r.Header.Get("X-API-Key")
		if apiKey == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					apiKey = parts[1]
				}
			}
		}

		if requestDomain == "" && apiKey == "" {
			http.Error(w, "Forbidden: Missing origin or host header", http.StatusForbidden)
			return
		}

		// Check if it's an API route or presigned upload.
		isAPIRoute := strings.HasPrefix(r.URL.Path, "/api/")
		isPresignedUpload := r.URL.Path == "/upload" && r.URL.Query().Get("token") != ""

		if isAPIRoute {
			dashboardOrigin := os.Getenv("WEB_DASHBOARD_ORIGIN")
			if dashboardOrigin == "" {
				http.Error(w, "Forbidden: WEB_DASHBOARD_ORIGIN env is required", http.StatusForbidden)
				return
			}
			parsedDashboardOrigin := ParseDomain(dashboardOrigin)
			if requestDomain != parsedDashboardOrigin {
				http.Error(w, "Forbidden: Invalid dashboard origin", http.StatusForbidden)
				return
			}

			originHeader := r.Header.Get("Origin")
			if originHeader != "" && ParseDomain(originHeader) == parsedDashboardOrigin {
				w.Header().Set("Access-Control-Allow-Origin", originHeader)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", dashboardOrigin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if isPresignedUpload {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// 2. Validate domain against DB or WEB_DASHBOARD_ORIGIN env
		var dbOrigin originModel.Origin
		var isValid bool

		if apiKey != "" {
			if err := database.DB.
				Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
				Where("api_key = ?", apiKey).
				First(&dbOrigin).
				Error; err == nil {
				isValid = true
			}
		}

		dashboardOrigin := ParseDomain(os.Getenv("WEB_DASHBOARD_ORIGIN"))
		if !isValid && requestDomain != "" {
			if err := database.DB.
				Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}). // silent mode to avoid noise
				Where("domain = ?", requestDomain).
				First(&dbOrigin).
				Error; err == nil {
				isValid = true
			} else {
				if dashboardOrigin != "" {
					if requestDomain == dashboardOrigin {
						isValid = true
					}
				}
			}
		}
		log.Printf("dashboardOrigin %s, requestDomain %s, isValid %v\n", dashboardOrigin, requestDomain, isValid)

		if !isValid {
			// Domain not registered: Log or update UnknownOrigin
			ip := GetClientIP(r)
			log.Printf("Unknown origin request: domain=%s, ip=%s", requestDomain, ip)

			var existing originModel.UnknownOrigin
			if err := database.DB.Where("domain = ?", requestDomain).First(&existing).Error; err == nil {
				// Update existing entry
				existing.AccessAt = time.Now()
				existing.IPAddress = ip
				database.DB.Save(&existing)
			} else {
				// Create new entry
				unknown := originModel.UnknownOrigin{
					Domain:    requestDomain,
					AccessAt:  time.Now(),
					IPAddress: ip,
				}
				database.DB.Create(&unknown)
			}

			serveForbiddenOriginHTML(w, requestDomain)
			return
		}

		if dbOrigin.IsBlocked {
			http.Error(w, "Forbidden: Origin domain is blocked", http.StatusForbidden)
			return
		}

		// 3. Set custom CORS for allowed origin
		originHeader := r.Header.Get("Origin")
		if originHeader != "" {
			w.Header().Set("Access-Control-Allow-Origin", originHeader)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// serveForbiddenOriginHTML serves a beautifully designed HTML page for unregistered domain access.
func serveForbiddenOriginHTML(w http.ResponseWriter, domain string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)

	escapedDomain := html.EscapeString(domain)
	res := strings.ReplaceAll(templates.ForbiddenOriginTemplate, "{{.Domain}}", escapedDomain)
	w.Write([]byte(res))
}
