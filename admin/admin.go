package admin

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

type GridCard struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Link        string `json:"link"`
	Color       string `json:"color"`
	Order       int    `json:"order"`
	Enabled     bool   `json:"enabled"`
}

// In-memory storage for grid cards (replace with database in production)
var gridCards = []GridCard{
	{
		ID:          "evidence-aut",
		Title:       "Evidence aut",
		Description: "Záznam o jízdách služebním autem",
		Icon:        "🚗",
		Link:        "/evidence-aut",
		Color:       "#004990",
		Order:       1,
		Enabled:     true,
	},
	{
		ID:          "kontakt",
		Title:       "Kontakt",
		Description: "Kontaktní formulář",
		Icon:        "📧",
		Link:        "/kontakt",
		Color:       "#0072b0",
		Order:       2,
		Enabled:     true,
	},
}

func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Administrace</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f5f5f5;
            line-height: 1.6;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        
        .header h1 {
            font-size: 1.5rem;
        }
        
        .user-info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        
        .logout-btn {
            background: rgba(255,255,255,0.2);
            color: white;
            border: none;
            padding: 0.5rem 1rem;
            border-radius: 5px;
            cursor: pointer;
            transition: background 0.3s;
        }
        
        .logout-btn:hover {
            background: rgba(255,255,255,0.3);
        }
        
        .container {
            max-width: 1200px;
            margin: 2rem auto;
            padding: 0 2rem;
        }
        
        .section {
            background: white;
            border-radius: 10px;
            padding: 2rem;
            margin-bottom: 2rem;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        
        .section h2 {
            color: #333;
            margin-bottom: 1.5rem;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid #667eea;
        }
        
        .cards-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }
        
        .card-item {
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 1rem;
            background: #f9f9f9;
        }
        
        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
        }
        
        .card-title {
            font-weight: 600;
            color: #333;
        }
        
        .card-toggle {
            width: 50px;
            height: 24px;
            background: #ccc;
            border-radius: 12px;
            position: relative;
            cursor: pointer;
            transition: background 0.3s;
        }
        
        .card-toggle.active {
            background: #667eea;
        }
        
        .card-toggle::before {
            content: '';
            position: absolute;
            width: 20px;
            height: 20px;
            border-radius: 50%;
            background: white;
            top: 2px;
            left: 2px;
            transition: transform 0.3s;
        }
        
        .card-toggle.active::before {
            transform: translateX(26px);
        }
        
        .form-group {
            margin-bottom: 1rem;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 500;
            color: #333;
        }
        
        .form-group input,
        .form-group textarea {
            width: 100%;
            padding: 0.5rem;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 0.9rem;
        }
        
        .form-group input:focus,
        .form-group textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 0.5rem 1rem;
            border-radius: 5px;
            cursor: pointer;
            transition: background 0.3s;
            margin-right: 0.5rem;
        }
        
        .btn:hover {
            background: #5a67d8;
        }
        
        .btn-danger {
            background: #e53e3e;
        }
        
        .btn-danger:hover {
            background: #c53030;
        }
        
        .add-card-btn {
            background: #38a169;
            color: white;
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
            margin-bottom: 1rem;
        }
        
        .add-card-btn:hover {
            background: #2f855a;
        }
        
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
        }
        
        .modal-content {
            background: white;
            margin: 5% auto;
            padding: 2rem;
            border-radius: 10px;
            width: 90%;
            max-width: 500px;
        }
        
        .close {
            color: #aaa;
            float: right;
            font-size: 28px;
            font-weight: bold;
            cursor: pointer;
        }
        
        .close:hover {
            color: #000;
        }
        
        .success-message {
            background: #c6f6d5;
            color: #22543d;
            padding: 0.75rem;
            border-radius: 5px;
            margin-bottom: 1rem;
            display: none;
        }
        
        .error-message {
            background: #fed7d7;
            color: #822727;
            padding: 0.75rem;
            border-radius: 5px;
            margin-bottom: 1rem;
            display: none;
        }
        
        @media (max-width: 768px) {
            .header {
                padding: 1rem;
                flex-direction: column;
                gap: 1rem;
            }
            
            .container {
                padding: 0 1rem;
            }
            
            .section {
                padding: 1rem;
            }
            
            .cards-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Administrace</h1>
        <div class="user-info">
            <span>Přihlášen jako: <strong>{{.Username}}</strong></span>
            <button class="logout-btn" onclick="logout()">Odhlásit se</button>
        </div>
    </div>
    
    <div class="container">
        <div class="section">
            <h2>Správa karet hlavní stránky</h2>
            <div class="success-message" id="successMessage"></div>
            <div class="error-message" id="errorMessage"></div>
            
            <button class="add-card-btn" onclick="openAddCardModal()">Přidat novou kartu</button>
            
            <div class="cards-grid" id="cardsGrid">
                <!-- Cards will be loaded here -->
            </div>
        </div>
    </div>
    
    <!-- Add/Edit Card Modal -->
    <div id="cardModal" class="modal">
        <div class="modal-content">
            <span class="close" onclick="closeCardModal()">&times;</span>
            <h3 id="modalTitle">Přidat kartu</h3>
            <form id="cardForm">
                <input type="hidden" id="cardId" name="id">
                
                <div class="form-group">
                    <label for="cardTitle">Název</label>
                    <input type="text" id="cardTitle" name="title" required>
                </div>
                
                <div class="form-group">
                    <label for="cardDescription">Popis</label>
                    <textarea id="cardDescription" name="description" rows="3"></textarea>
                </div>
                
                <div class="form-group">
                    <label for="cardIcon">Ikona (emoji nebo text)</label>
                    <input type="text" id="cardIcon" name="icon">
                </div>
                
                <div class="form-group">
                    <label for="cardLink">Odkaz</label>
                    <input type="text" id="cardLink" name="link" required>
                </div>
                
                <div class="form-group">
                    <label for="cardColor">Barva</label>
                    <input type="color" id="cardColor" name="color" value="#004990">
                </div>
                
                <div class="form-group">
                    <label for="cardOrder">Pořadí</label>
                    <input type="number" id="cardOrder" name="order" min="1" value="1">
                </div>
                
                <button type="submit" class="btn">Uložit</button>
                <button type="button" class="btn" onclick="closeCardModal()">Zrušit</button>
            </form>
        </div>
    </div>

    <script>
        let cards = [];
        
        // Load cards on page load
        document.addEventListener('DOMContentLoaded', function() {
            loadCards();
        });
        
        async function loadCards() {
            try {
                const response = await fetch('/admin/cards', {
                    headers: {
                        'Authorization': localStorage.getItem('authToken') || ''
                    }
                });
                
                if (response.ok) {
                    cards = await response.json();
                    renderCards();
                } else {
                    showError('Chyba při načítání karet');
                }
            } catch (error) {
                showError('Chyba při načítání karet');
            }
        }
        
        function renderCards() {
            const grid = document.getElementById('cardsGrid');
            grid.innerHTML = '';
            
            cards.sort((a, b) => a.order - b.order).forEach(card => {
                const cardElement = document.createElement('div');
                cardElement.className = 'card-item';
                cardElement.innerHTML = ` + "`" + `
                    <div class="card-header">
                        <div class="card-title">${card.icon} ${card.title}</div>
                        <div class="card-toggle ${card.enabled ? 'active' : ''}" 
                             onclick="toggleCard('${card.id}')"></div>
                    </div>
                    <p><strong>Popis:</strong> ${card.description}</p>
                    <p><strong>Odkaz:</strong> ${card.link}</p>
                    <p><strong>Barva:</strong> <span style="background: ${card.color}; padding: 2px 8px; color: white; border-radius: 3px;">${card.color}</span></p>
                    <p><strong>Pořadí:</strong> ${card.order}</p>
                    <div style="margin-top: 1rem;">
                        <button class="btn" onclick="editCard('${card.id}')">Upravit</button>
                        <button class="btn btn-danger" onclick="deleteCard('${card.id}')">Smazat</button>
                    </div>
                ` + "`" + `;
                grid.appendChild(cardElement);
            });
        }
        
        async function toggleCard(cardId) {
            try {
                const response = await fetch(` + "`" + `/admin/cards/${cardId}/toggle` + "`" + `, {
                    method: 'POST',
                    headers: {
                        'Authorization': localStorage.getItem('authToken') || ''
                    }
                });
                
                if (response.ok) {
                    await loadCards();
                    showSuccess('Karta byla aktualizována');
                } else {
                    showError('Chyba při aktualizaci karty');
                }
            } catch (error) {
                showError('Chyba při aktualizaci karty');
            }
        }
        
        function openAddCardModal() {
            document.getElementById('modalTitle').textContent = 'Přidat kartu';
            document.getElementById('cardForm').reset();
            document.getElementById('cardId').value = '';
            document.getElementById('cardModal').style.display = 'block';
        }
        
        function editCard(cardId) {
            const card = cards.find(c => c.id === cardId);
            if (!card) return;
            
            document.getElementById('modalTitle').textContent = 'Upravit kartu';
            document.getElementById('cardId').value = card.id;
            document.getElementById('cardTitle').value = card.title;
            document.getElementById('cardDescription').value = card.description;
            document.getElementById('cardIcon').value = card.icon;
            document.getElementById('cardLink').value = card.link;
            document.getElementById('cardColor').value = card.color;
            document.getElementById('cardOrder').value = card.order;
            document.getElementById('cardModal').style.display = 'block';
        }
        
        function closeCardModal() {
            document.getElementById('cardModal').style.display = 'none';
        }
        
        document.getElementById('cardForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const cardData = {
                id: formData.get('id') || generateId(),
                title: formData.get('title'),
                description: formData.get('description'),
                icon: formData.get('icon'),
                link: formData.get('link'),
                color: formData.get('color'),
                order: parseInt(formData.get('order')),
                enabled: true
            };
            
            try {
                const response = await fetch('/admin/cards', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': localStorage.getItem('authToken') || ''
                    },
                    body: JSON.stringify(cardData)
                });
                
                if (response.ok) {
                    closeCardModal();
                    await loadCards();
                    showSuccess('Karta byla uložena');
                } else {
                    showError('Chyba při ukládání karty');
                }
            } catch (error) {
                showError('Chyba při ukládání karty');
            }
        });
        
        async function deleteCard(cardId) {
            if (!confirm('Opravdu chcete smazat tuto kartu?')) return;
            
            try {
                const response = await fetch(` + "`" + `/admin/cards/${cardId}` + "`" + `, {
                    method: 'DELETE',
                    headers: {
                        'Authorization': localStorage.getItem('authToken') || ''
                    }
                });
                
                if (response.ok) {
                    await loadCards();
                    showSuccess('Karta byla smazána');
                } else {
                    showError('Chyba při mazání karty');
                }
            } catch (error) {
                showError('Chyba při mazání karty');
            }
        }
        
        async function logout() {
            try {
                await fetch('/logout', {
                    method: 'POST',
                    headers: {
                        'Authorization': localStorage.getItem('authToken') || ''
                    }
                });
            } catch (error) {
                // Ignore error
            }
            
            localStorage.removeItem('authToken');
            localStorage.removeItem('userRole');
            window.location.href = '/login';
        }
        
        function generateId() {
            return 'card-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        }
        
        function showSuccess(message) {
            const successDiv = document.getElementById('successMessage');
            successDiv.textContent = message;
            successDiv.style.display = 'block';
            setTimeout(() => {
                successDiv.style.display = 'none';
            }, 3000);
        }
        
        function showError(message) {
            const errorDiv = document.getElementById('errorMessage');
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            setTimeout(() => {
                errorDiv.style.display = 'none';
            }, 5000);
        }
        
        // Close modal when clicking outside
        window.onclick = function(event) {
            const modal = document.getElementById('cardModal');
            if (event.target === modal) {
                closeCardModal();
            }
        }
    </script>
</body>
</html>`

	t, err := template.New("admin").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Role     string
	}{
		Username: user.Username,
		Role:     user.Role,
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func HandleAdminCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(gridCards)

	case "POST":
		var card GridCard
		if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		// Check if updating existing card
		found := false
		for i, existingCard := range gridCards {
			if existingCard.ID == card.ID {
				gridCards[i] = card
				found = true
				break
			}
		}

		if !found {
			gridCards = append(gridCards, card)
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Card saved successfully"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func HandleAdminCardToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract card ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid card ID"})
		return
	}

	cardID := parts[3] // /admin/cards/{id}/toggle

	for i, card := range gridCards {
		if card.ID == cardID {
			gridCards[i].Enabled = !gridCards[i].Enabled
			json.NewEncoder(w).Encode(map[string]string{"message": "Card toggled successfully"})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Card not found"})
}

func HandleAdminCardDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "DELETE" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract card ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid card ID"})
		return
	}

	cardID := parts[3] // /admin/cards/{id}

	for i, card := range gridCards {
		if card.ID == cardID {
			// Remove card from slice
			gridCards = append(gridCards[:i], gridCards[i+1:]...)
			json.NewEncoder(w).Encode(map[string]string{"message": "Card deleted successfully"})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Card not found"})
}

func HandleGetCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Filter only enabled cards and sort by order
	var enabledCards []GridCard
	for _, card := range gridCards {
		if card.Enabled {
			enabledCards = append(enabledCards, card)
		}
	}

	// Sort by order
	for i := 0; i < len(enabledCards)-1; i++ {
		for j := i + 1; j < len(enabledCards); j++ {
			if enabledCards[i].Order > enabledCards[j].Order {
				enabledCards[i], enabledCards[j] = enabledCards[j], enabledCards[i]
			}
		}
	}

	json.NewEncoder(w).Encode(enabledCards)
}
