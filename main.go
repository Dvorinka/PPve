package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type TripEntry struct {
	Name        string `json:"name" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	Date        string `json:"date" binding:"required"`
	Purpose     string `json:"purpose" binding:"required"`
	KmStart     int    `json:"km_start" binding:"required"`
	KmEnd       int    `json:"km_end" binding:"required"`
}

func main() {
	r := gin.Default()

	// Enable CORS for all origins
	r.Use(cors.Default())

	r.POST("/submit", func(c *gin.Context) {
		var entry TripEntry
		if err := c.ShouldBindJSON(&entry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Send email with trip details
		err := sendEmail(entry)
		if err != nil {
			log.Println("Failed to send email:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send email"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Entry submitted and email sent successfully"})
	})

	r.Run(":8080")
}

func sendEmail(entry TripEntry) error {
	smtpHost := "smtp.example.com"
	smtpPort := "587"
	sender := "your@email.com"
	password := "yourpassword"
	recipient := "fleet@company.com"

	auth := smtp.PlainAuth("", sender, password, smtpHost)

	subject := "Nový záznam o jízdě služebním autem"

	body := fmt.Sprintf(`
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
        <p><span class="label">Řidič:</span> %s</p>
        <p><span class="label">Kam:</span> %s</p>
        <p><span class="label">Datum:</span> %s</p>
        <p><span class="label">Účel jízdy:</span> %s</p>
        <p><span class="label">Kilometry na začátku:</span> %d km</p>
        <p><span class="label">Kilometry na konci:</span> %d km</p>
        <p><span class="label">Ujeté kilometry:</span> %d km</p>
      </div>
    </body>
    </html>
    `, entry.Name, entry.Destination, entry.Date, entry.Purpose, entry.KmStart, entry.KmEnd, entry.KmEnd-entry.KmStart)

	msg := []byte(
		"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"To: " + recipient + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" + body + "\r\n",
	)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, []string{recipient}, msg)
}
