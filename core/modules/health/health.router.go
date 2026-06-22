package health

import (
	"net/http"
)

// RegisterRoutes registers the health check endpoint
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", HealthHandler)
}
