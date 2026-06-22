package origin

import (
	"net/http"
)

// RegisterRoutes registers routes for the origin module in the http.ServeMux
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/origins", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListOrigins(w, r)
		case http.MethodPost:
			CreateOrigin(w, r)
		case http.MethodPut:
			UpdateOrigin(w, r)
		case http.MethodDelete:
			DeleteOrigin(w, r)
		default:
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/unknown-origins", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListUnknownOrigins(w, r)
		case http.MethodDelete:
			DeleteUnknownOrigin(w, r)
		default:
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/unknown-origins/promote", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			PromoteUnknownOrigin(w, r)
		} else {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/origins/apikey", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			GenerateApiKey(w, r)
		} else {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
}
