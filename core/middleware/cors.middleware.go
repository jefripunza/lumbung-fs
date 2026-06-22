package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"lumbung-fs/core/database"
	originModel "lumbung-fs/core/modules/origin/model"
)

// ParseDomain strips protocol and port from raw origin/host strings
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
	
	// Strip path or port
	if idx := strings.IndexAny(raw, ":/"); idx != -1 {
		raw = raw[:idx]
	}
	
	return strings.ToLower(raw)
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
		originHeader := r.Header.Get("Origin")
		hostHeader := r.Host
		if hostHeader == "" {
			hostHeader = r.Header.Get("Host")
		}
		
		requestDomain := ""
		if originHeader != "" {
			requestDomain = ParseDomain(originHeader)
		} else {
			requestDomain = ParseDomain(hostHeader)
		}

		if requestDomain == "" {
			http.Error(w, "Forbidden: Missing origin or host header", http.StatusForbidden)
			return
		}

		// Allow localhost or direct server address for admin dashboard access
		// but enforce rules on custom domains. We also check if it's an API route.
		isAPIRoute := strings.HasPrefix(r.URL.Path, "/api/")
		if isAPIRoute {
			// Apply standard open CORS for administrative dashboard APIs
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

		// 2. Validate domain against DB
		var dbOrigin originModel.Origin
		result := database.DB.Where("domain = ?", requestDomain).First(&dbOrigin)
		
		if result.Error != nil {
			// Domain not registered: Log to UnknownOrigin
			ip := GetClientIP(r)
			log.Printf("Unknown origin request: domain=%s, ip=%s", requestDomain, ip)
			
			unknown := originModel.UnknownOrigin{
				Domain:    requestDomain,
				AccessAt:  time.Now(),
				IPAddress: ip,
			}
			database.DB.Create(&unknown)
			
			http.Error(w, "Forbidden: Unknown origin domain", http.StatusForbidden)
			return
		}

		if dbOrigin.IsBlocked {
			http.Error(w, "Forbidden: Origin domain is blocked", http.StatusForbidden)
			return
		}

		// 3. Set custom CORS for allowed origin
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
