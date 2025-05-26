package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

type BannerStyle struct {
	BackgroundColor string `json:"backgroundColor"`
	TextColor       string `json:"textColor"`
	FontSize        string `json:"fontSize"`
	TextAlign       string `json:"textAlign"`
	Padding         string `json:"padding"`
	IsVisible       bool   `json:"isVisible"`
}

type BannerContent struct {
	Text  string      `json:"text"`
	Style BannerStyle `json:"style"`
}

var (
	banner     BannerContent
	bannerLock sync.RWMutex
	bannerFile = "banner.json"
)

func init() {
	// Initialize with default values
	banner = BannerContent{
		Text: "Důležité oznámení: Tento banner lze upravit v administraci.",
		Style: BannerStyle{
			BackgroundColor: "#f8d7da",
			TextColor:       "#721c24",
			FontSize:        "16px",
			TextAlign:       "center",
			Padding:         "10px",
			IsVisible:       true,
		},
	}
	loadBanner()
}

func loadBanner() {
	if _, err := os.Stat(bannerFile); os.IsNotExist(err) {
		saveBanner()
		return
	}

	data, err := os.ReadFile(bannerFile)
	if err != nil {
		log.Printf("Error reading banner file: %v", err)
		return
	}

	bannerLock.Lock()
	defer bannerLock.Unlock()

	if err := json.Unmarshal(data, &banner); err != nil {
		log.Printf("Error parsing banner data: %v", err)
	}
}

func saveBanner() error {
	bannerLock.RLock()
	defer bannerLock.RUnlock()

	data, err := json.MarshalIndent(banner, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bannerFile, data, 0644)
}

func GetBannerHandler(w http.ResponseWriter, r *http.Request) {
	bannerLock.RLock()
	defer bannerLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(banner)
}

func UpdateBannerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newBanner BannerContent
	if err := json.NewDecoder(r.Body).Decode(&newBanner); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bannerLock.Lock()
	banner = newBanner
	err := saveBanner()
	bannerLock.Unlock()

	if err != nil {
		http.Error(w, "Error saving banner", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
