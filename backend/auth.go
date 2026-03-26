package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// generateToken creates a signed JWT for the given user.
// Tokens are valid for 30 days.
func generateToken(userID int, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// getUserFromRequest extracts the authenticated user from the Authorization
// header. Returns (userID, username) or (0, "") if the request is not
// authenticated or the token is invalid.
func getUserFromRequest(r *http.Request) (int, string) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, ""
	}

	token, err := jwt.Parse(strings.TrimPrefix(auth, "Bearer "), func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ""
	}

	uid, _ := claims["user_id"].(float64)
	uname, _ := claims["username"].(string)
	return int(uid), uname
}

// requireAuth is a convenience wrapper around getUserFromRequest.
// Returns (userID, username, authenticated).
func requireAuth(r *http.Request) (int, string, bool) {
	uid, uname := getUserFromRequest(r)
	return uid, uname, uid != 0
}

// handleRegister creates a new user account.
//
//	POST /api/auth/register  {"username": "...", "password": "..."}
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	body.Username = strings.TrimSpace(body.Username)
	if at := strings.Index(body.Username, "@"); at > 0 {
		body.Username = body.Username[:at]
	}
	if len(body.Username) < 2 || len(body.Username) > 30 {
		jsonError(w, "username must be 2-30 characters", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 4 {
		jsonError(w, "password must be at least 4 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", body.Username, string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonError(w, "username already taken", http.StatusConflict)
			return
		}
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	token, err := generateToken(int(id), body.Username)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]any{
		"token":    token,
		"user_id":  id,
		"username": body.Username,
	})
}

// handleLogin authenticates an existing user.
//
//	POST /api/auth/login  {"username": "...", "password": "..."}
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var id int
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", strings.TrimSpace(body.Username)).
		Scan(&id, &hash)
	if err != nil {
		jsonError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
		jsonError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := generateToken(id, body.Username)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]any{
		"token":    token,
		"user_id":  id,
		"username": body.Username,
	})
}

// handleMe returns the currently authenticated user's info.
//
//	GET /api/auth/me  (requires Authorization header)
func handleMe(w http.ResponseWriter, r *http.Request) {
	uid, uname, ok := requireAuth(r)
	if !ok {
		jsonError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	jsonResponse(w, map[string]any{
		"user_id":  uid,
		"username": uname,
	})
}
