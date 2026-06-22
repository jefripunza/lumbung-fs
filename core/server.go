package core

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"lumbung-fs/core/database"
	"lumbung-fs/core/middlewares"
	"lumbung-fs/core/modules"
	"lumbung-fs/core/modules/auth"
	fileExplorer "lumbung-fs/core/modules/file-explorer"
	fileExplorerModel "lumbung-fs/core/modules/file-explorer/model"
	originModel "lumbung-fs/core/modules/origin/model"
	ruleModel "lumbung-fs/core/modules/rule/model"
)

// ServerStart initializes the database, checks files, registers routes, and starts the server
func ServerStart() {
	log.Println("Initializing LumbungFS Core Service...")

	// 1. Establish DB Connection
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// 2. Run Database AutoMigrations
	err = db.AutoMigrate(
		&originModel.Origin{},
		&originModel.UnknownOrigin{},
		&ruleModel.Rule{},
		&fileExplorerModel.PresignedURL{},
	)
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}
	log.Println("Database migration completed successfully.")

	// Start presigned URL cleanup worker
	go fileExplorer.StartPresignedURLCleanupWorker(db)

	// 3. Initialize credentials file
	auth.InitCredentialsFile()

	// 4. Create standard http ServeMux
	mux := http.NewServeMux()

	// 5. Register Module Routes
	modules.RegisterAllRoutes(mux)

	// 6. Wrap administrative and client routes with appropriate middlewares
	// Note: CORSAndOriginHandler handles origin validation & dashboard CORS.
	// AdminAuth is applied individually inside RegisterAllRoutes or via a sub-router pattern.
	// For standard ServeMux, we can wrap the entire multiplexer with CORSAndOriginHandler
	// and then check JWT inside /api/... handlers except login.
	mainHandler := middlewares.CORSAndOriginHandler(adminAuthWrapper(mux))

	port := 8080
	log.Printf("LumbungFS server successfully started on port :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mainHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// adminAuthWrapper intercepts /api/ requests (excluding /api/auth/login) and applies JWT checks
func adminAuthWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") && path != "/api/auth/login" {
			// Wrap with AdminAuth middlewares logic
			middlewares.AdminAuth(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

