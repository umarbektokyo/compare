package main

import (
	"encoding/json"
	"math"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// --- Helper: extract room ID from URL path segments ---

// parseRoomID extracts the room ID from paths like /api/rooms/{id}/...
func parseRoomID(path string) (int, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return 0, false
	}
	id, err := strconv.Atoi(parts[2])
	return id, err == nil
}

// roomOwner returns the owner_id for the given room, or 0 on error.
func roomOwner(roomID int) int {
	var ownerID int
	db.QueryRow("SELECT owner_id FROM rooms WHERE id = ?", roomID).Scan(&ownerID)
	return ownerID
}

// getH2HCount returns how many prior matches exist between two items.
func getH2HCount(itemA, itemB int) int {
	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM matches
		WHERE (item_a_id = ? AND item_b_id = ?) OR (item_a_id = ? AND item_b_id = ?)
	`, itemA, itemB, itemB, itemA).Scan(&count)
	return count
}

// --- Room Handlers ---

// handleRooms handles listing all rooms (GET) and creating a new room (POST).
//
//	GET  /api/rooms
//	POST /api/rooms  {"name": "...", "description": "..."}
func handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
			SELECT r.id, r.name, r.description, COALESCE(r.image_url, ''), r.created_at, r.owner_id,
				COALESCE(u.username, ''),
				(SELECT COUNT(*) FROM items WHERE room_id = r.id) as item_count
			FROM rooms r
			LEFT JOIN users u ON r.owner_id = u.id
			ORDER BY r.created_at DESC
		`)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Room struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			ImageURL    string `json:"image_url"`
			CreatedAt   string `json:"created_at"`
			OwnerID     int    `json:"owner_id"`
			OwnerName   string `json:"owner_name"`
			ItemCount   int    `json:"item_count"`
		}

		rooms := []Room{}
		for rows.Next() {
			var room Room
			rows.Scan(&room.ID, &room.Name, &room.Description, &room.ImageURL, &room.CreatedAt, &room.OwnerID, &room.OwnerName, &room.ItemCount)
			rooms = append(rooms, room)
		}
		jsonResponse(w, rooms)

	case http.MethodPost:
		uid, _, ok := requireAuth(r)
		if !ok {
			jsonError(w, "login required to create rooms", http.StatusUnauthorized)
			return
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ImageURL    string `json:"image_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}

		res, err := db.Exec("INSERT INTO rooms (name, description, image_url, owner_id) VALUES (?, ?, ?, ?)", body.Name, body.Description, body.ImageURL, uid)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		jsonResponse(w, map[string]int64{"id": id})

	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRoom handles fetching (GET) or deleting (DELETE) a single room.
//
//	GET    /api/rooms/{id}
//	DELETE /api/rooms/{id}   (owner only)
func handleRoom(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	if idx := strings.Index(idStr, "/"); idx != -1 {
		idStr = idStr[:idx]
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid room id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var room struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			ImageURL    string `json:"image_url"`
			CreatedAt   string `json:"created_at"`
			OwnerID     int    `json:"owner_id"`
			OwnerName   string `json:"owner_name"`
		}
		err := db.QueryRow(`
			SELECT r.id, r.name, r.description, COALESCE(r.image_url, ''), r.created_at, r.owner_id, COALESCE(u.username, '')
			FROM rooms r LEFT JOIN users u ON r.owner_id = u.id
			WHERE r.id = ?
		`, id).Scan(&room.ID, &room.Name, &room.Description, &room.ImageURL, &room.CreatedAt, &room.OwnerID, &room.OwnerName)
		if err != nil {
			jsonError(w, "room not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, room)

	case http.MethodDelete:
		uid, _, ok := requireAuth(r)
		if !ok {
			jsonError(w, "login required", http.StatusUnauthorized)
			return
		}
		if roomOwner(id) != uid {
			jsonError(w, "you can only delete your own rooms", http.StatusForbidden)
			return
		}
		db.Exec("DELETE FROM items WHERE room_id = ?", id)
		db.Exec("DELETE FROM matches WHERE room_id = ?", id)
		db.Exec("DELETE FROM rooms WHERE id = ?", id)
		jsonResponse(w, map[string]string{"status": "deleted"})

	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Item Handlers ---

// handleItems handles listing items (GET) or adding an item (POST) for a room.
//
//	GET  /api/rooms/{id}/items
//	POST /api/rooms/{id}/items  {"title": "...", "description": "...", "image_url": "..."}
func handleItems(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parseRoomID(r.URL.Path)
	if !ok {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
			SELECT id, title, description, image_url, elo, COALESCE(rd, 350), matches, wins, created_at
			FROM items WHERE room_id = ? ORDER BY elo DESC
		`, roomID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Item struct {
			ID          int     `json:"id"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			ImageURL    string  `json:"image_url"`
			Elo         float64 `json:"elo"`
			RD          float64 `json:"rd"`
			Matches     int     `json:"matches"`
			Wins        int     `json:"wins"`
			CreatedAt   string  `json:"created_at"`
		}

		items := []Item{}
		for rows.Next() {
			var item Item
			rows.Scan(&item.ID, &item.Title, &item.Description, &item.ImageURL, &item.Elo, &item.RD, &item.Matches, &item.Wins, &item.CreatedAt)
			items = append(items, item)
		}
		jsonResponse(w, items)

	case http.MethodPost:
		uid, _, ok := requireAuth(r)
		if !ok {
			jsonError(w, "login required to add items", http.StatusUnauthorized)
			return
		}
		if roomOwner(roomID) != uid {
			jsonError(w, "you can only add items to your own rooms", http.StatusForbidden)
			return
		}

		var body struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			ImageURL    string `json:"image_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Title) == "" {
			jsonError(w, "title is required", http.StatusBadRequest)
			return
		}

		res, err := db.Exec("INSERT INTO items (room_id, title, description, image_url) VALUES (?, ?, ?, ?)",
			roomID, body.Title, body.Description, body.ImageURL)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		jsonResponse(w, map[string]int64{"id": id})

	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeleteItem removes an item by ID. Only the room owner may delete.
//
//	DELETE /api/items/{id}
func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uid, _, ok := requireAuth(r)
	if !ok {
		jsonError(w, "login required", http.StatusUnauthorized)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/items/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var ownerID int
	err = db.QueryRow("SELECT r.owner_id FROM items i JOIN rooms r ON i.room_id = r.id WHERE i.id = ?", id).Scan(&ownerID)
	if err != nil {
		jsonError(w, "item not found", http.StatusNotFound)
		return
	}
	if ownerID != uid {
		jsonError(w, "you can only delete items from your own rooms", http.StatusForbidden)
		return
	}

	db.Exec("DELETE FROM items WHERE id = ?", id)
	jsonResponse(w, map[string]string{"status": "deleted"})
}

// --- Match / Play Handlers ---

// handleRandomPair returns two randomly selected items for a head-to-head
// comparison. Items with fewer matches are weighted more heavily to ensure
// even coverage.
//
//	GET /api/rooms/{id}/pair
func handleRandomPair(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parseRoomID(r.URL.Path)
	if !ok {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	type Item struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		ImageURL    string  `json:"image_url"`
		Elo         float64 `json:"elo"`
		RD          float64 `json:"rd"`
		Matches     int     `json:"matches"`
		Wins        int     `json:"wins"`
	}

	rows, err := db.Query(
		"SELECT id, title, description, image_url, elo, COALESCE(rd, 350), matches, wins FROM items WHERE room_id = ?",
		roomID,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		rows.Scan(&item.ID, &item.Title, &item.Description, &item.ImageURL, &item.Elo, &item.RD, &item.Matches, &item.Wins)
		items = append(items, item)
	}

	if len(items) < 2 {
		jsonError(w, "need at least 2 items to play", http.StatusBadRequest)
		return
	}

	// Weighted random selection: items with fewer matches get picked more often.
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	weights := make([]float64, len(items))
	totalWeight := 0.0
	for i, item := range items {
		w := 1.0 / (float64(item.Matches) + 1.0)
		weights[i] = w
		totalWeight += w
	}

	pickWeighted := func(exclude int) int {
		target := rng.Float64() * totalWeight
		cumulative := 0.0
		for i := range items {
			if i == exclude {
				continue
			}
			cumulative += weights[i]
			if cumulative >= target {
				return i
			}
		}
		for i := range items {
			if i != exclude {
				return i
			}
		}
		return 0
	}

	idxA := pickWeighted(-1)
	idxB := idxA
	for idxB == idxA {
		idxB = pickWeighted(idxA)
	}

	jsonResponse(w, map[string]Item{
		"item_a": items[idxA],
		"item_b": items[idxB],
	})
}

// handleVote records a vote (winner/loser) and updates both items' ratings
// using the Glicko system.
//
//	POST /api/vote  {"room_id": N, "winner_id": N, "loser_id": N}
func handleVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		RoomID   int `json:"room_id"`
		WinnerID int `json:"winner_id"`
		LoserID  int `json:"loser_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var winnerElo, winnerRD float64
	var winnerMatches, winnerWins int
	err := db.QueryRow("SELECT elo, COALESCE(rd, 350), matches, wins FROM items WHERE id = ?", body.WinnerID).
		Scan(&winnerElo, &winnerRD, &winnerMatches, &winnerWins)
	if err != nil {
		jsonError(w, "winner not found", http.StatusNotFound)
		return
	}

	var loserElo, loserRD float64
	var loserMatches, loserWins int
	err = db.QueryRow("SELECT elo, COALESCE(rd, 350), matches, wins FROM items WHERE id = ?", body.LoserID).
		Scan(&loserElo, &loserRD, &loserMatches, &loserWins)
	if err != nil {
		jsonError(w, "loser not found", http.StatusNotFound)
		return
	}

	h2hCount := getH2HCount(body.WinnerID, body.LoserID)

	winResult := CalculateGlicko(winnerElo, winnerRD, loserElo, loserRD, 1.0, h2hCount)
	loseResult := CalculateGlicko(loserElo, loserRD, winnerElo, winnerRD, 0.0, h2hCount)

	tx, _ := db.Begin()
	tx.Exec("UPDATE items SET elo = ?, rd = ?, matches = matches + 1, wins = wins + 1 WHERE id = ?",
		winResult.NewRating, winResult.NewRD, body.WinnerID)
	tx.Exec("UPDATE items SET elo = ?, rd = ?, matches = matches + 1 WHERE id = ?",
		loseResult.NewRating, loseResult.NewRD, body.LoserID)
	tx.Exec("INSERT INTO matches (room_id, item_a_id, item_b_id, winner_id, elo_change) VALUES (?, ?, ?, ?, ?)",
		body.RoomID, body.WinnerID, body.LoserID, body.WinnerID, winResult.Change)
	tx.Commit()

	jsonResponse(w, map[string]any{
		"winner_new_elo": math.Round(winResult.NewRating*10) / 10,
		"loser_new_elo":  math.Round(loseResult.NewRating*10) / 10,
		"winner_gain":    math.Round(winResult.Change*10) / 10,
		"loser_loss":     math.Round(-loseResult.Change*10) / 10,
		"h2h_count":      h2hCount + 1,
		"winner_rd":      math.Round(winResult.NewRD*10) / 10,
		"loser_rd":       math.Round(loseResult.NewRD*10) / 10,
	})
}

// handleMatchHistory returns the last 50 matches for a room.
//
//	GET /api/rooms/{id}/history
func handleMatchHistory(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parseRoomID(r.URL.Path)
	if !ok {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT m.id, a.title, b.title, w.title, m.elo_change, m.created_at
		FROM matches m
		JOIN items a ON m.item_a_id = a.id
		JOIN items b ON m.item_b_id = b.id
		JOIN items w ON m.winner_id = w.id
		WHERE m.room_id = ?
		ORDER BY m.created_at DESC
		LIMIT 50
	`, roomID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Match struct {
		ID        int     `json:"id"`
		ItemA     string  `json:"item_a"`
		ItemB     string  `json:"item_b"`
		Winner    string  `json:"winner"`
		EloChange float64 `json:"elo_change"`
		CreatedAt string  `json:"created_at"`
	}

	matches := []Match{}
	for rows.Next() {
		var m Match
		rows.Scan(&m.ID, &m.ItemA, &m.ItemB, &m.Winner, &m.EloChange, &m.CreatedAt)
		matches = append(matches, m)
	}
	jsonResponse(w, matches)
}
