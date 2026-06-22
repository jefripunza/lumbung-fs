package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	uploadDir = "uploads"
)

func main() {
	// Create upload directory if it doesn't exist
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Handler for file upload
	http.HandleFunc("/upload", uploadHandler)

	// Handler for serving uploaded files
	http.HandleFunc("/files/", serveFileHandler)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	port := 8080
	log.Printf("Starting Lumbung File System server on :%d", port)

	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// uploadHandler handles file upload requests
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form, max memory 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a unique filename (timestamp + original filename)
	timestamp := time.Now().Format("20060102150405")
	uniqueFilename := timestamp + "_" + handler.Filename
	filePath := uploadDir + "/" + uniqueFilename

	// Create the file
	newFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error creating file on server", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Copy the uploaded file to the new file
	if _, err := io.Copy(newFile, file); err != nil {
		http.Error(w, "Error writing file to server", http.StatusInternalServerError)
		return
	}

	// Log the upload
	log.Printf("File uploaded: %s (path: %s)", handler.Filename, filePath)

	// Respond with the URL to access the file
	accessURL := fmt.Sprintf("/files/%s", uniqueFilename)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully! Access it at: %s\n", accessURL)
}

// serveFileHandler serves uploaded files
func serveFileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the filename from the URL
	// r.URL.Path = "/files/filename.txt"
	// We want "filename.txt"
	filename := r.URL.Path[len("/files/"):]

	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	filePath := uploadDir + "/" + filename

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}
