package rule

import (
	"net/http"
)

// RegisterRoutes registers routes for the rule module in the http.ServeMux
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListRules(w, r)
		case http.MethodPost:
			CreateRule(w, r)
		case http.MethodPut:
			UpdateRule(w, r)
		case http.MethodDelete:
			DeleteRule(w, r)
		default:
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
}
