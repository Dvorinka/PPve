package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateCredentialsRequest struct {
	CurrentUsername string `json:"currentUsername"`
	CurrentPassword string `json:"currentPassword"`
	NewUsername     string `json:"newUsername"`
	NewPassword     string `json:"newPassword"`
}

type CredentialsResponse struct {
	IsDefaultCredentials bool `json:"isDefaultCredentials"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var (
	// In production, use environment variable for JWT key
	jwtKey = getJWTKey()

	// Default credentials
	defaultUsername     = "admin"
	defaultPassword     = "admin"
	defaultPasswordHash = mustHashPassword(defaultPassword)

	// Current credentials (in-memory, would be from DB in production)
	adminUsername     = defaultUsername
	adminPasswordHash = defaultPasswordHash

	// Mutex for thread-safe credential updates
	credentialsMutex sync.RWMutex
)

func getJWTKey() []byte {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		return []byte("default-secret-key-change-in-production")
	}
	return []byte(key)
}

func mustHashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}
	return string(hash)
}

func authenticateUser(creds Credentials) (string, bool, error) {
	credentialsMutex.RLock()
	defer credentialsMutex.RUnlock()

	if creds.Username != adminUsername {
		return "", false, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminPasswordHash), []byte(creds.Password)); err != nil {
		return "", false, errors.New("invalid credentials")
	}

	// Check if using default credentials
	isDefault := creds.Username == defaultUsername && bcrypt.CompareHashAndPassword(
		[]byte(defaultPasswordHash), []byte(creds.Password)) == nil

	tokenString, err := createToken(creds.Username)
	return tokenString, isDefault, err
}

func createToken(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func verifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// AuthMiddleware verifies the JWT token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Format: Bearer <token>
		tokenString := ""
		if len(authHeader) > 7 && strings.ToUpper(authHeader[0:7]) == "BEARER " {
			tokenString = authHeader[7:]
		} else {
			http.Error(w, `{"error":"Authorization header format must be Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		_, err := verifyToken(tokenString)
		if err != nil {
			http.Error(w, `{"error":"Invalid or expired token`+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed with the request
		next.ServeHTTP(w, r)
	})
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenString, isDefault, err := authenticateUser(creds)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":               tokenString,
		"isDefaultCredentials": isDefault,
	})
}

// UpdateCredentialsHandler handles credential updates
func UpdateCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var updateCreds UpdateCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&updateCreds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Authenticate current credentials
	if _, _, err := authenticateUser(Credentials{
		Username: updateCreds.CurrentUsername,
		Password: updateCreds.CurrentPassword,
	}); err != nil {
		http.Error(w, "Invalid current credentials", http.StatusUnauthorized)
		return
	}

	// Update credentials
	credentialsMutex.Lock()
	defer credentialsMutex.Unlock()

	adminUsername = updateCreds.NewUsername
	adminPasswordHash = mustHashPassword(updateCreds.NewPassword)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}
