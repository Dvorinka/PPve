package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppName    = "FolderOpener"
	AppVersion = "1.0.0"
	ServerPort = "8765"
)

var (
	logFile *os.File
	logger  *log.Logger
)

func main() {
	// Set up logging
	setupLogging()

	// Print startup message
	fmt.Printf("%s v%s starting up\n", AppName, AppVersion)

	// Start the HTTP server
	startServer()
}

func setupLogging() {
	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(os.Getenv("APPDATA"), AppName, "logs")
	os.MkdirAll(logsDir, 0755)

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logsDir, fmt.Sprintf("%s.log", timestamp))

	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Set up logger
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Printf("%s v%s starting up", AppName, AppVersion)
}

func startServer() {
	// Set up HTTP server
	http.HandleFunc("/open", openFolderHandler)

	// Start server on the specified port
	serverAddr := fmt.Sprintf(":%s", ServerPort)
	logger.Printf("Folder opener server running on http://localhost%s", serverAddr)

	// Log to console as well
	fmt.Printf("Folder opener server running on http://localhost%s\n", serverAddr)

	// Start the server
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		logger.Fatalf("Server error: %v", err)
	}
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
	logger.Printf("Opening folder: %s", folderPath)
	
	// Properly handle backslashes in Windows paths
	// First, replace any forward slashes with backslashes
	folderPath = strings.ReplaceAll(folderPath, "/", "\\")
	
	// Fix any double backslashes that might have been created by JavaScript escaping
	for strings.Contains(folderPath, "\\\\") {
		folderPath = strings.ReplaceAll(folderPath, "\\\\", "\\")
	}
	
	// Log the cleaned path
	logger.Printf("Cleaned path: %s", folderPath)
	
	// Try multiple methods to open the folder
	successful := false
	
	// Method 1: Direct explorer.exe call
	cmd := exec.Command("explorer.exe", folderPath)
	err := cmd.Start()
	if err == nil {
		logger.Printf("Opened folder using direct method")
		successful = true
	} else {
		logger.Printf("Error opening folder with direct method: %v", err)
	}
	
	// Method 2: Using /root parameter if Method 1 failed
	if !successful {
		cmd = exec.Command("explorer.exe", "/root," + folderPath)
		err = cmd.Start()
		if err == nil {
			logger.Printf("Opened folder using /root parameter")
			successful = true
		} else {
			logger.Printf("Error opening folder with /root parameter: %v", err)
		}
	}
	
	// Method 3: Try with shell execute
	if !successful {
		cmd = exec.Command("cmd.exe", "/c", "start", "", folderPath)
		err = cmd.Start()
		if err == nil {
			logger.Printf("Opened folder using shell execute")
			successful = true
		} else {
			logger.Printf("Error opening folder with shell execute: %v", err)
		}
	}
	
	// Return appropriate response
	w.Header().Set("Content-Type", "application/json")
	
	if successful {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"success","message":"Opening folder: %s"}`, folderPath)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status":"error","message":"Failed to open folder: %s"}`, folderPath)
	}
}
