package upload

import (
	"net/http"
)

// RegisterRoutes registers endpoints for the upload module
func RegisterRoutes(mux *http.ServeMux) {
	// GET/POST unified upload handler (with token/API Key)
	mux.HandleFunc("/upload", UploadHandler)

	// REST API presigned URL preparation endpoint
	mux.HandleFunc("/upload/prepare", PrepareUploadHandler)
}
