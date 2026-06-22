package auth

import (
	"net/http"
)

// RegisterRoutes registers the login endpoint for auth module
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/login", LoginHandler)
}
