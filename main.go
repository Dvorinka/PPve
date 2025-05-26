package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

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

type GeoCoords struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Set up reverse proxy to kontakt service
	kontaktURL, _ := url.Parse("http://webportal:8080")
	kontaktProxy := httputil.NewSingleHostReverseProxy(kontaktURL)

	http.Handle("/kontakt/", http.StripPrefix("/kontakt", kontaktProxy))

	http.HandleFunc("/submit", enableCORS(handleSubmit))
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))

	http.HandleFunc("/", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	}))

	http.HandleFunc("/evidence-aut", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "evidence-aut.html")
	}))

	http.HandleFunc("/kontakt", enableCORS(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	// Authentication routes
	http.HandleFunc("/login", enableCORS(handleLogin))
	http.HandleFunc("/logout", enableCORS(handleLogout))

	// Admin routes (protected)
	http.HandleFunc("/admin", enableCORS(requireAdminAuth(handleAdmin)))
	http.HandleFunc("/admin/cards", enableCORS(requireAdminAuth(handleAdminCards)))

	http.HandleFunc("/admin/cards/", enableCORS(requireAdminAuth(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/toggle") {
			handleAdminCardToggle(w, r)
		} else if r.Method == "DELETE" {
			handleAdminCardDelete(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})))

	// Public API to get cards for homepage
	http.HandleFunc("/api/cards", enableCORS(handleGetCards))

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Server běží na portu %s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Chyba při spuštění serveru: %v", err)
	}
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if next != nil {
			next(w, r)
		}
	}
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
