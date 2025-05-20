package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type TripEntry struct {
	Name        string     `json:"name"`
	Destination string     `json:"destination"`
	Date        string     `json:"date"`
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

	http.HandleFunc("/submit", enableCORS(handleSubmit))
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))

	http.HandleFunc("/", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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

	if entry.Name == "" || entry.Destination == "" || entry.Date == "" || entry.Purpose == "" {
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

	// Formátování data do českého formátu
	parsedDate, err := time.Parse("2006-01-02", entry.Date)
	if err == nil {
		czechMonths := []string{
			"ledna", "února", "března", "dubna", "května", "června",
			"července", "srpna", "září", "října", "listopadu", "prosince",
		}
		monthName := czechMonths[parsedDate.Month()-1]
		entry.Date = fmt.Sprintf("%d. %s %d", parsedDate.Day(), monthName, parsedDate.Year())
	} else {
		log.Printf("Chyba při parsování data: %v", err)
	}

	err = sendEmail(entry)
	if err != nil {
		log.Printf("Chyba při odesílání emailu: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error":"Failed to send email: %v"}`, err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Záznam byl úspěšně uložen a email odeslán"}`))
}

func sendEmail(entry TripEntry) error {
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
    <html>
    <head>
      <style>
        body {
          font-family: Arial, sans-serif;
          background-color: #f9f9f9;
          padding: 20px;
        }
        .container {
          background-color: #ffffff;
          border: 1px solid #ddd;
          border-radius: 8px;
          padding: 20px;
          max-width: 600px;
          margin: auto;
        }
        h2 {
          color: #2c3e50;
          border-bottom: 2px solid #3498db;
          padding-bottom: 10px;
        }
        p {
          font-size: 16px;
          color: #34495e;
          line-height: 1.5;
        }
        .label {
          font-weight: bold;
          color: #2980b9;
        }
      </style>
    </head>
    <body>
      <div class="container">
        <h2>Záznam o jízdě služebním autem</h2>
    `)

	fmt.Fprintf(&htmlContent, `<p><span class="label">Řidič:</span> %s</p>`, entry.Name)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Kam:</span> %s</p>`, entry.Destination)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Datum:</span> %s</p>`, entry.Date)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Účel jízdy:</span> %s</p>`, entry.Purpose)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Kilometry na začátku:</span> %d km</p>`, entry.KmStart)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Kilometry na konci:</span> %d km</p>`, entry.KmEnd)
	fmt.Fprintf(&htmlContent, `<p><span class="label">Ujeté kilometry:</span> %d km</p>`, entry.KmEnd-entry.KmStart)

	if entry.Coordinates != nil {
		fmt.Fprintf(&htmlContent, `<p><span class="label">GPS souřadnice:</span> %s, %s</p>`, entry.Coordinates.Lat, entry.Coordinates.Lng)
		fmt.Fprintf(&htmlContent, `<p><a href="https://mapy.cz/zakladni?x=%s&y=%s&z=15" target="_blank">Zobrazit na mapě</a></p>`, entry.Coordinates.Lng, entry.Coordinates.Lat)
	}

	htmlContent.WriteString(`
      </div>
    </body>
    </html>
    `)

	m.SetBody("text/html", htmlContent.String())

	d := gomail.NewDialer(smtpHost, smtpPort, sender, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return d.DialAndSend(m)
}
