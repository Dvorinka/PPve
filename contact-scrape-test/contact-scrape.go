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
	Table        int    `json:"table"` // 1 for first table, 2 for second table
}

type ContactData struct {
	Contacts    []Contact `json:"contacts"`
	LastUpdated time.Time `json:"last_updated"`
	FileHash    string    `json:"file_hash"`
}

var (
	currentData *ContactData
	dataFile    = "data/contacts.json"
	xlsxFile    = "contacts.xlsx"
)

func main() {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: Could not create data directory: %v", err)
	}

	// Load existing data or parse from Excel
	loadData()

	// Set up HTTP handlers
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/contacts", serveContacts)
	http.HandleFunc("/reload", reloadData)

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
	// Check if Excel file exists
	if _, err := os.Stat(xlsxFile); os.IsNotExist(err) {
		log.Printf("Excel file %s not found, using empty data", xlsxFile)
		currentData = &ContactData{
			Contacts:    []Contact{},
			LastUpdated: time.Now(),
			FileHash:    "",
		}
		return
	}

	// Calculate current file hash
	currentHash, err := calculateFileHash(xlsxFile)
	if err != nil {
		log.Printf("Error calculating file hash: %v", err)
		return
	}

	// Check if cached data exists and is up to date
	if cachedData, err := loadCachedData(); err == nil {
		if cachedData.FileHash == currentHash {
			log.Println("Using cached data (file unchanged)")
			currentData = cachedData
			return
		}
	}

	// Parse Excel file
	log.Println("Parsing Excel file...")
	contacts, err := parseExcelFile(xlsxFile)
	if err != nil {
		log.Printf("Error parsing Excel file: %v", err)
		// Use empty data if parsing fails
		currentData = &ContactData{
			Contacts:    []Contact{},
			LastUpdated: time.Now(),
			FileHash:    currentHash,
		}
		return
	}

	currentData = &ContactData{
		Contacts:    contacts,
		LastUpdated: time.Now(),
		FileHash:    currentHash,
	}

	// Save to cache
	if err := saveCachedData(currentData); err != nil {
		log.Printf("Warning: Could not save cached data: %v", err)
	}

	log.Printf("Loaded %d contacts from Excel file", len(contacts))
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
	var contacts []Contact

	// Parse first table (A-D columns)
	contacts = append(contacts, parseTable(f, sheetName, "A", "D", 1)...)

	// Parse second table (F-H columns)
	contacts = append(contacts, parseTable(f, sheetName, "F", "H", 2)...)

	return contacts, nil
}

func parseTable(f *excelize.File, sheetName, startCol, endCol string, tableNum int) []Contact {
	var contacts []Contact
	var currentContact *Contact

	// Get all rows in the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("Error getting rows: %v", err)
		return contacts
	}

	// Skip header rows (first 3 rows based on your description)
	startRow := 3
	if len(rows) <= startRow {
		return contacts
	}

	// Column indices
	var nameCol, positionCol, phoneCol, servicePhoneCol int
	if tableNum == 1 {
		nameCol, positionCol, phoneCol, servicePhoneCol = 0, 1, 2, 3 // A, B, C, D
	} else {
		nameCol, positionCol, phoneCol = 5, 6, 7 // F, G, H
	}

	for i := startRow; i < len(rows); i++ {
		row := rows[i]

		// Skip if row is too short
		if len(row) <= nameCol {
			continue
		}

		// Check for "Aktualizace" - end of data
		if len(row) > nameCol && strings.Contains(strings.ToLower(row[nameCol]), "aktualizace") {
			break
		}

		// Check for special formatting rows (like "*02(xx)")
		if len(row) > positionCol && strings.Contains(row[positionCol], "*") {
			continue
		}

		name := strings.TrimSpace(row[nameCol])
		position := ""
		phone := ""
		servicePhone := ""

		if len(row) > positionCol {
			position = strings.TrimSpace(row[positionCol])
		}
		if len(row) > phoneCol {
			phone = strings.TrimSpace(row[phoneCol])
		}
		if tableNum == 1 && len(row) > servicePhoneCol {
			servicePhone = strings.TrimSpace(row[servicePhoneCol])
		}

		// Clean phone numbers
		phone = cleanPhoneNumber(phone)
		servicePhone = cleanPhoneNumber(servicePhone)

		// If we have a name, start a new contact
		if name != "" && !strings.Contains(name, "(") {
			currentContact = &Contact{
				Name:         name,
				Position:     position,
				Phone:        phone,
				ServicePhone: servicePhone,
				Table:        tableNum,
			}
			contacts = append(contacts, *currentContact)
		} else if currentContact != nil {
			// This is additional data for the current contact
			newContact := *currentContact
			if position != "" {
				newContact.Position = position
			}
			if phone != "" {
				newContact.Phone = phone
			}
			if servicePhone != "" {
				newContact.ServicePhone = servicePhone
			}
			contacts = append(contacts, newContact)
		}
	}

	return contacts
}

func cleanPhoneNumber(phone string) string {
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

func reloadData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Manual reload requested")
	loadData()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "reloaded", "contacts_count": %d}`, len(currentData.Contacts))
}

func getEmbeddedHTML() string {
	return `<!DOCTYPE html>
<html lang="cs">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kontakty</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="bg-white rounded-lg shadow-lg p-6">
            <div class="flex justify-between items-center mb-6">
                <h1 class="text-3xl font-bold text-gray-800">Kontakty</h1>
                <button onclick="reloadContacts()" id="reloadBtn" 
                        class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-lg transition-colors">
                    Obnovit
                </button>
            </div>
            
            <div class="mb-4">
                <input type="text" id="searchInput" placeholder="Hledat kontakt..." 
                       class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                       onkeyup="filterContacts()">
            </div>

            <div id="loading" class="text-center py-8">
                <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
                <p class="mt-2 text-gray-600">Načítání kontaktů...</p>
            </div>

            <div id="contactsList" class="hidden">
                <div id="stats" class="mb-4 text-sm text-gray-600"></div>
                <div id="contacts" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3"></div>
            </div>

            <div id="error" class="hidden text-center py-8 text-red-600"></div>
        </div>
    </div>

    <script>
        let allContacts = [];
        let filteredContacts = [];

        async function loadContacts() {
            try {
                const response = await fetch('/contacts');
                const data = await response.json();
                allContacts = data.contacts || [];
                filteredContacts = [...allContacts];
                
                document.getElementById('loading').classList.add('hidden');
                document.getElementById('contactsList').classList.remove('hidden');
                
                updateStats(data);
                displayContacts(filteredContacts);
            } catch (error) {
                document.getElementById('loading').classList.add('hidden');
                document.getElementById('error').classList.remove('hidden');
                document.getElementById('error').innerHTML = '<p>Chyba při načítání kontaktů: ' + error.message + '</p>';
            }
        }

        function updateStats(data) {
            const lastUpdated = new Date(data.last_updated).toLocaleString('cs-CZ');
            const table1Count = allContacts.filter(c => c.table === 1).length;
            const table2Count = allContacts.filter(c => c.table === 2).length;
            
            document.getElementById('stats').innerHTML = 
                'Celkem: ' + allContacts.length + ' kontaktů ' +
                '(Tabulka 1: ' + table1Count + ', Tabulka 2: ' + table2Count + ') | ' +
                'Aktualizováno: ' + lastUpdated;
        }

        function displayContacts(contacts) {
            const container = document.getElementById('contacts');
            container.innerHTML = '';

            if (contacts.length === 0) {
                container.innerHTML = '<div class="col-span-full text-center py-8 text-gray-500">Žádné kontakty nenalezeny</div>';
                return;
            }

            contacts.forEach(contact => {
                const contactCard = document.createElement('div');
                contactCard.className = 'bg-gray-50 p-4 rounded-lg border border-gray-200 hover:shadow-md transition-shadow';
                
                contactCard.innerHTML = 
                    '<div class="flex items-start justify-between mb-2">' +
                        '<h3 class="font-semibold text-gray-800 text-lg">' + (contact.name || 'Bez jména') + '</h3>' +
                        '<span class="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">T' + contact.table + '</span>' +
                    '</div>' +
                    (contact.position ? '<p class="text-gray-600 mb-3">' + contact.position + '</p>' : '') +
                    '<div class="space-y-1">' +
                        (contact.phone ? '<div class="flex items-center text-sm"><span class="font-medium text-gray-700 w-16">Tel:</span><a href="tel:' + contact.phone + '" class="text-blue-600 hover:underline">' + contact.phone + '</a></div>' : '') +
                        (contact.service_phone ? '<div class="flex items-center text-sm"><span class="font-medium text-gray-700 w-16">Služ:</span><a href="tel:' + contact.service_phone + '" class="text-blue-600 hover:underline">' + contact.service_phone + '</a></div>' : '') +
                    '</div>';
                
                container.appendChild(contactCard);
            });
        }

        function filterContacts() {
            const query = document.getElementById('searchInput').value.toLowerCase();
            filteredContacts = allContacts.filter(contact => 
                (contact.name && contact.name.toLowerCase().includes(query)) ||
                (contact.position && contact.position.toLowerCase().includes(query)) ||
                (contact.phone && contact.phone.includes(query)) ||
                (contact.service_phone && contact.service_phone.includes(query))
            );
            displayContacts(filteredContacts);
        }

        async function reloadContacts() {
            const btn = document.getElementById('reloadBtn');
            btn.disabled = true;
            btn.textContent = 'Načítání...';
            
            try {
                await fetch('/reload', { method: 'POST' });
                await loadContacts();
            } catch (error) {
                alert('Chyba při obnovování: ' + error.message);
            } finally {
                btn.disabled = false;
                btn.textContent = 'Obnovit';
            }
        }

        // Load contacts on page load
        loadContacts();
    </script>
</body>
</html>`
}
