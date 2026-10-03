package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
	"github.com/kuldeep-poonia/homeboard/backend/internal/router"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

func setupTestServer(t *testing.T) (*httptest.Server, *db.DB, func()) {
	t.Helper()

	// Temporary SQLite test database
	tmpFile, err := os.CreateTemp("", "homeboard_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()

	database, err := db.Open(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("failed to open test db: %v", err)
	}

	cfg := &config.Config{
		Port:                    "8080",
		Host:                    "127.0.0.1",
		DBPath:                  dbPath,
		AppEnv:                  "development",
		BaseURL:                 "http://127.0.0.1:8080",
		TokenTTL:                10 * time.Minute,
		MaxItemsPerBoard:        200,
		MaxDevicesPerBoard:      10,
		RateLimitRedeemPerMin:   100,
		RateLimitTokenGenPerMin: 100,
	}

	hub := ws.NewHub(10)
	go hub.Run()

	r := router.NewRouter(cfg, database, hub)
	ts := httptest.NewServer(r)

	cleanup := func() {
		ts.Close()
		database.Close()
		os.Remove(dbPath)
	}

	return ts, database, cleanup
}

// Phase 0: Health Check
func TestPhase0_HealthCheck(t *testing.T) {
	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if data["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", data["status"])
	}
}

// Phase 1: Board creation, validation, CRUD, board isolation
func TestPhase1_BoardAndItems(t *testing.T) {
	ts, database, cleanup := setupTestServer(t)
	defer cleanup()

	// 1.1: Create board
	resp, err := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /v1/boards failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var boardRes models.CreateBoardResponse
	json.NewDecoder(resp.Body).Decode(&boardRes)
	resp.Body.Close()

	if boardRes.BoardID == "" || boardRes.TVSecret == "" {
		t.Fatalf("invalid board creation response: %+v", boardRes)
	}
	boardID := boardRes.BoardID
	tvSecret := boardRes.TVSecret

	// 1.8: Missing TV authentication
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+boardID+"/items", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing TV secret, got %d", resp.StatusCode)
	}

	// 1.9: Invalid TV credential
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+boardID+"/items", nil)
	req.Header.Set("X-TV-Secret", "wrong-secret-1234567890")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong TV secret, got %d", resp.StatusCode)
	}

	// 1.2: Add valid shopping item
	addItemBody := `{"type":"shopping","text":"Organic Milk 2L"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(addItemBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
	var createdItem models.Item
	json.NewDecoder(resp.Body).Decode(&createdItem)
	resp.Body.Close()
	if createdItem.Text != "Organic Milk 2L" || createdItem.Type != "shopping" {
		t.Errorf("unexpected created item: %+v", createdItem)
	}

	// 1.3: Test every supported item type: reminder, shopping, event, movie, status
	types := []string{"reminder", "shopping", "event", "movie", "status"}
	for _, itype := range types {
		b := `{"type":"` + itype + `","text":"Sample ` + itype + `"}`
		req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(b))
		req.Header.Set("X-TV-Secret", tvSecret)
		req.Header.Set("Content-Type", "application/json")
		r, err := http.DefaultClient.Do(req)
		if err != nil || r.StatusCode != http.StatusCreated {
			t.Errorf("failed creating item of type %s: code %d", itype, r.StatusCode)
		}
		r.Body.Close()
	}

	// 1.4: Invalid item type -> 400
	badTypeBody := `{"type":"unsupported_type","text":"test"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(badTypeBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid item type, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.5: Empty or whitespace-only text -> 400
	emptyTextBody := `{"type":"reminder","text":"    "}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(emptyTextBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty text, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.6: Text over 200 characters -> 400
	longText := strings.Repeat("A", 201)
	longBody := `{"type":"reminder","text":"` + longText + `"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(longBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for text > 200 chars, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.7: Hindi + emoji text -> 201 UTF-8 preserved
	unicodeText := "दूध और ब्रेड लाना 🥛🍞"
	unicodeBody := `{"type":"shopping","text":"` + unicodeText + `"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(unicodeBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for UTF-8 text, got %d", resp.StatusCode)
	}
	var uItem models.Item
	json.NewDecoder(resp.Body).Decode(&uItem)
	resp.Body.Close()
	if uItem.Text != unicodeText {
		t.Errorf("expected '%s', got '%s'", unicodeText, uItem.Text)
	}

	// 1.10: Board isolation: Create Board B and verify Board A cannot access Board B
	respB, _ := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	var boardResB models.CreateBoardResponse
	json.NewDecoder(respB.Body).Decode(&boardResB)
	respB.Body.Close()

	// Attempt using Board A secret against Board B
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+boardResB.BoardID+"/items", nil)
	req.Header.Set("X-TV-Secret", tvSecret) // Board A's secret
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Errorf("board isolation violation: expected 401/403, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.11: PATCH done=true
	patchBody := `{"done":true}`
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/v1/boards/"+boardID+"/items/"+createdItem.ID, strings.NewReader(patchBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on PATCH done, got %d", resp.StatusCode)
	}
	var patchedItem models.Item
	json.NewDecoder(resp.Body).Decode(&patchedItem)
	resp.Body.Close()
	if !patchedItem.Done {
		t.Errorf("expected done=true, got false")
	}

	// 1.12: PATCH archive=true
	patchArchiveBody := `{"archived":true}`
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/v1/boards/"+boardID+"/items/"+createdItem.ID, strings.NewReader(patchArchiveBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on PATCH archive, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify archived item is excluded from normal active list
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+boardID+"/items", nil)
	req.Header.Set("X-TV-Secret", tvSecret)
	resp, _ = http.DefaultClient.Do(req)
	var activeItems []models.Item
	json.NewDecoder(resp.Body).Decode(&activeItems)
	resp.Body.Close()
	for _, it := range activeItems {
		if it.ID == createdItem.ID {
			t.Errorf("archived item should not appear in active list")
		}
	}

	// 1.13: Delete item -> 204
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/v1/boards/"+boardID+"/items/"+uItem.ID, nil)
	req.Header.Set("X-TV-Secret", tvSecret)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 No Content on DELETE, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.14: Invalid JSON -> 400
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(`{invalid-json`))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 1.16: SQL Injection test in text field
	sqliText := "'); DROP TABLE items; --"
	sqliBody := `{"type":"reminder","text":"` + sqliText + `"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(sqliBody))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for SQL payload handled as data, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify database is still intact
	_, err = database.CountActiveItems(boardID)
	if err != nil {
		t.Fatalf("database error after SQL injection payload: table corrupted! %v", err)
	}
}

// Phase 2: Pairing, Short-lived Join Tokens, Device Sessions, Revocation
func TestPhase2_PairingAndSessions(t *testing.T) {
	ts, database, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Create Board
	resp, _ := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	var bRes models.CreateBoardResponse
	json.NewDecoder(resp.Body).Decode(&bRes)
	resp.Body.Close()

	// 2.1: Generate Join Token
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+bRes.BoardID+"/join-tokens", nil)
	req.Header.Set("X-TV-Secret", bRes.TVSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed generating join token: code %d, err %v", resp.StatusCode, err)
	}
	var jtRes models.CreateJoinTokenResponse
	json.NewDecoder(resp.Body).Decode(&jtRes)
	resp.Body.Close()

	if jtRes.Token == "" || !strings.Contains(jtRes.URL, "/j/") {
		t.Fatalf("invalid join token response: %+v", jtRes)
	}

	// 2.2: Verify raw token is NOT in database (only SHA-256 hash)
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM join_tokens WHERE token_hash = ?", jtRes.Token).Scan(&count)
	if err != nil || count != 0 {
		t.Errorf("CRITICAL SECURITY FLAW: raw join token found in database!")
	}

	// 2.3: Redeem token once
	redeemBody := `{"token":"` + jtRes.Token + `","label":"Pixel 9 Pro"}`
	resp, err = http.Post(ts.URL+"/v1/join/redeem", "application/json", strings.NewReader(redeemBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to redeem token: code %d, err %v", resp.StatusCode, err)
	}

	// 2.4: Inspect cookie
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "homeboard_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly {
		t.Errorf("expected HttpOnly session cookie, got %+v", sessionCookie)
	}

	var redeemRes map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&redeemRes)
	resp.Body.Close()

	deviceID := redeemRes["device_id"].(string)
	sessionToken := redeemRes["session_token"].(string)

	// 2.6: Redeem the same token again -> 410 Gone (one-time use)
	resp, _ = http.Post(ts.URL+"/v1/join/redeem", "application/json", strings.NewReader(redeemBody))
	if resp.StatusCode != http.StatusGone {
		t.Errorf("expected 410 Gone for reused token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2.8: Random nonexistent token -> 404
	badTokenBody := `{"token":"00000000000000000000000000000000"}`
	resp, _ = http.Post(ts.URL+"/v1/join/redeem", "application/json", strings.NewReader(badTokenBody))
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for random invalid token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2.9: Phone session reads own board
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+bRes.BoardID+"/items", nil)
	req.AddCookie(sessionCookie)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("phone session read items failed: code %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Phone adds item
	phoneItemBody := `{"type":"reminder","text":"Feed the dog"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+bRes.BoardID+"/items", strings.NewReader(phoneItemBody))
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("phone add item failed: code %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2.10: Phone session attempts to access another board
	respOther, _ := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	var otherBoard models.CreateBoardResponse
	json.NewDecoder(respOther.Body).Decode(&otherBoard)
	respOther.Body.Close()

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+otherBoard.BoardID+"/items", nil)
	req.AddCookie(sessionCookie)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("cross-board phone access should be rejected with 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2.11: Phone session attempts TV-only join-token creation -> rejected
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+bRes.BoardID+"/join-tokens", nil)
	req.AddCookie(sessionCookie)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("phone session must not be allowed to create join tokens, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2.12: TV revokes phone device -> next request fails
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/v1/boards/"+bRes.BoardID+"/devices/"+deviceID, nil)
	req.Header.Set("X-TV-Secret", bRes.TVSecret)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("TV revoke device failed: code %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Next phone request should be 401
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+bRes.BoardID+"/items", nil)
	req.AddCookie(sessionCookie)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked device session should be rejected with 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	_ = sessionToken
}

// Phone web HTML & QR image generation
func TestPhase3_PhoneWebAndQR(t *testing.T) {
	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// GET /j/{token}
	resp, err := http.Get(ts.URL + "/j/sampletoken123")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /j/token failed: %v, code: %d", err, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if !strings.Contains(string(body), "HomeBoard Mobile") {
		t.Errorf("expected HTML title 'HomeBoard Mobile'")
	}
	// Verify strict textContent used and no innerHTML
	if strings.Contains(string(body), "innerHTML = item.text") {
		t.Errorf("SECURITY RISK: unsafe innerHTML found in phone web template")
	}

	// GET /qr/{token}.png
	respQR, err := http.Get(ts.URL + "/qr/sampletoken123.png")
	if err != nil || respQR.StatusCode != http.StatusOK {
		t.Fatalf("GET /qr/token.png failed: %v, code: %d", err, respQR.StatusCode)
	}
	if respQR.Header.Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png content type, got %s", respQR.Header.Get("Content-Type"))
	}
	qrBytes, _ := io.ReadAll(respQR.Body)
	respQR.Body.Close()

	// PNG signature check: 89 50 4E 47
	if len(qrBytes) < 8 || !bytes.Equal(qrBytes[0:4], []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Errorf("invalid PNG header in QR response")
	}
}
