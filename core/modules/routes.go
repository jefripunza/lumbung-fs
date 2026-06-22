package modules

import (
	"net/http"

	"lumbung-fs/core/modules/file-explorer"
	"lumbung-fs/core/modules/origin"
	"lumbung-fs/core/modules/rule"
)

// RegisterAllRoutes aggregates routes from all modules and registers them on the ServeMux
func RegisterAllRoutes(mux *http.ServeMux) {
	origin.RegisterRoutes(mux)
	rule.RegisterRoutes(mux)
	file_explorer.RegisterRoutes(mux)
}
