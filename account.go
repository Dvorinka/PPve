package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Session struct {
	Token     string
	Username  string
	Role      string
	ExpiresAt time.Time
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	Role    string `json:"role,omitempty"`
}

// In-memory storage (replace with database in production)
var (
	users = map[string]User{
		"admin": {
			Username: "admin",
			Password: "admin123", // In production, use hashed passwords
			Role:     "admin",
		},
	}
	sessions = make(map[string]Session)
)

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		// Serve login page
		tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Přihlášení - Správa</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 10px;
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
            width: 100%;
            max-width: 400px;
        }
        
        .login-header {
            text-align: center;
            margin-bottom: 2rem;
        }
        
        .login-header h1 {
            color: #333;
            font-size: 2rem;
            margin-bottom: 0.5rem;
        }
        
        .login-header p {
            color: #666;
            font-size: 0.9rem;
        }
        
        .form-group {
            margin-bottom: 1.5rem;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }
        
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e1e5e9;
            border-radius: 5px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }
        
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .login-button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: transform 0.2s;
        }
        
        .login-button:hover {
            transform: translateY(-2px);
        }
        
        .login-button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        
        .error-message {
            background: #fee;
            color: #c33;
            padding: 0.75rem;
            border-radius: 5px;
            margin-bottom: 1rem;
            display: none;
        }
        
        .loading {
            display: none;
            text-align: center;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1>Přihlášení</h1>
            <p>Administrátorské rozhraní</p>
        </div>
        
        <div class="error-message" id="errorMessage"></div>
        
        <form id="loginForm">
            <div class="form-group">
                <label for="username">Uživatelské jméno</label>
                <input type="text" id="username" name="username" required>
            </div>
            
            <div class="form-group">
                <label for="password">Heslo</label>
                <input type="password" id="password" name="password" required>
            </div>
            
            <button type="submit" class="login-button" id="loginButton">
                Přihlásit se
            </button>
        </form>
        
        <div class="loading" id="loading">
            Přihlašování...
        </div>
    </div>

    <script>
        document.getElementById('loginForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const errorDiv = document.getElementById('errorMessage');
            const loginButton = document.getElementById('loginButton');
            const loading = document.getElementById('loading');
            
            // Reset error
            errorDiv.style.display = 'none';
            loginButton.disabled = true;
            loading.style.display = 'block';
            
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        username: username,
                        password: password
                    })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    localStorage.setItem('authToken', data.token);
                    localStorage.setItem('userRole', data.role);
                    window.location.href = '/admin';
                } else {
                    errorDiv.textContent = data.message;
                    errorDiv.style.display = 'block';
                }
            } catch (error) {
                errorDiv.textContent = 'Chyba při přihlašování. Zkuste to znovu.';
                errorDiv.style.display = 'block';
            } finally {
                loginButton.disabled = false;
                loading.style.display = 'none';
            }
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(tmpl))
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	// Check credentials
	user, exists := users[loginReq.Username]
	if !exists || user.Password != loginReq.Password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Neplatné přihlašovací údaje",
		})
		return
	}

	// Generate session token
	token, err := generateToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Chyba při vytváření relace",
		})
		return
	}

	// Store session
	sessions[token] = Session{
		Token:     token,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	json.NewEncoder(w).Encode(LoginResponse{
		Success: true,
		Message: "Přihlášení úspěšné",
		Token:   token,
		Role:    user.Role,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token := r.Header.Get("Authorization")
	if token == "" {
		// Try to get from cookie
		if cookie, err := r.Cookie("authToken"); err == nil {
			token = cookie.Value
		}
	}

	if token != "" {
		delete(sessions, token)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Odhlášení úspěšné",
	})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Try to get from cookie
			if cookie, err := r.Cookie("authToken"); err == nil {
				token = cookie.Value
			}
		}

		if token == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session, exists := sessions[token]
		if !exists || time.Now().After(session.ExpiresAt) {
			if exists {
				delete(sessions, token)
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Extend session
		session.ExpiresAt = time.Now().Add(24 * time.Hour)
		sessions[token] = session

		next(w, r)
	}
}

func requireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Try to get from cookie
			if cookie, err := r.Cookie("authToken"); err == nil {
				token = cookie.Value
			}
		}

		if token == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session, exists := sessions[token]
		if !exists || time.Now().After(session.ExpiresAt) || session.Role != "admin" {
			if exists {
				delete(sessions, token)
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Extend session
		session.ExpiresAt = time.Now().Add(24 * time.Hour)
		sessions[token] = session

		next(w, r)
	}
}

func getCurrentUser(r *http.Request) *Session {
	token := r.Header.Get("Authorization")
	if token == "" {
		if cookie, err := r.Cookie("authToken"); err == nil {
			token = cookie.Value
		}
	}

	if token == "" {
		return nil
	}

	session, exists := sessions[token]
	if !exists || time.Now().After(session.ExpiresAt) {
		return nil
	}

	return &session
}
