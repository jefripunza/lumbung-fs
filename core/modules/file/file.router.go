package file

import (
	"net/http"
)

// RegisterRoutes registers client file serving route
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/file/", ClientFileHandler)
}
