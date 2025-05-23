package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type Contact struct {
	Name         string `json:"name"`
	Position     string `json:"position"`
	Phone        string `json:"phone,omitempty"`
	ServicePhone string `json:"service_phone,omitempty"`
	Internal     bool   `json:"internal"`
	PhoneFlap    string `json:"phone_flap,omitempty"`
}

type ContactData struct {
	Contacts         []Contact `json:"contacts"`
	InternalContacts []Contact `json:"internal_contacts"`
	LastUpdated      time.Time `json:"last_updated"`
	FileHash         string    `json:"file_hash"`
}

var (
	currentData *ContactData
	dataFile    = "data/contacts.json"
	xlsxFile    = "TelefonniSeznamWeb.xlsx"
)

func startAutoReload() {
	ticker := time.NewTicker(3 * 24 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("Auto-reloading contact data...")
				loadData()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func main() {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: Could not create data directory: %v", err)
	}

	// Start auto-reload scheduler
	startAutoReload()

	// Load existing data or parse from Excel
	loadData()

	// Set up HTTP handlers
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/contacts", serveContacts)
	http.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Println("Manual reload requested")
		reloadData()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "reloaded", "contacts_count": %d}`,
			len(currentData.Contacts)+len(currentData.InternalContacts))
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Access the application at: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadData() {
	// Check file every 5 minutes
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			checkFileAndReload()
		}
	}()

	// Initial load
	checkFileAndReload()
}

func checkFileAndReload() {
	currentHash, err := calculateFileHash(xlsxFile)
	if err != nil {
		log.Printf("Hash check error: %v", err)
		return
	}

	if currentHash != currentData.FileHash {
		log.Println("Detected file changes - reloading data")
		reloadData()
	}
}

func calculateFileHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func loadCachedData() (*ContactData, error) {
	file, err := os.Open(dataFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data ContactData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

func saveCachedData(data *ContactData) error {
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func parseExcelFile(filename string) ([]Contact, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Get the first sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	sheetName := sheets[0]

	// Parse the single table (columns A-E)
	contacts := parseTable(f, sheetName, "A", "E")
	return contacts, nil
}

func parseTable(f *excelize.File, sheetName, startCol, endCol string) []Contact {
	var contacts []Contact

	// Get all rows in the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("Error getting rows: %v", err)
		return contacts
	}

	// Skip first 3 and last 3 lines
	startRow := 3
	endRow := len(rows) - 3
	if endRow <= startRow {
		return nil
	}

	// Column indices
	const (
		nameCol         = 0
		positionCol     = 1
		phoneCol        = 2
		servicePhoneCol = 3
		flapCol         = 4
	)

	for i := startRow; i < endRow; i++ {
		row := rows[i]

		// Skip if row is too short
		if len(row) <= nameCol {
			continue
		}

		// Check for "Aktualizace" - end of data
		if len(row) > nameCol && strings.Contains(strings.ToLower(row[nameCol]), "aktualizace") {
			break
		}

		contact := Contact{
			Name:         strings.TrimSpace(row[nameCol]),
			Position:     safeGet(row, positionCol, ""),
			Phone:        formatPhoneNumber(safeGet(row, phoneCol, "")),
			ServicePhone: formatPhoneNumber(safeGet(row, servicePhoneCol, "")), // Full mobile number
			PhoneFlap:    formatPhoneFlap(safeGet(row, flapCol, "")),           // Internal extension with *
		}

		contacts = append(contacts, contact)
	}

	return contacts
}

func safeGet(row []string, index int, defaultValue string) string {
	if index < len(row) {
		return row[index]
	}
	return defaultValue
}

func formatPhoneNumber(phone string) string {
	if phone == "" {
		return ""
	}

	// Remove extra whitespace
	phone = strings.TrimSpace(phone)

	// Remove common formatting characters
	re := regexp.MustCompile(`[^\d+\-\s()]`)
	phone = re.ReplaceAllString(phone, "")

	// If it's just a short number (internal extension), keep as is
	if len(phone) <= 3 {
		return phone
	}

	// If it looks like a Czech number without country code, add it
	if regexp.MustCompile(`^[67]\d{8}$`).MatchString(strings.ReplaceAll(phone, " ", "")) {
		return "+420 " + phone
	}

	return phone
}

func formatPhoneFlap(flap string) string {
	flap = strings.TrimSpace(flap)
	if flap == "" {
		return ""
	}
	if !strings.HasPrefix(flap, "*") {
		return "*" + flap
	}
	return flap
}

func processContacts(contacts []Contact) *ContactData {
	var data ContactData
	data.Contacts = []Contact{}
	data.InternalContacts = []Contact{}

	for _, contact := range contacts {
		// Trim whitespace and check for "Interní"
		if strings.TrimSpace(contact.Name) == "Interní" {
			contact.Internal = true
			data.InternalContacts = append(data.InternalContacts, contact)
		} else {
			contact.Internal = false
			data.Contacts = append(data.Contacts, contact)
		}
	}
	data.LastUpdated = time.Now()
	return &data
}

func reloadData() {
	currentHash, err := calculateFileHash(xlsxFile)
	if err != nil {
		log.Printf("Hash check error: %v", err)
		return
	}

	if currentHash != currentData.FileHash {
		log.Println("Detected file changes - reloading data")
		// Check if Excel file exists
		if _, err := os.Stat(xlsxFile); os.IsNotExist(err) {
			log.Printf("Excel file %s not found, using empty data", xlsxFile)
			currentData = &ContactData{
				Contacts:         []Contact{},
				InternalContacts: []Contact{},
				LastUpdated:      time.Now(),
				FileHash:         "",
			}
			return
		}

		// Parse Excel file
		log.Println("Parsing Excel file...")
		contacts, err := parseExcelFile(xlsxFile)
		if err != nil {
			log.Printf("Error parsing Excel file: %v", err)
			// Use empty data if parsing fails
			currentData = &ContactData{
				Contacts:         []Contact{},
				InternalContacts: []Contact{},
				LastUpdated:      time.Now(),
				FileHash:         currentHash,
			}
			return
		}

		currentData = processContacts(contacts)
		currentData.FileHash = currentHash

		// Save to cache
		if err := saveCachedData(currentData); err != nil {
			log.Printf("Warning: Could not save cached data: %v", err)
		}

		log.Printf("Loaded %d contacts from Excel file", len(currentData.Contacts)+len(currentData.InternalContacts))
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Check if index.html exists
	if _, err := os.Stat("index.html"); os.IsNotExist(err) {
		// Serve embedded HTML if file doesn't exist
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, getEmbeddedHTML())
		return
	}

	http.ServeFile(w, r, "index.html")
}

func serveContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if currentData == nil {
		http.Error(w, `{"error": "No data available"}`, http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(currentData)
}

func getEmbeddedHTML() string {
	return `<!DOCTYPE html>
<html lang="cs">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kontakty</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.2/css/all.min.css">
</head>
<body class="bg-gray-100 min-h-screen">
    <header class="bg-gradient-to-r from-blue-600 to-indigo-700 text-white shadow-lg">
        <div class="container mx-auto px-4 py-6">
            <h1 class="text-3xl font-bold">📞 Firemní telefonní seznam</h1>
            <p class="mt-2 text-blue-100">Poppe + Potthoff kontakty</p>
        </div>
    </header>

    <div class="container mx-auto px-4 py-8 max-w-7xl">
        <div class="bg-white rounded-xl shadow p-6 md:p-8">
            <div class="flex justify-between items-center mb-6">
                <button onclick="reloadContacts()" id="reloadBtn" 
                        class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg transition-all duration-200 shadow-md hover:shadow-lg flex items-center gap-2">
                    <i class="fas fa-sync-alt"></i>
                    Obnovit
                </button>
            </div>
            
            <div class="mb-6">
                <div class="relative">
                    <input type="text" id="searchInput" placeholder="Hledat podle jména, pozice nebo telefonu..." 
                           class="w-full px-4 py-3 pl-12 rounded-lg shadow-sm border-gray-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none text-lg"
                           onkeyup="filterContacts()">
                    <i class="fas fa-search absolute left-4 top-1/2 transform -translate-y-1/2 text-gray-400"></i>
                </div>
            </div>

            <div id="loading" class="text-center py-16">
                <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mb-4"></div>
                <p class="text-gray-600 text-lg">Načítání kontaktů...</p>
            </div>

            <div id="contactsList" class="hidden">
                <div id="stats" class="mb-6 p-4 bg-gray-50 rounded-lg border text-sm text-gray-700"></div>
                <div id="contacts" class="grid gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"></div>
            </div>
        </div>
    </div>

    <footer class="bg-gray-800 text-gray-400 py-6 mt-12">
        <div class="container mx-auto px-4 text-center">
            <p> 2025 Poppe + Potthoff</p>
        </div>
    </footer>
</body>
</html>`
}
