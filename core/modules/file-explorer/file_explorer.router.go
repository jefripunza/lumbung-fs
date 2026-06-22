package file_explorer

import (
	"net/http"
)

// RegisterRoutes registers routes for the file-explorer module in the http.ServeMux
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/explorer/list", ListItems)
	mux.HandleFunc("/api/explorer/folder", CreateFolder)
	mux.HandleFunc("/api/explorer/upload", UploadFile)
	mux.HandleFunc("/api/explorer/download", DownloadFile)
	mux.HandleFunc("/api/explorer/delete", DeleteItem)
}
