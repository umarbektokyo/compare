// Compare is a head-to-head comparison platform where users create rooms,
// add items, and vote on pairwise matchups. Items are ranked using a
// Glicko-inspired rating system. Authentication is username/password with
// JWT sessions; voting is anonymous.
//
// Run:
//
//	go run . OR ./compare-server
//
// Environment variables:
//
//	PORT      - HTTP port (default "8080")
//	DATA_DIR  - Directory for SQLite DB and JWT secret (default ".")
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Global state initialized at startup.
var (
	db        *sql.DB
	jwtSecret []byte
)

// dataDir returns the directory used for persistent storage (DB, secrets).
// Defaults to "." but can be overridden with the DATA_DIR env var.
func dataDir() string {
	if d := os.Getenv("DATA_DIR"); d != "" {
		return d
	}
	return "."
}

// initSecret sets the JWT signing key. Priority:
//  1. JWT_SECRET env var (recommended for production/Docker)
//  2. Existing key file on disk
//  3. Auto-generate and persist to disk
func initSecret() {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		jwtSecret = []byte(s)
		return
	}

	dir := dataDir()
	os.MkdirAll(dir, 0755)
	secretFile := filepath.Join(dir, "jwt_secret")

	data, err := os.ReadFile(secretFile)
	if err == nil && len(data) > 0 {
		jwtSecret = data
		return
	}

	b := make([]byte, 32)
	rand.Read(b)
	jwtSecret = []byte(hex.EncodeToString(b))
	os.WriteFile(secretFile, jwtSecret, 0600)
}

// initDB opens the SQLite database and creates tables if they don't exist.
func initDB() {
	var err error
	dbPath := filepath.Join(dataDir(), "compare.db")
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS rooms (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id    INTEGER NOT NULL DEFAULT 0,
		name        TEXT NOT NULL,
		description TEXT DEFAULT '',
		image_url   TEXT DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS items (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id     INTEGER NOT NULL,
		title       TEXT NOT NULL,
		description TEXT DEFAULT '',
		image_url   TEXT DEFAULT '',
		elo         REAL DEFAULT 1500,
		rd          REAL DEFAULT 350,
		matches     INTEGER DEFAULT 0,
		wins        INTEGER DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS matches (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id    INTEGER NOT NULL,
		item_a_id  INTEGER NOT NULL,
		item_b_id  INTEGER NOT NULL,
		winner_id  INTEGER NOT NULL,
		elo_change REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err = db.Exec(schema); err != nil {
		log.Fatal(err)
	}

	// Migrations for existing databases
	db.Exec("ALTER TABLE rooms ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE rooms ADD COLUMN image_url TEXT DEFAULT ''")
	db.Exec("ALTER TABLE items ADD COLUMN rd REAL DEFAULT 350")
}

// --- HTTP helpers ---

// jsonResponse writes a JSON-encoded response with 200 OK.
func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes a JSON error response with the given status code.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// cors wraps a handler with permissive CORS headers for development.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// --- Router ---

func main() {
	initSecret()
	initDB()
	defer db.Close()

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("/api/auth/register", cors(handleRegister))
	mux.HandleFunc("/api/auth/login", cors(handleLogin))
	mux.HandleFunc("/api/auth/me", cors(handleMe))

	// Rooms & nested resources
	mux.HandleFunc("/api/rooms", cors(handleRooms))
	mux.HandleFunc("/api/rooms/", cors(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/items"):
			handleItems(w, r)
		case strings.HasSuffix(path, "/pair"):
			handleRandomPair(w, r)
		case strings.HasSuffix(path, "/history"):
			handleMatchHistory(w, r)
		default:
			handleRoom(w, r)
		}
	}))

	// Items & voting
	mux.HandleFunc("/api/items/", cors(handleDeleteItem))
	mux.HandleFunc("/api/vote", cors(handleVote))

	// Static frontend (SPA with fallback to index.html)
	// Check multiple paths: Docker layout and local dev layout.
	frontendDir := "./frontend/build"
	if _, err := os.Stat(frontendDir); err != nil {
		frontendDir = "../frontend/build"
	}
	if _, err := os.Stat(frontendDir); err == nil {
		fs := http.FileServer(http.Dir(frontendDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(frontendDir, r.URL.Path)
			if _, err := os.Stat(path); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Frontend not built. Run: cd frontend && npm run build")
		})
	}

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("Compare server running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
