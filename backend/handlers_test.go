package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	os.Setenv("DATA_DIR", dir)
	jwtSecret = []byte("test-secret-key-for-testing-only")

	initDB()
	t.Cleanup(func() {
		db.Close()
		os.Unsetenv("DATA_DIR")
	})
}

// doRequest is a test helper that sends an HTTP request to a handler.
func doRequest(handler http.HandlerFunc, method, path string, body any, token string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// decodeJSON is a test helper that decodes a JSON response body.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode JSON: %v (body: %s)", err, w.Body.String())
	}
	return m
}

func TestRegisterAndLogin(t *testing.T) {
	setupTestDB(t)

	// Register
	w := doRequest(handleRegister, "POST", "/api/auth/register",
		map[string]string{"username": "alice", "password": "test1234"}, "")
	if w.Code != 200 {
		t.Fatalf("register failed: %d %s", w.Code, w.Body.String())
	}
	res := decodeJSON(t, w)
	if res["username"] != "alice" {
		t.Errorf("expected username alice, got %v", res["username"])
	}
	token := res["token"].(string)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Duplicate register
	w = doRequest(handleRegister, "POST", "/api/auth/register",
		map[string]string{"username": "alice", "password": "other"}, "")
	if w.Code != 409 {
		t.Errorf("expected 409 for duplicate, got %d", w.Code)
	}

	// Login
	w = doRequest(handleLogin, "POST", "/api/auth/login",
		map[string]string{"username": "alice", "password": "test1234"}, "")
	if w.Code != 200 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}

	// Wrong password
	w = doRequest(handleLogin, "POST", "/api/auth/login",
		map[string]string{"username": "alice", "password": "wrong"}, "")
	if w.Code != 401 {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name string
		body map[string]string
		code int
	}{
		{"short username", map[string]string{"username": "a", "password": "test"}, 400},
		{"short password", map[string]string{"username": "bob", "password": "ab"}, 400},
		{"empty username", map[string]string{"username": "", "password": "test"}, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(handleRegister, "POST", "/api/auth/register", tt.body, "")
			if w.Code != tt.code {
				t.Errorf("expected %d, got %d: %s", tt.code, w.Code, w.Body.String())
			}
		})
	}
}

func TestMe(t *testing.T) {
	setupTestDB(t)

	// No auth
	w := doRequest(handleMe, "GET", "/api/auth/me", nil, "")
	if w.Code != 401 {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}

	// Register to get a token
	w = doRequest(handleRegister, "POST", "/api/auth/register",
		map[string]string{"username": "charlie", "password": "pass1234"}, "")
	token := decodeJSON(t, w)["token"].(string)

	// With auth
	w = doRequest(handleMe, "GET", "/api/auth/me", nil, token)
	if w.Code != 200 {
		t.Fatalf("me failed: %d %s", w.Code, w.Body.String())
	}
	res := decodeJSON(t, w)
	if res["username"] != "charlie" {
		t.Errorf("expected charlie, got %v", res["username"])
	}
}

func TestRoomsCRUD(t *testing.T) {
	setupTestDB(t)

	// Register a user
	w := doRequest(handleRegister, "POST", "/api/auth/register",
		map[string]string{"username": "dave", "password": "pass1234"}, "")
	token := decodeJSON(t, w)["token"].(string)

	// Create room without auth -> 401
	w = doRequest(handleRooms, "POST", "/api/rooms",
		map[string]string{"name": "Test Room"}, "")
	if w.Code != 401 {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}

	// Create room with auth
	w = doRequest(handleRooms, "POST", "/api/rooms",
		map[string]string{"name": "Test Room", "description": "A test"}, token)
	if w.Code != 200 {
		t.Fatalf("create room failed: %d %s", w.Code, w.Body.String())
	}

	// List rooms
	w = doRequest(handleRooms, "GET", "/api/rooms", nil, "")
	if w.Code != 200 {
		t.Fatalf("list rooms failed: %d", w.Code)
	}
	var rooms []map[string]any
	json.NewDecoder(w.Body).Decode(&rooms)
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0]["name"] != "Test Room" {
		t.Errorf("expected 'Test Room', got %v", rooms[0]["name"])
	}
}

func TestDataDirDefault(t *testing.T) {
	os.Unsetenv("DATA_DIR")
	if d := dataDir(); d != "." {
		t.Errorf("expected '.', got %q", d)
	}
}

func TestDataDirEnv(t *testing.T) {
	os.Setenv("DATA_DIR", "/tmp/test-compare")
	defer os.Unsetenv("DATA_DIR")
	if d := dataDir(); d != "/tmp/test-compare" {
		t.Errorf("expected '/tmp/test-compare', got %q", d)
	}
}

func TestInitSecret(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("DATA_DIR", dir)
	defer os.Unsetenv("DATA_DIR")

	// First call: generates secret
	initSecret()
	first := string(jwtSecret)
	if len(first) == 0 {
		t.Fatal("expected non-empty secret")
	}

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(dir, "jwt_secret"))
	if err != nil {
		t.Fatal("secret file not written")
	}
	if string(data) != first {
		t.Error("secret file contents don't match")
	}

	// Second call: loads from file
	jwtSecret = nil
	initSecret()
	if string(jwtSecret) != first {
		t.Error("secret should be loaded from file on second call")
	}
}
