package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/gomail.v2"
)

type Visitor struct {
	FirstVisit time.Time `json:"first_visit"`
	LastVisit  time.Time `json:"last_visit"`
	Visits     int       `json:"visits"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
}

type VisitorStats struct {
	TotalVisits    int           `json:"total_visits"`
	TodayVisits    int           `json:"today_visits"`
	LastVisit      time.Time     `json:"last_visit"`
	MonthlyVisits  int           `json:"monthly_visits"`
	WeeklyVisits   int           `json:"weekly_visits"`
	LastUpdated    time.Time     `json:"last_updated"`
	UniqueVisitors map[string]*Visitor `json:"unique_visitors"`
	MostActiveHours []struct {
		Hour int   `json:"hour"`
		Count int  `json:"count"`
	} `json:"most_active_hours"`
	MostActiveDays []struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	} `json:"most_active_days"`
	BrowserStats map[string]int `json:"browser_stats"`
	OSStats      map[string]int `json:"os_stats"`
	ReferrerStats map[string]int `json:"referrer_stats"`
}

// Initialize VisitorStats with proper struct types
func (v *VisitorStats) init() {
	if v.TotalVisits == 0 {
		v.TotalVisits = 0
		v.TodayVisits = 0
		v.MonthlyVisits = 0
		v.WeeklyVisits = 0
		v.LastVisit = time.Now()
		v.LastUpdated = time.Now()
		v.UniqueVisitors = make(map[string]*Visitor)
		v.MostActiveHours = make([]struct {
			Hour int   `json:"hour"`
			Count int  `json:"count"`
		}, 0)
	}
	
	if len(v.MostActiveDays) == 0 {
		v.MostActiveDays = make([]struct {
			Day   string `json:"day"`
			Count int    `json:"count"`
		}, 7)
	}
}

const visitorStatsFile = "data/visitor_stats.json"

func loadVisitorStats() (*VisitorStats, error) {
	stats := &VisitorStats{
		TotalVisits:    0,
		TodayVisits:    0,
		MonthlyVisits:  0,
		WeeklyVisits:   0,
		LastVisit:      time.Now(),
		LastUpdated:    time.Now(),
	}
	
	// Initialize all fields
	stats.init()
	
	data, err := os.ReadFile(visitorStatsFile)
	if err != nil {
		return stats, nil // Return default stats if file doesn't exist
	}
	
	if err := json.Unmarshal(data, stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal visitor stats: %v", err)
	}
	
	return stats, nil
}

func saveVisitorStats(stats *VisitorStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal visitor stats: %v", err)
	}
	
	return os.WriteFile(visitorStatsFile, data, 0644)
}

func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    bytes := make([]byte, n)
    for i := range bytes {
        num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
        if err != nil {
            log.Printf("Error generating random number: %v", err)
            return ""
        }
        bytes[i] = letters[num.Int64()]
    }
    return string(bytes)
}

func getVisitorId(w http.ResponseWriter, r *http.Request) string {
    // Check if visitor ID is in cookie
    cookie, err := r.Cookie("visitor_id")
    if err == nil && cookie.Value != "" {
        return cookie.Value
    }
    
    // Generate a new unique ID
    visitorId := fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String() + randomString(16))))
    
    // Set the cookie to expire in 1 year
    http.SetCookie(w, &http.Cookie{
        Name:    "visitor_id",
        Value:   visitorId,
        Path:    "/",
        MaxAge:  365 * 24 * 60 * 60, // 1 year
        HttpOnly: true,
    })
    
    return visitorId
}

func trackVisit(w http.ResponseWriter, r *http.Request) {
    stats, err := loadVisitorStats()
    if err != nil {
        log.Printf("Error loading visitor stats: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    // Initialize stats if needed
    stats.init()
    
    // Get visitor info
    visitorId := getVisitorId(w, r)
    ip := r.RemoteAddr
    userAgent := r.UserAgent()
    
    // Update visit counts
    stats.TotalVisits++
    stats.TodayVisits++
    stats.LastVisit = time.Now()
    stats.LastUpdated = time.Now()
    
    // Reset weekly visits on Monday
    if time.Now().Weekday() == time.Monday && stats.LastUpdated.Weekday() != time.Monday {
        stats.WeeklyVisits = 1
    } else {
        stats.WeeklyVisits++
    }
    
    // Reset monthly visits at the start of the month
    if stats.LastUpdated.Month() != time.Now().Month() {
        stats.MonthlyVisits = 1
    } else {
        stats.MonthlyVisits++
    }
    
    // Update unique visitors
    if _, ok := stats.UniqueVisitors[visitorId]; !ok {
        stats.UniqueVisitors[visitorId] = &Visitor{
            FirstVisit: time.Now(),
            LastVisit:  time.Now(),
            Visits:     1,
            IP:         ip,
            UserAgent:  userAgent,
        }
    } else {
        visitor := stats.UniqueVisitors[visitorId]
        visitor.LastVisit = time.Now()
        visitor.Visits++
        stats.UniqueVisitors[visitorId] = visitor
    }
    
    // Update browser stats
    browser := detectBrowser(userAgent)
    if stats.BrowserStats == nil {
        stats.BrowserStats = make(map[string]int)
    }
    stats.BrowserStats[browser]++
    
    // Update OS stats
    os := detectOS(userAgent)
    if stats.OSStats == nil {
        stats.OSStats = make(map[string]int)
    }
    stats.OSStats[os]++
    
    // Update referrer stats
    referrer := r.Referer()
    if referrer != "" {
        if stats.ReferrerStats == nil {
            stats.ReferrerStats = make(map[string]int)
        }
        stats.ReferrerStats[referrer]++
    }
    
    // Update active hours
    hour := time.Now().Hour()
    found := false
    for i, h := range stats.MostActiveHours {
        if h.Hour == hour {
            stats.MostActiveHours[i].Count++
            found = true
            break
        }
    }
    if !found {
        stats.MostActiveHours = append(stats.MostActiveHours, struct {
            Hour int   `json:"hour"`
            Count int  `json:"count"`
        }{Hour: hour, Count: 1})
    }
    
    // Update active days
    day := time.Now().Weekday().String()
    found = false
    for i, d := range stats.MostActiveDays {
        if d.Day == day {
            stats.MostActiveDays[i].Count++
            found = true
            break
        }
    }
    if !found {
        stats.MostActiveDays = append(stats.MostActiveDays, struct {
            Day   string `json:"day"`
            Count int    `json:"count"`
        }{Day: day, Count: 1})
    }
    
    // Save visitor stats
    if err := saveVisitorStats(stats); err != nil {
        log.Printf("Error saving visitor stats: %v", err)
    }
}

// Helper function to extract browser from User-Agent
func detectBrowser(userAgent string) string {
    userAgent = strings.ToLower(userAgent)
    if strings.Contains(userAgent, "chrome") {
        return "Chrome"
    } else if strings.Contains(userAgent, "safari") && !strings.Contains(userAgent, "chrome") {
        return "Safari"
    } else if strings.Contains(userAgent, "firefox") {
        return "Firefox"
    } else if strings.Contains(userAgent, "msie") || strings.Contains(userAgent, "trident") {
        return "Internet Explorer"
    } else if strings.Contains(userAgent, "edge") {
        return "Edge"
    } else if strings.Contains(userAgent, "opera") {
        return "Opera"
    } else {
        return "Unknown"
    }
}

// Helper function to extract OS from User-Agent
func detectOS(userAgent string) string {
    userAgent = strings.ToLower(userAgent)
    if strings.Contains(userAgent, "windows") {
        return "Windows"
    } else if strings.Contains(userAgent, "mac os") || strings.Contains(userAgent, "macintosh") {
        return "macOS"
    } else if strings.Contains(userAgent, "iphone") || strings.Contains(userAgent, "ipad") || strings.Contains(userAgent, "ipod") {
        return "iOS"
    } else if strings.Contains(userAgent, "android") {
        return "Android"
    } else if strings.Contains(userAgent, "linux") {
        return "Linux"
    } else if strings.Contains(userAgent, "bsd") {
        return "BSD"
    } else {
        return "Unknown"
    }
}

// Helper function to extract OS from User-Agent
func getOSFromUserAgent(userAgent string) string {
	return detectOS(userAgent)
}

// Get visitor stats
func getVisitorStats(w http.ResponseWriter, r *http.Request) {
	stats, err := loadVisitorStats()
	if err != nil {
		log.Printf("Error loading visitor stats: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding visitor stats: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// App structure for dashboard
// Note: This is a duplicate of the struct in admin-dashboard.html
// Consider moving this to a shared package if needed in multiple files
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`      // For file uploads
	IconClass   string `json:"iconClass,omitempty"` // For Font Awesome icons
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Trip entry structure
// Note: This is a duplicate of the struct in admin-dashboard.html
// Consider moving this to a shared package if needed in multiple files
type TripEntry struct {
	Name        string     `json:"name"`
	Vehicle     string     `json:"vehicle"`
	Destination string     `json:"destination"`
	DateStart   string     `json:"date_start"`
	TimeStart   string     `json:"time_start"`
	DateEnd     string     `json:"date_end"`
	TimeEnd     string     `json:"time_end"`
	Purpose     string     `json:"purpose"`
	KmStart     int        `json:"km_start"`
	KmEnd       int        `json:"km_end"`
	Coordinates *GeoCoords `json:"coordinates,omitempty"`
}

// Geo coordinates structure
// Note: This is a duplicate of the struct in admin-dashboard.html
// Consider moving this to a shared package if needed in multiple files
type GeoCoords struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

// Reservation structure
// Note: This is a duplicate of the struct in admin-dashboard.html
// Consider moving this to a shared package if needed in multiple files
type Reservation struct {
	ID         string `json:"id"`
	DriverName string `json:"driverName"`
	Vehicle    string `json:"vehicle"`
	StartDate  string `json:"startDate"`
	StartTime  string `json:"startTime"`
	EndDate    string `json:"endDate"`
	EndTime    string `json:"endTime"`
	Purpose    string `json:"purpose,omitempty"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create necessary directories
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	r := mux.NewRouter()

    // Visitor tracking endpoints
    r.HandleFunc("/api/track-visit", trackVisit).Methods("GET")
    r.HandleFunc("/api/visitor-stats", getVisitorStats).Methods("GET")

	// Set up reverse proxy to kontakt service
	kontaktURL, _ := url.Parse("http://webportal:8080")
	kontaktProxy := httputil.NewSingleHostReverseProxy(kontaktURL)

	// CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow all origins for development
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Auth middleware
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for GET requests and OPTIONS
			if r.Method == "GET" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization token", http.StatusUnauthorized)
				return
			}

			// Verify token (in a real app, you would validate this against your auth system)
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			// In a real app, you would validate the token here
			next.ServeHTTP(w, r)
		})
	}

	// Apply CORS middleware to all routes
	r.Use(corsMiddleware)

	// Public routes
	r.PathPrefix("/kontakt/").Handler(http.StripPrefix("/kontakt", kontaktProxy))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET", "OPTIONS")

	// Authentication routes
	r.HandleFunc("/api/login", LoginHandler).Methods("POST", "OPTIONS")

	// Public endpoints (must be defined before protected ones)
	r.HandleFunc("/api/banner", GetBannerHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/submit", handleSubmit).Methods("POST", "OPTIONS") // Public submit endpoint for evidence-aut.html

	// Add redirect for /rezervace-aut to /rezervace-aut.html
	r.HandleFunc("/rezervace-aut", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "rezervace-aut.html")
	}).Methods("GET")

	// Make reservation endpoints public by moving them outside of the protected API routes
	r.HandleFunc("/api/reservations", handleGetReservations).Methods("GET")
	r.HandleFunc("/api/reservations", handleCreateReservation).Methods("POST")
	r.HandleFunc("/api/check-availability", handleCheckAvailability).Methods("GET")

	// Add these new routes after existing reservation endpoints
	r.HandleFunc("/api/reservations/{id}", handleUpdateReservation).Methods("PUT")
	r.HandleFunc("/api/reservations/{id}", handleDeleteReservation).Methods("DELETE")

	// Protected API routes with auth middleware
	api := r.PathPrefix("/api").Subrouter()
	api.Use(authMiddleware)

	// Protected API endpoints
	api.HandleFunc("/submit", handleSubmit).Methods("POST")
	api.HandleFunc("/banner/update", UpdateBannerHandler).Methods("POST", "OPTIONS")

	// App management routes
	api.HandleFunc("/apps", GetAppsHandler).Methods("GET")
	api.HandleFunc("/apps", CreateAppHandler).Methods("POST")
	api.HandleFunc("/apps/{id}", GetAppHandler).Methods("GET")
	api.HandleFunc("/apps/{id}", UpdateAppHandler).Methods("PUT")
	api.HandleFunc("/apps/{id}", DeleteAppHandler).Methods("DELETE")

	// Admin routes - defined before the catch-all static file server
	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin.html")
	}).Methods("GET")

	r.HandleFunc("/admin/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin-dashboard.html")
	}).Methods("GET")

	// Redirect root to index.html
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
		} else {
			// Let the static file server handle other root-level paths
			http.FileServer(http.Dir(".")).ServeHTTP(w, r)
		}
	}).Methods("GET")

	// Public route for evidence-aut.html
	r.HandleFunc("/evidence-aut", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "evidence-aut.html")
	}).Methods("GET")

	// Contact page route
	r.HandleFunc("/kontakt", contactHandler).Methods("GET")

	// Static file server for all other routes - must be the last route defined
	fs := http.FileServer(http.Dir("."))
	r.PathPrefix("/").Handler(fs)

	// Apply CORS middleware to all routes
	handler := enableCORS(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Server běží na portu %s", port)
	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatalf("Chyba při spuštění serveru: %v", err)
	}
}

// contactHandler handles the contact page request
func contactHandler(w http.ResponseWriter, r *http.Request) {
	// Check if kontakt service is already running
	resp, err := http.Get("http://webportal:8080/health")
	if err == nil && resp.StatusCode == 200 {
		http.Redirect(w, r, "http://webportal:8080/", http.StatusFound)
		return
	}

	// Start the service if not running
	cmd := exec.Command("make", "dev")
	cmd.Dir = "kontakt"
	err = cmd.Start()
	if err != nil {
		http.Error(w, "Failed to start kontakt service", http.StatusInternalServerError)
		return
	}

	// Wait briefly for service to start
	time.Sleep(2 * time.Second)
	http.Redirect(w, r, "http://webportal:8080/", http.StatusFound)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// File path for storing apps
const appsFile = "data/apps.json"

// loadApps loads apps from the JSON file
func loadApps() ([]App, error) {
	var apps []App

	// Check if file exists
	if _, err := os.Stat(appsFile); os.IsNotExist(err) {
		// Return empty slice if file doesn't exist
		return []App{}, nil
	}

	// Read file
	data, err := os.ReadFile(appsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading apps file: %v", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, fmt.Errorf("error parsing apps JSON: %v", err)
	}

	return apps, nil
}

// saveApps saves apps to the JSON file
func saveApps(apps []App) error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(appsFile), 0755); err != nil {
		return fmt.Errorf("error creating data directory: %v", err)
	}

	// Marshal to pretty-printed JSON
	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling apps to JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(appsFile, data, 0644); err != nil {
		return fmt.Errorf("error writing apps file: %v", err)
	}

	return nil
}

// Reservation Handlers
func handleGetReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := loadReservations()
	if err != nil {
		http.Error(w, "Nepodařilo se načíst rezervace", http.StatusInternalServerError)
		return
	}

	// Get current time
	now := time.Now()

	// Filter out expired reservations (ended more than 24 hours ago)
	var activeReservations []Reservation
	for _, res := range reservations {
		// Parse end time
		endDateTime, err := time.Parse("2006-01-02 15:04",
			fmt.Sprintf("%s %s", res.EndDate, res.EndTime))
		if err != nil {
			continue
		} // Keep reservation if it ended less than 48 hours ago
		if endDateTime.Add(48 * time.Hour).After(now) {
			activeReservations = append(activeReservations, res)
		}
	}

	// If we filtered out any reservations, save the cleaned up list
	if len(activeReservations) < len(reservations) {
		err = saveReservations(activeReservations)
		if err != nil {
			log.Printf("Error saving cleaned up reservations: %v", err)
		}
	}

	// Convert active reservations to calendar events
	type Event struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Start      string `json:"start"`
		End        string `json:"end"`
		DriverName string `json:"driverName"`
		Vehicle    string `json:"vehicle"`
		Purpose    string `json:"purpose"`
	}

	var events []Event
	for _, res := range activeReservations {
		// Create proper ISO datetime strings
		startDateTime := fmt.Sprintf("%sT%s:00", res.StartDate, res.StartTime)
		endDateTime := fmt.Sprintf("%sT%s:00", res.EndDate, res.EndTime)

		events = append(events, Event{
			ID:         res.ID,
			Title:      fmt.Sprintf("%s - %s", res.Vehicle, res.DriverName),
			Start:      startDateTime,
			End:        endDateTime,
			DriverName: res.DriverName,
			Vehicle:    res.Vehicle,
			Purpose:    res.Purpose,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func handleCreateReservation(w http.ResponseWriter, r *http.Request) {
	var reservation Reservation
	if err := json.NewDecoder(r.Body).Decode(&reservation); err != nil {
		http.Error(w, "Invalid reservation data", http.StatusBadRequest)
		return
	}

	// Log received data for debugging
	log.Printf("Received reservation data: %+v", reservation)

	// Validate required fields
	if reservation.DriverName == "" || reservation.Vehicle == "" ||
		reservation.StartDate == "" || reservation.StartTime == "" ||
		reservation.EndDate == "" || reservation.EndTime == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create combined date-time string for validation
	startDateTime, err := time.Parse("2006-01-02 15:04",
		fmt.Sprintf("%s %s", reservation.StartDate, reservation.StartTime))
	if err != nil {
		log.Printf("Error parsing start date/time: %v", err)
		http.Error(w, "Invalid start date/time format", http.StatusBadRequest)
		return
	}

	endDateTime, err := time.Parse("2006-01-02 15:04",
		fmt.Sprintf("%s %s", reservation.EndDate, reservation.EndTime))
	if err != nil {
		log.Printf("Error parsing end date/time: %v", err)
		http.Error(w, "Invalid end date/time format", http.StatusBadRequest)
		return
	}

	// Validate time order
	if endDateTime.Before(startDateTime) {
		http.Error(w, "End time must be after start time", http.StatusBadRequest)
		return
	}

	// Check availability
	available, err := checkReservationAvailability(reservation.Vehicle, startDateTime, endDateTime)
	if err != nil {
		http.Error(w, "Failed to check availability", http.StatusInternalServerError)
		return
	}
	if !available {
		http.Error(w, "Selected time slot is not available", http.StatusConflict)
		return
	}

	// Generate unique ID
	reservation.ID = fmt.Sprintf("res_%d", time.Now().UnixNano())

	// Save reservation
	reservations, err := loadReservations()
	if err != nil {
		http.Error(w, "Failed to load reservations", http.StatusInternalServerError)
		return
	}

	reservations = append(reservations, reservation)

	if err := saveReservations(reservations); err != nil {
		http.Error(w, "Failed to save reservation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reservation)
}

// Add helper function to check availability
func checkReservationAvailability(vehicle string, start, end time.Time) (bool, error) {
	reservations, err := loadReservations()
	if err != nil {
		return false, err
	}

	for _, res := range reservations {
		resStart, err := time.Parse("2006-01-02 15:04",
			fmt.Sprintf("%s %s", res.StartDate, res.StartTime))
		if err != nil {
			continue
		}

		resEnd, err := time.Parse("2006-01-02 15:04",
			fmt.Sprintf("%s %s", res.EndDate, res.EndTime))
		if err != nil {
			continue
		}

		if res.Vehicle == vehicle &&
			!(end.Before(resStart) || start.After(resEnd)) {
			return false, nil
		}
	}

	return true, nil
}

func handleCheckAvailability(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	vehicle := r.URL.Query().Get("vehicle")
	startDate := r.URL.Query().Get("startDate")
	startTime := r.URL.Query().Get("startTime")
	endDate := r.URL.Query().Get("endDate")
	endTime := r.URL.Query().Get("endTime")

	// Validate inputs
	if vehicle == "" || startDate == "" || startTime == "" || endDate == "" || endTime == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Parse the dates with specific format
	startDateTime, err := time.Parse("2006-01-02 15:04",
		fmt.Sprintf("%s %s", startDate, startTime))
	if err != nil {
		http.Error(w, "Invalid start date/time format", http.StatusBadRequest)
		return
	}

	endDateTime, err := time.Parse("2006-01-02 15:04",
		fmt.Sprintf("%s %s", endDate, endTime))
	if err != nil {
		http.Error(w, "Invalid end date/time format", http.StatusBadRequest)
		return
	}

	// Load existing reservations
	reservations, err := loadReservations()
	if err != nil {
		http.Error(w, "Failed to load reservations", http.StatusInternalServerError)
		return
	}

	// Check for conflicts
	available := true
	for _, res := range reservations {
		if res.Vehicle == vehicle {
			resStart, err := time.Parse("2006-01-02 15:04",
				fmt.Sprintf("%s %s", res.StartDate, res.StartTime))
			if err != nil {
				continue
			}

			resEnd, err := time.Parse("2006-01-02 15:04",
				fmt.Sprintf("%s %s", res.EndDate, res.EndTime))
			if err != nil {
				continue
			}

			// Check if there is any overlap
			if !(endDateTime.Before(resStart) || startDateTime.After(resEnd)) {
				available = false
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"available": available})
}

func loadReservations() ([]Reservation, error) {
	data, err := os.ReadFile("data/reservations.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []Reservation{}, nil
		}
		return nil, err
	}

	var reservations []Reservation
	if err := json.Unmarshal(data, &reservations); err != nil {
		return nil, err
	}

	return reservations, nil
}

func saveReservations(reservations []Reservation) error {
	data, err := json.MarshalIndent(reservations, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	return os.WriteFile("data/reservations.json", data, 0644)
}

// App Handlers
func GetAppsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET requests
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load apps from JSON file
	apps, err := loadApps()
	if err != nil {
		log.Printf("Error loading apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return empty array if no apps
	if apps == nil {
		apps = []App{}
	}

	// Add hardcoded apps
	hardcodedApps := []App{
		{
			ID:          "hardcoded-car",
			Name:        "Záznam služebních jízd",
			URL:         "/evidence-aut",
			Description: "Jednoduchý systém pro evidenci a správu jízd služebními vozidly.",
			Icon:        "fa-car-side",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
		{
			ID:          "hardcoded-lunch",
			Name:        "Objednávka obědů",
			URL:         "http://ppc-app/pwkweb2/",
			Description: "Portál pro objednávku a přehled firemních obědů",
			Icon:        "fa-utensils",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
		{
			ID:          "hardcoded-osticket",
			Name:        "OSTicket",
			URL:         "http://osticket/",
			Description: "Systém technické podpory a hlášení problémů",
			Icon:        "fa-headset",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
		{
			ID:          "hardcoded-kanboard",
			Name:        "Kanboard",
			URL:         "http://kanboard/",
			Description: "Správa úkolů a projektů v přehledném kanban stylu",
			Icon:        "fa-tasks",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
	}

	// Combine hardcoded and dynamic apps
	allApps := append(hardcodedApps, apps...)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allApps); err != nil {
		log.Printf("Error encoding apps to JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func GetAppHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	// Check if it's a hardcoded app
	if strings.HasPrefix(appID, "hardcoded-") {
		// Return 404 for non-existent hardcoded apps
		http.NotFound(w, r)
		return
	}

	// Load apps from file
	apps, err := loadApps()
	if err != nil {
		log.Printf("Error loading apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Find the app by ID
	var foundApp *App
	for i, app := range apps {
		if app.ID == appID {
			foundApp = &apps[i]
			break
		}
	}

	if foundApp == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(foundApp); err != nil {
		log.Printf("Error encoding app to JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func CreateAppHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Get form values
	name := r.FormValue("name")
	url := r.FormValue("url")
	description := r.FormValue("description")
	iconClass := r.FormValue("iconClass")

	// Validate required fields
	if name == "" || url == "" || iconClass == "" {
		http.Error(w, "Name, URL, and Icon are required", http.StatusBadRequest)
		return
	}

	// Create a new app
	app := App{
		ID:          fmt.Sprintf("app_%d", time.Now().UnixNano()),
		Name:        strings.TrimSpace(name),
		URL:         strings.TrimSpace(url),
		Description: strings.TrimSpace(description),
		IconClass:   iconClass,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	// Load existing apps
	apps, err := loadApps()
	if err != nil {
		log.Printf("Error loading apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add the new app to the list
	apps = append(apps, app)

	// Save the updated list of apps
	if err := saveApps(apps); err != nil {
		log.Printf("Error saving apps: %v", err)
		// No need to clean up files since we're not handling file uploads anymore
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(app); err != nil {
		log.Printf("Error encoding app to JSON: %v", err)
	}
}

func UpdateAppHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	// Prevent updating hardcoded apps
	if strings.HasPrefix(appID, "hardcoded-") {
		http.Error(w, "Cannot update hardcoded app", http.StatusForbidden)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Get form values
	name := r.FormValue("name")
	url := r.FormValue("url")
	description := r.FormValue("description")
	iconClass := r.FormValue("iconClass")

	// Validate required fields
	if name == "" || url == "" || iconClass == "" {
		http.Error(w, "Name, URL, and Icon are required", http.StatusBadRequest)
		return
	}

	// Load existing apps
	apps, err := loadApps()
	if err != nil {
		log.Printf("Error loading apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Find the app to update
	var found bool
	var updatedApps []App
	for _, app := range apps {
		if app.ID == appID {
			// Update the app
			app.Name = strings.TrimSpace(name)
			app.URL = strings.TrimSpace(url)
			app.Description = strings.TrimSpace(description)
			app.IconClass = iconClass
			app.UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
		}
		updatedApps = append(updatedApps, app)
	}

	if !found {
		http.Error(w, "App not found", http.StatusNotFound)
		return
	}

	// Save the updated apps
	if err := saveApps(updatedApps); err != nil {
		log.Printf("Error saving apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedApps); err != nil {
		log.Printf("Error encoding apps to JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func DeleteAppHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	// Prevent deleting hardcoded apps
	if strings.HasPrefix(appID, "hardcoded-") {
		http.Error(w, "Cannot delete hardcoded app", http.StatusForbidden)
		return
	}

	// Load existing apps
	apps, err := loadApps()
	if err != nil {
		log.Printf("Error loading apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Find the app to delete
	var appIndex = -1
	var iconToDelete string
	for i, app := range apps {
		if app.ID == appID {
			appIndex = i
			iconToDelete = app.Icon
			break
		}
	}

	if appIndex == -1 {
		http.NotFound(w, r)
		return
	}

	// Remove the app from the slice
	updatedApps := append(apps[:appIndex], apps[appIndex+1:]...)

	// Save the updated apps
	if err := saveApps(updatedApps); err != nil {
		log.Printf("Error saving apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If the app had an icon, check if it's used by any other app before deleting
	if iconToDelete != "" {
		// Check if any other app is using this icon
		iconInUse := false
		for _, app := range updatedApps {
			if app.Icon == iconToDelete {
				iconInUse = true
				break
			}
		}

		// Delete the icon file if it's not in use
		if !iconInUse {
			iconPath := filepath.Join("uploads", iconToDelete)
			if _, err := os.Stat(iconPath); err == nil {
				if err := os.Remove(iconPath); err != nil {
					log.Printf("Error deleting icon file: %v", err)
					// Continue even if we can't delete the file
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"Only POST method is allowed"}`))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Chyba při čtení těla požadavku: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Failed to read request body"}`))
		return
	}
	defer r.Body.Close()

	log.Printf("Přijatá data: %s", string(body))

	var entry TripEntry
	err = json.Unmarshal(body, &entry)
	if err != nil {
		log.Printf("Chyba při parsování JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":"Failed to parse JSON: %v"}`, err)))
		return
	}

	if entry.Name == "" || entry.Destination == "" || entry.DateStart == "" || entry.DateEnd == "" || entry.Purpose == "" {
		log.Printf("Chybějící povinná pole: %+v", entry)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Missing required fields"}`))
		return
	}

	if entry.KmEnd < entry.KmStart {
		log.Printf("Neplatný stav tachometru: %d -> %d", entry.KmStart, entry.KmEnd)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"End kilometers must be greater than or equal to start kilometers"}`))
		return
	}

	// Formátování dat do českého formátu
	czechMonths := []string{
		"ledna", "února", "března", "dubna", "května", "června",
		"července", "srpna", "září", "října", "listopadu", "prosince",
	}

	// Zpracování začátku cesty
	parsedDateStart, err := time.Parse("2006-01-02", entry.DateStart)
	if err != nil {
		log.Printf("Chyba při parsování data začátku: %v", err)
	}

	// Zpracování konce cesty
	parsedDateEnd, err := time.Parse("2006-01-02", entry.DateEnd)
	if err != nil {
		log.Printf("Chyba při parsování data konce: %v", err)
	}

	err = sendEmail(entry, parsedDateStart, parsedDateEnd, czechMonths)
	if err != nil {
		log.Printf("Chyba při odesílání emailu: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error":"Failed to send email: %v"}`, err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Záznam byl úspěšně uložen a email odeslán"}`))
}

func sendEmail(entry TripEntry, parsedDateStart, parsedDateEnd time.Time, czechMonths []string) error {
	smtpHost := "mail.pp-kunovice.cz"
	smtpPort := 465
	sender := "sluzebnicek@pp-kunovice.cz"
	password := "7g}qznB5bj"
	recipient := "sluzebnicek@pp-kunovice.cz"

	m := gomail.NewMessage()
	m.SetHeader("From", sender)
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", "Nový záznam o jízdě služebním autem")

	var htmlContent strings.Builder

	htmlContent.WriteString(`
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="UTF-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>Záznam o jízdě služebním autem</title>
      <style>
        @media only screen and (max-width: 620px) {
          .container {
            width: 100% !important;
            padding: 10px !important;
          }
          .content {
            padding: 15px !important;
          }
          .header {
            padding: 15px !important;
          }
          .info-row {
            display: block !important;
            width: 100% !important;
          }
          .info-item {
            width: 100% !important;
            margin-bottom: 10px !important;
          }
        }
        
        body {
          font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
          background-color: #f0f2f5;
          margin: 0;
          padding: 0;
          -webkit-font-smoothing: antialiased;
          -moz-osx-font-smoothing: grayscale;
        }
        
        .container {
          max-width: 600px;
          margin: 20px auto;
          background-color: #ffffff;
          border-radius: 8px;
          overflow: hidden;
          box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        .header {
          background-color: #004990;
          color: white;
          padding: 20px 25px;
          text-align: center;
        }
        
        .header h1 {
          margin: 0;
          font-size: 24px;
          font-weight: 600;
        }
        
        .content {
          padding: 25px;
        }
        
        .section {
          margin-bottom: 25px;
          border-bottom: 1px solid #eaeaea;
          padding-bottom: 15px;
        }
        
        .section:last-child {
          border-bottom: none;
          margin-bottom: 0;
          padding-bottom: 0;
        }
        
        .section-title {
          font-size: 18px;
          color: #004990;
          margin-bottom: 15px;
          font-weight: 600;
        }
        
        .info-row {
          display: flex;
          flex-wrap: wrap;
          margin-bottom: 10px;
        }
        
        .info-item {
          width: 48%;
          margin-bottom: 15px;
        }
        
        .label {
          font-weight: 600;
          color: #555;
          font-size: 14px;
          display: block;
          margin-bottom: 5px;
        }
        
        .value {
          color: #333;
          font-size: 16px;
        }
        
        .highlight {
          background-color: #f8f9fa;
          border-left: 3px solid #0072b0;
          padding: 10px 15px;
          margin: 15px 0;
        }
        
        .map-link {
          display: inline-block;
          margin-top: 10px;
          color: #0072b0;
          text-decoration: none;
          font-weight: 500;
        }
        
        .map-link:hover {
          text-decoration: underline;
        }
        
        .footer {
          text-align: center;
          padding: 15px;
          font-size: 12px;
          color: #777;
          background-color: #f8f9fa;
        }
      </style>
    </head>
    <body>
      <div class="container">
        <div class="header">
          <h1>Záznam o jízdě služebním autem</h1>
        </div>
        <div class="content">
    `)

	// Formátování dat a časů pro zobrazení
	formattedDateStart := ""
	if parsedDateStart.IsZero() == false {
		monthNameStart := czechMonths[parsedDateStart.Month()-1]
		formattedDateStart = fmt.Sprintf("%d. %s %d", parsedDateStart.Day(), monthNameStart, parsedDateStart.Year())
	} else {
		formattedDateStart = entry.DateStart
	}

	formattedDateEnd := ""
	if !parsedDateEnd.IsZero() {
		monthNameEnd := czechMonths[parsedDateEnd.Month()-1]
		formattedDateEnd = fmt.Sprintf("%d. %s %d", parsedDateEnd.Day(), monthNameEnd, parsedDateEnd.Year())
	} else {
		formattedDateEnd = entry.DateEnd
	}

	// Výpočet celkové doby jízdy
	startDateTime, startErr := time.Parse("2006-01-02T15:04", fmt.Sprintf("%sT%s", entry.DateStart, entry.TimeStart))
	endDateTime, endErr := time.Parse("2006-01-02T15:04", fmt.Sprintf("%sT%s", entry.DateEnd, entry.TimeEnd))

	totalDurationStr := "Neznámá"
	if startErr == nil && endErr == nil {
		diffMs := endDateTime.Sub(startDateTime)
		if diffMs >= 0 {
			diffDays := int(diffMs.Hours() / 24)
			diffHours := int(diffMs.Hours()) % 24
			diffMinutes := int(diffMs.Minutes()) % 60

			if diffDays > 0 {
				dayWord := "dní"
				if diffDays == 1 {
					dayWord = "den"
				} else if diffDays >= 2 && diffDays <= 4 {
					dayWord = "dny"
				}
				totalDurationStr = fmt.Sprintf("%d %s, %d h %d min", diffDays, dayWord, diffHours, diffMinutes)
			} else {
				totalDurationStr = fmt.Sprintf("%d h %d min", diffHours, diffMinutes)
			}
		}
	}

	// Vypsání informací o řidiči a vozidle
	htmlContent.WriteString(`<div class="section">
		<div class="section-title">Informace o řidiči a vozidle</div>
		<div class="info-row">
		  <div class="info-item">
			<span class="label">Řidič</span>
			<span class="value">` + entry.Name + `</span>
		  </div>
		  <div class="info-item">
			<span class="label">Vozidlo</span>
			<span class="value">` + entry.Vehicle + `</span>
		  </div>
		</div>
	  </div>`)

	// Vypsání informací o trase
	htmlContent.WriteString(`<div class="section">
		<div class="section-title">Informace o trase</div>
		<div class="info-row">
		  <div class="info-item">
			<span class="label">Cíl cesty</span>
			<span class="value">` + entry.Destination + `</span>
		  </div>
		  <div class="info-item">
			<span class="label">Účel jízdy</span>
			<span class="value">` + entry.Purpose + `</span>
		  </div>
		</div>
	  </div>`)

	// Vypsání informací o času
	htmlContent.WriteString(`<div class="section">
		<div class="section-title">Časové údaje</div>
		<div class="info-row">
		  <div class="info-item">
			<span class="label">Datum a čas odjezdu</span>
			<span class="value">` + formattedDateStart + `, ` + entry.TimeStart + `</span>
		  </div>
		  <div class="info-item">
			<span class="label">Datum a čas příjezdu</span>
			<span class="value">` + formattedDateEnd + `, ` + entry.TimeEnd + `</span>
		  </div>
		</div>
		<div class="highlight">
		  <span class="label">Celková doba jízdy</span>
		  <span class="value">` + totalDurationStr + `</span>
		</div>
	  </div>`)

	// Vypsání informací o kilometrech
	htmlContent.WriteString(`<div class="section">
		<div class="section-title">Stav tachometru</div>
		<div class="info-row">
		  <div class="info-item">
			<span class="label">Stav na začátku</span>
			<span class="value">` + fmt.Sprintf("%d km", entry.KmStart) + `</span>
		  </div>
		  <div class="info-item">
			<span class="label">Stav na konci</span>
			<span class="value">` + fmt.Sprintf("%d km", entry.KmEnd) + `</span>
		  </div>
		</div>
		<div class="highlight">
		  <span class="label">Celkem ujeto</span>
		  <span class="value">` + fmt.Sprintf("%d km", entry.KmEnd-entry.KmStart) + `</span>
		</div>
	  </div>`)

	if entry.Coordinates != nil {
		htmlContent.WriteString(`<div class="section">
		<div class="section-title">GPS Souřadnice</div>
		<div class="info-row">
		  <div class="info-item">
			<span class="label">Souřadnice</span>
			<span class="value">` + entry.Coordinates.Lat + `, ` + entry.Coordinates.Lng + `</span>
		  </div>
		</div>
		<a href="https://mapy.cz/zakladni?x=` + entry.Coordinates.Lng + `&y=` + entry.Coordinates.Lat + `&z=15" target="_blank" class="map-link">
		  <i class="fas fa-map-marker-alt"></i> Zobrazit na mapě
		</a>
	  </div>`)
	}

	htmlContent.WriteString(`
        </div>
        <div class="footer">
          &copy; 2025 Poppe + Potthoff - Automaticky generovaný email
        </div>
      </div>
    </body>
    </html>
    `)

	m.SetBody("text/html", htmlContent.String())

	d := gomail.NewDialer(smtpHost, smtpPort, sender, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return d.DialAndSend(m)
}

// Add these new handler functions before the existing banner handlers

func handleUpdateReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reservationID := vars["id"]

	var updatedReservation Reservation
	if err := json.NewDecoder(r.Body).Decode(&updatedReservation); err != nil {
		http.Error(w, "Invalid reservation data", http.StatusBadRequest)
		return
	}

	// Load existing reservations
	reservations, err := loadReservations()
	if err != nil {
		http.Error(w, "Failed to load reservations", http.StatusInternalServerError)
		return
	}

	// Find and update the reservation
	found := false
	for i, res := range reservations {
		if res.ID == reservationID {
			updatedReservation.ID = reservationID // Ensure ID remains unchanged
			reservations[i] = updatedReservation
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Reservation not found", http.StatusNotFound)
		return
	}

	// Save updated reservations
	if err := saveReservations(reservations); err != nil {
		http.Error(w, "Failed to save reservation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedReservation)
}

func handleDeleteReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reservationID := vars["id"]

	// Load existing reservations
	reservations, err := loadReservations()
	if err != nil {
		http.Error(w, "Failed to load reservations", http.StatusInternalServerError)
		return
	}

	// Find and remove the reservation
	found := false
	var updatedReservations []Reservation
	for _, res := range reservations {
		if res.ID == reservationID {
			found = true
			continue
		}
		updatedReservations = append(updatedReservations, res)
	}

	if !found {
		http.Error(w, "Reservation not found", http.StatusNotFound)
		return
	}

	// Save updated reservations
	if err := saveReservations(updatedReservations); err != nil {
		http.Error(w, "Failed to save reservations", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
