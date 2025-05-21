package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func main() {
	// Set up HTTP server
	http.HandleFunc("/open", openFolderHandler)

	// Start server on port 8080
	fmt.Println("Folder opener server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func openFolderHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers to allow requests from any origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET requests
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the folder path from the query parameter
	folderPath := r.URL.Query().Get("path")
	if folderPath == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	// Log the request
	fmt.Printf("Opening folder: %s\n", folderPath)

	// Open the folder in Windows Explorer
	// The /select flag opens Explorer with the specified folder selected
	cmd := exec.Command("explorer.exe", folderPath)
	err := cmd.Start()

	if err != nil {
		// If there was an error, try to clean the path and retry
		cleanPath := strings.ReplaceAll(folderPath, "/", "\\")
		cmd = exec.Command("explorer.exe", cleanPath)
		err = cmd.Start()

		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening folder: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Opening folder: %s", folderPath)
}
