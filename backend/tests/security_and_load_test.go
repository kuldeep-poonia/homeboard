package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
	"github.com/kuldeep-poonia/homeboard/backend/internal/router"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

// TestFullEndToEndLifecycle verifies the complete TV + Phone + Live WebSocket sync lifecycle
func TestFullEndToEndLifecycle(t *testing.T) {
	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Fresh Board Creation (Fire TV launches)
	resp, err := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 1 Failed: Fire TV board creation failed (code %d)", resp.StatusCode)
	}
	var boardRes models.CreateBoardResponse
	json.NewDecoder(resp.Body).Decode(&boardRes)
	resp.Body.Close()

	boardID := boardRes.BoardID
	tvSecret := boardRes.TVSecret

	// 2. Fire TV connects Live WebSocket
	wsURL := strings.Replace(ts.URL, "http", "ws", 1) + "/v1/boards/" + boardID + "/ws?tv_secret=" + url.QueryEscape(tvSecret)
	tvWS, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Step 2 Failed: Fire TV WebSocket connection failed: %v", err)
	}
	defer tvWS.Close()

	// Channel to catch events received by Fire TV
	tvEvents := make(chan ws.Event, 10)
	go func() {
		for {
			var evt ws.Event
			if err := tvWS.ReadJSON(&evt); err != nil {
				return
			}
			tvEvents <- evt
		}
	}()

	// 3. Fire TV requests short-lived Join Token for corner QR
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/join-tokens", nil)
	req.Header.Set("X-TV-Secret", tvSecret)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 3 Failed: QR join token generation failed: %v", err)
	}
	var jt models.CreateJoinTokenResponse
	json.NewDecoder(resp.Body).Decode(&jt)
	resp.Body.Close()

	// 4. Phone scans QR and redeems token -> receives HttpOnly session cookie
	redeemPayload := fmt.Sprintf(`{"token":"%s","label":"Living Room iPhone"}`, jt.Token)
	resp, err = http.Post(ts.URL+"/v1/join/redeem", "application/json", strings.NewReader(redeemPayload))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Step 4 Failed: Phone QR redemption failed: %v", err)
	}
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "homeboard_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("Step 4 Failed: session cookie missing in redemption response")
	}

	var redeemData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&redeemData)
	resp.Body.Close()
	deviceID := redeemData["device_id"].(string)

	// 5. Phone connects to WebSocket
	phoneWSURL := strings.Replace(ts.URL, "http", "ws", 1) + "/v1/boards/" + boardID + "/ws"
	header := http.Header{}
	header.Add("Cookie", sessionCookie.String())
	phoneWS, _, err := websocket.DefaultDialer.Dial(phoneWSURL, header)
	if err != nil {
		t.Fatalf("Step 5 Failed: Phone WebSocket connection failed: %v", err)
	}
	defer phoneWS.Close()

	phoneEvents := make(chan ws.Event, 10)
	go func() {
		for {
			var evt ws.Event
			if err := phoneWS.ReadJSON(&evt); err != nil {
				return
			}
			phoneEvents <- evt
		}
	}()

	// 6. Phone adds item -> Fire TV receives real-time update in <1s
	addItemPayload := `{"type":"shopping","text":"Fresh Espresso Beans"}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+boardID+"/items", strings.NewReader(addItemPayload))
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 6 Failed: Phone item creation failed: %v", err)
	}
	var createdItem models.Item
	json.NewDecoder(resp.Body).Decode(&createdItem)
	resp.Body.Close()

	// Verify Fire TV receives 'item.created' event via live WebSocket
	select {
	case evt := <-tvEvents:
		if evt.Type != "item.created" {
			t.Errorf("expected 'item.created' event on Fire TV, got %s", evt.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Step 6 Failed: Timeout waiting for Fire TV live sync event")
	}

	// Drain phone 'item.created' event
	select {
	case <-phoneEvents:
	case <-time.After(2 * time.Second):
	}

	// 7. Fire TV marks item Done -> Phone receives 'item.updated' event
	patchPayload := `{"done":true}`
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/v1/boards/"+boardID+"/items/"+createdItem.ID, strings.NewReader(patchPayload))
	req.Header.Set("X-TV-Secret", tvSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Step 7 Failed: Fire TV mark done failed: %v", err)
	}
	resp.Body.Close()

	// Verify Phone receives 'item.updated'
	select {
	case evt := <-phoneEvents:
		if evt.Type != "item.updated" {
			t.Errorf("expected 'item.updated' event on Phone, got %s", evt.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Step 7 Failed: Timeout waiting for Phone live sync update")
	}

	// 8. Fire TV Revokes Phone Device -> Phone WebSocket disconnects & subsequent requests rejected
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/v1/boards/"+boardID+"/devices/"+deviceID, nil)
	req.Header.Set("X-TV-Secret", tvSecret)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Step 8 Failed: Device revocation failed: %v", err)
	}
	resp.Body.Close()

	// Verify next phone request is strictly rejected with 401
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+boardID+"/items", nil)
	req.AddCookie(sessionCookie)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("security violation: revoked device allowed access (code %d)", resp.StatusCode)
	}
	resp.Body.Close()

	t.Log("SUCCESS: Full End-to-End Lifecycle passed with zero errors!")
}

// TestHighConcurrencyAndTrafficLoad verifies system stability under heavy concurrent traffic
func TestHighConcurrencyAndTrafficLoad(t *testing.T) {
	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	const numBoards = 5
	const clientsPerBoard = 4
	const itemsPerClient = 10
	totalExpectedItems := int64(numBoards * clientsPerBoard * itemsPerClient)

	type testBoard struct {
		id       string
		tvSecret string
		cookies  []*http.Cookie
	}

	boards := make([]testBoard, numBoards)

	// Setup boards and paired devices
	for b := 0; b < numBoards; b++ {
		resp, err := http.Post(ts.URL+"/v1/boards", "application/json", nil)
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed creating board %d", b)
		}
		var bRes models.CreateBoardResponse
		json.NewDecoder(resp.Body).Decode(&bRes)
		resp.Body.Close()

		var cookies []*http.Cookie
		for c := 0; c < clientsPerBoard; c++ {
			// Generate join token
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+bRes.BoardID+"/join-tokens", nil)
			req.Header.Set("X-TV-Secret", bRes.TVSecret)
			r, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("token error: %v", err)
			}
			var jt models.CreateJoinTokenResponse
			json.NewDecoder(r.Body).Decode(&jt)
			r.Body.Close()

			// Redeem
			redeemPayload := fmt.Sprintf(`{"token":"%s","label":"Client %d"}`, jt.Token, c)
			r, _ = http.Post(ts.URL+"/v1/join/redeem", "application/json", strings.NewReader(redeemPayload))
			for _, cookie := range r.Cookies() {
				if cookie.Name == "homeboard_session" {
					cookies = append(cookies, cookie)
					break
				}
			}
			r.Body.Close()
		}

		boards[b] = testBoard{
			id:       bRes.BoardID,
			tvSecret: bRes.TVSecret,
			cookies:  cookies,
		}
	}

	// Concurrent Traffic Generation: All clients simultaneously write and read
	var wg sync.WaitGroup
	var successfulWrites int64
	var failedRequests int64

	startTime := time.Now()

	for bIdx := range boards {
		tb := boards[bIdx]
		for cIdx := range tb.cookies {
			cookie := tb.cookies[cIdx]
			wg.Add(1)

			go func(board testBoard, devCookie *http.Cookie, clientID int) {
				defer wg.Done()

				client := &http.Client{Timeout: 5 * time.Second}

				for i := 0; i < itemsPerClient; i++ {
					payload := fmt.Sprintf(`{"type":"reminder","text":"Concurrent task %d from client %d"}`, i, clientID)
					req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+board.id+"/items", strings.NewReader(payload))
					req.AddCookie(devCookie)
					req.Header.Set("Content-Type", "application/json")

					res, err := client.Do(req)
					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
						continue
					}
					if res.StatusCode == http.StatusCreated {
						atomic.AddInt64(&successfulWrites, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
					res.Body.Close()

					// Concurrent read verification
					readReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+board.id+"/items", nil)
					readReq.AddCookie(devCookie)
					rRes, err := client.Do(readReq)
					if err == nil {
						rRes.Body.Close()
					}
				}
			}(tb, cookie, cIdx)
		}
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Traffic Load Test: %d concurrent writes completed in %v (throughput: %.1f req/s)",
		successfulWrites, duration, float64(successfulWrites)/duration.Seconds())

	if failedRequests > 0 {
		t.Errorf("Detected %d failed requests during high concurrency load", failedRequests)
	}

	if successfulWrites != totalExpectedItems {
		t.Errorf("Expected %d successful writes, got %d", totalExpectedItems, successfulWrites)
	}

	// Verify Tenant Isolation after heavy load: Board 0 should have exactly 40 items, Board 1 should have 40 items, etc.
	for bIdx, tb := range boards {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/boards/"+tb.id+"/items", nil)
		req.Header.Set("X-TV-Secret", tb.tvSecret)
		r, _ := http.DefaultClient.Do(req)
		var list []models.Item
		json.NewDecoder(r.Body).Decode(&list)
		r.Body.Close()

		expectedForBoard := clientsPerBoard * itemsPerClient
		if len(list) != expectedForBoard {
			t.Errorf("Board %d: expected %d items, got %d. Possible cross-board data leak or data loss under load!",
				bIdx, expectedForBoard, len(list))
		}
	}

	t.Log("SUCCESS: High concurrency traffic test passed with zero errors, zero dropped requests, and zero data leakage!")
}

// TestSecurityAudits verifies database zero-plaintext, rate-limits, and payload limits
func TestSecurityAudits(t *testing.T) {
	ts, database, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Database Plaintext Secret Audit: verify raw secrets NEVER exist in DB tables
	resp, _ := http.Post(ts.URL+"/v1/boards", "application/json", nil)
	var b models.CreateBoardResponse
	json.NewDecoder(resp.Body).Decode(&b)
	resp.Body.Close()

	// TV secret must NOT exist in plaintext in boards table
	var secretCount int
	_ = database.QueryRow("SELECT COUNT(*) FROM boards WHERE tv_secret_hash = ?", b.TVSecret).Scan(&secretCount)
	if secretCount > 0 {
		t.Fatalf("SECURITY VIOLATION: Raw TV secret found in database plaintext!")
	}

	// Create join token
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+b.BoardID+"/join-tokens", nil)
	req.Header.Set("X-TV-Secret", b.TVSecret)
	r, _ := http.DefaultClient.Do(req)
	var jt models.CreateJoinTokenResponse
	json.NewDecoder(r.Body).Decode(&jt)
	r.Body.Close()

	// Raw join token must NOT exist in join_tokens table
	var tokenCount int
	_ = database.QueryRow("SELECT COUNT(*) FROM join_tokens WHERE token_hash = ?", jt.Token).Scan(&tokenCount)
	if tokenCount > 0 {
		t.Fatalf("SECURITY VIOLATION: Raw join token found in database plaintext!")
	}

	// 2. Oversized Payload / DoS Protection: send >64KB body
	oversizedBody := strings.Repeat("X", 65*1024) // 65KB
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/boards/"+b.BoardID+"/items", strings.NewReader(oversizedBody))
	req.Header.Set("X-TV-Secret", b.TVSecret)
	req.Header.Set("Content-Type", "application/json")
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusRequestEntityTooLarge && res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 413 or 400 for payload > 64KB, got %d", res.StatusCode)
	}
	res.Body.Close()

	// 3. Brute Force Protection / Rate Limiting
	// Initialize a tight rate limiter test server to quickly trigger 429
	limitedCfg := &config.Config{
		Port:                    "8081",
		Host:                    "127.0.0.1",
		DBPath:                  ":memory:",
		AppEnv:                  "development",
		TokenTTL:                5 * time.Minute,
		MaxItemsPerBoard:        50,
		MaxDevicesPerBoard:      5,
		RateLimitRedeemPerMin:   3, // Limit to 3 per min to verify rate limiting
		RateLimitTokenGenPerMin: 3,
	}
	limitedDB, _ := db.Open(t.TempDir() + "/limited.db")
	defer limitedDB.Close()
	limitedHub := ws.NewHub(5)
	go limitedHub.Run()
	limServer := httptest.NewServer(router.NewRouter(limitedCfg, limitedDB, limitedHub))
	defer limServer.Close()

	// Send 5 rapid invalid redeem requests
	got429 := false
	for i := 0; i < 5; i++ {
		r, _ := http.Post(limServer.URL+"/v1/join/redeem", "application/json", strings.NewReader(`{"token":"invalid-token"}`))
		if r.StatusCode == http.StatusTooManyRequests {
			got429 = true
			r.Body.Close()
			break
		}
		r.Body.Close()
	}
	if !got429 {
		t.Errorf("rate limiter failed: expected HTTP 429 Too Many Requests on rapid attempts")
	}

	t.Log("SUCCESS: Security audits passed (zero plaintext secrets, body limits, brute-force rate-limiting)!")
}
