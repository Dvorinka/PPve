package main

import (
	"encoding/json"
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
	Text  string      `json:"text"`
	Image string      `json:"image,omitempty"`
	Link  string      `json:"link,omitempty"`
	Style BannerStyle `json:"style"`
}

type BannerStyle struct {
	BackgroundColor string `json:"backgroundColor"`
	TextColor       string `json:"textColor"`
	TextAlign       string `json:"textAlign"`
	FontSize        string `json:"fontSize"`
	Padding         string `json:"padding"`
	Margin          string `json:"margin"`
	BorderRadius    string `json:"borderRadius"`
	IsVisible       bool   `json:"isVisible"`
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

func saveBannerData() error {
	bannerLock.Lock()
	defer bannerLock.Unlock()

	data, err := json.MarshalIndent(banner, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(bannerDataFile, data, 0644)
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
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form for file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	newBanner := BannerContent{
		Text: r.FormValue("text"),
		Link: r.FormValue("link"),
		Style: BannerStyle{
			BackgroundColor: r.FormValue("style[backgroundColor]"),
			TextColor:       r.FormValue("style[textColor]"),
			TextAlign:       r.FormValue("style[textAlign]"),
			FontSize:        r.FormValue("style[fontSize]"),
			Padding:         r.FormValue("style[padding]"),
			Margin:          r.FormValue("style[margin]"),
			BorderRadius:    r.FormValue("style[borderRadius]"),
			IsVisible:       r.FormValue("style[isVisible]") == "true",
		},
	}

	// Handle file upload
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Ensure uploads directory exists
		if err := ensureDirs(); err != nil {
			http.Error(w, "Error preparing upload directory", http.StatusInternalServerError)
			return
		}

		// Create a new file in the uploads directory with a unique name
		ext := filepath.Ext(handler.Filename)
		tempFile, err := ioutil.TempFile(uploadDir, "upload-*"+ext)
		if err != nil {
			http.Error(w, "Error creating file", http.StatusInternalServerError)
			return
		}
		defer tempFile.Close()

		// Copy the uploaded file to the destination file
		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		// Update banner data with the new image path
		newBanner.Image = "/uploads/" + filepath.Base(tempFile.Name())
	} else if r.FormValue("removeImage") == "true" {
		// If removeImage is set, clear the image
		newBanner.Image = ""
	} else {
		// Keep the existing image if no new one is uploaded
		bannerLock.RLock()
		newBanner.Image = banner.Image
		bannerLock.RUnlock()
	}

	// Update banner data
	bannerLock.Lock()
	banner = newBanner
	err = saveBannerData()
	bannerLock.Unlock()

	if err != nil {
		http.Error(w, "Error saving banner data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(banner)
}

// ServeUploads handles serving uploaded files
func ServeUploads(w http.ResponseWriter, r *http.Request) {
	// Only serve files from the uploads directory
	if !strings.HasPrefix(r.URL.Path, "/"+uploadDir+"/") {
		http.NotFound(w, r)
		return
	}
	// Strip the leading slash to get the relative path
	http.ServeFile(w, r, r.URL.Path[1:])
}
