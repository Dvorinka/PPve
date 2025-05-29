package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Initialize banner data
func init() {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: Failed to create data directory: %v", err)
	}

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Warning: Failed to create uploads directory: %v", err)
	}

	// Load banner data from file if it exists
	if data, err := ioutil.ReadFile(bannerDataFile); err == nil {
		if err := json.Unmarshal(data, &banner); err != nil {
			log.Printf("Error loading banner data: %v", err)
			initDefaultBanner()
		}
	} else {
		initDefaultBanner()
	}
}

const (
	bannerDataFile = "data/banner.json"
	uploadDir      = "uploads"
)

// Ensure directories exist
func ensureDirs() error {
	if err := os.MkdirAll(filepath.Dir(bannerDataFile), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}
	return nil
}

type BannerContent struct {
	Text  string      `json:"Text"`
	Image string      `json:"Image,omitempty"`
	Link  string      `json:"Link,omitempty"`
	Style BannerStyle `json:"Style"`
}

type BannerStyle struct {
	BackgroundColor string `json:"BackgroundColor"`
	TextColor      string `json:"TextColor"`
	TextAlign      string `json:"TextAlign"`
	FontSize       string `json:"FontSize"`
	Padding        string `json:"Padding"`
	Margin         string `json:"Margin"`
	BorderRadius   string `json:"BorderRadius"`
	IsVisible      bool   `json:"IsVisible"`
	ImageWidth     int    `json:"ImageWidth"`    // Width in pixels
	ImageHeight    int    `json:"ImageHeight"`   // Height in pixels
	ImagePosition  string `json:"ImagePosition"` // center, left, right
	ImageX         int    `json:"ImageX"`        // X position for custom
	ImageY         int    `json:"ImageY"`        // Y position for custom
	Background     string `json:"Background,omitempty"`
	ContainerStyle string `json:"ContainerStyle,omitempty"`
}

var (
	banner     BannerContent
	bannerLock sync.RWMutex
)

func init() {
	// Ensure directories exist
	if err := ensureDirs(); err != nil {
		log.Printf("Warning: Failed to create required directories: %v", err)
	}

	// Load banner data from file if it exists
	if data, err := ioutil.ReadFile(bannerDataFile); err == nil {
		if err := json.Unmarshal(data, &banner); err != nil {
			log.Printf("Error loading banner data: %v", err)
			initDefaultBanner()
		}
	} else {
		initDefaultBanner()
	}
}

func initDefaultBanner() {
	banner = BannerContent{
		Text: "Vítejte na našem webu!",
		Style: BannerStyle{
			BackgroundColor: "#f8d7da",
			TextColor:       "#721c24",
			TextAlign:       "center",
			FontSize:        "18px",
			Padding:         "20px",
			Margin:          "20px",
			BorderRadius:    "8px",
			IsVisible:       true,
		},
	}
	saveBannerData()
}

// Helper function to convert string to int with default value
func parseIntOrDefault(str string, defaultValue int) int {
	if str == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return value
}

func saveBannerData() error {
	bannerLock.Lock()
	defer bannerLock.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(bannerDataFile), 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(banner, "", "  ")
	if err != nil {
		log.Printf("Error marshaling banner data to JSON: %v", err)
		return fmt.Errorf("failed to marshal banner data: %w", err)
	}

	if err := ioutil.WriteFile(bannerDataFile, data, 0644); err != nil {
		log.Printf("Error writing banner data to file %s: %v", bannerDataFile, err)
		return fmt.Errorf("failed to write banner data to file: %w", err)
	}

	log.Printf("Successfully saved banner data to %s with content: %+v", bannerDataFile, banner)
	return nil
}

func GetBannerHandler(w http.ResponseWriter, r *http.Request) {
	bannerLock.RLock()
	defer bannerLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(banner)
}

func UpdateBannerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form for file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		log.Printf("Error parsing form data: %v", err)
		http.Error(w, "Error parsing form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Log form values for debugging
	log.Printf("Form values: %+v", r.Form)

	// Create a new banner with default values
	newBanner := BannerContent{
		Text: r.FormValue("Text"),
		Link: r.FormValue("Link"),
		Style: BannerStyle{
			// Parse style values from form
			BackgroundColor: r.FormValue("style[BackgroundColor]"),
			TextColor:       r.FormValue("style[TextColor]"),
			TextAlign:       r.FormValue("style[TextAlign]"),
			FontSize:        r.FormValue("style[FontSize]"),
			Padding:         r.FormValue("style[Padding]"),
			Margin:          r.FormValue("style[Margin]"),
			BorderRadius:    r.FormValue("style[BorderRadius]"),
			IsVisible:       r.FormValue("IsVisible") == "true" || r.FormValue("style[IsVisible]") == "true",
			// Add image position fields
			ImagePosition: r.FormValue("style[ImagePosition]"),
			ImageX:        parseIntOrDefault(r.FormValue("style[ImageX]"), 0),
			ImageY:        parseIntOrDefault(r.FormValue("style[ImageY]"), 0),
			// Additional style fields
			Background:     r.FormValue("style[Background]"),
			ContainerStyle: r.FormValue("style[ContainerStyle]"),
		},
	}

	// Handle the background style (gradient or solid color)
	if bgStyle := r.FormValue("style[background]"); bgStyle != "" {
		if strings.Contains(bgStyle, "gradient") {
			// If it's a gradient, use it as is
			newBanner.Style.BackgroundColor = bgStyle
		} else if bgStyle != newBanner.Style.BackgroundColor {
			// If it's a different solid color, use it
			newBanner.Style.BackgroundColor = bgStyle
		}
	}

	// Handle container style if present
	if containerStyle := r.FormValue("style[containerStyle]"); containerStyle != "" {
		// You can parse additional styles from containerStyle if needed
		log.Printf("Container style received: %s", containerStyle)
	}

	// Log the banner data for debugging
	log.Printf("Parsed banner data: %+v", newBanner)

	// Handle file upload
	file, handler, err := r.FormFile("image")
	if err == nil {
		log.Println("Processing file upload...")
		defer file.Close()

		// Ensure uploads directory exists
		if err := ensureDirs(); err != nil {
			log.Printf("Error ensuring directories exist: %v", err)
			http.Error(w, "Error preparing upload directory", http.StatusInternalServerError)
			return
		}

		// Get the file extension
		ext := filepath.Ext(handler.Filename)
		if ext == "" {
			ext = ".jpg" // Default extension if none provided
		}

		// Create a new file in the uploads directory with a unique name
		tempFile, err := ioutil.TempFile(uploadDir, "banner-*"+ext)
		if err != nil {
			log.Printf("Error creating temp file: %v", err)
			http.Error(w, "Error creating file", http.StatusInternalServerError)
			return
		}
		defer tempFile.Close()

		// Copy the uploaded file to the destination file
		if _, err := io.Copy(tempFile, file); err != nil {
			log.Printf("Error saving file: %v", err)
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		// Get the relative path for the web
		relPath := "/uploads/" + filepath.Base(tempFile.Name())
		log.Printf("File uploaded successfully: %s", relPath)
		newBanner.Image = relPath
	} else if r.FormValue("removeImage") == "true" {
		// If removeImage is set, clear the image
		log.Println("Removing banner image")
		newBanner.Image = ""
	} else {
		// Keep the existing image if no new one is uploaded
		bannerLock.RLock()
		newBanner.Image = banner.Image
		bannerLock.RUnlock()
		log.Printf("Keeping existing image: %s", newBanner.Image)
	}

	// Update banner data
	bannerLock.Lock()
	banner = newBanner
	bannerLock.Unlock()

	// Save banner data
	err = saveBannerData()
	if err != nil {
		log.Printf("Error saving banner data: %v", err)
		http.Error(w, "Error saving banner data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Banner updated successfully with image: %s", newBanner.Image)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(banner)
}

// ServeUploads handles serving uploaded files
func ServeUploads(w http.ResponseWriter, r *http.Request) {
	// Clean the path to prevent directory traversal
	path := filepath.Clean(r.URL.Path)

	// Only serve files from the uploads directory
	if !strings.HasPrefix(path, "/uploads/") {
		log.Printf("Access denied: %s is outside uploads directory", path)
		http.NotFound(w, r)
		return
	}

	// Remove the /uploads/ prefix to get the actual file path
	filename := filepath.Join(uploadDir, strings.TrimPrefix(path, "/uploads/"))

	// Ensure the file exists and is within the uploads directory
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("File not found: %s", filename)
		http.NotFound(w, r)
		return
	}

	// Set cache control headers (cache for 1 day)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	// Set content type based on file extension
	ext := filepath.Ext(filename)
	if ext == ".jpg" || ext == ".jpeg" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else if ext == ".png" {
		w.Header().Set("Content-Type", "image/png")
	} else if ext == ".gif" {
		w.Header().Set("Content-Type", "image/gif")
	} else if ext == ".svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, filename)
	log.Printf("Served file: %s", filename)
}
