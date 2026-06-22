package health

import (
	"net/http"
)

// HealthHandler returns the health status of lumbung-fs service
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"lumbung-fs"}`))
}
