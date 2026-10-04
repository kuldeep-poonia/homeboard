package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/kuldeep-poonia/homeboard/backend/internal/auth"
	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
	"github.com/kuldeep-poonia/homeboard/backend/internal/security"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

type JoinHandler struct {
	cfg           *config.Config
	db            *db.DB
	hub           *ws.Hub
	redeemLimiter *security.RateLimiter
}

func NewJoinHandler(cfg *config.Config, database *db.DB, hub *ws.Hub, redeemLimiter *security.RateLimiter) *JoinHandler {
	return &JoinHandler{
		cfg:           cfg,
		db:            database,
		hub:           hub,
		redeemLimiter: redeemLimiter,
	}
}

// Redeem handles POST /v1/join/redeem
func (h *JoinHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit by client IP
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		clientIP = host
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if !h.redeemLimiter.Allow("ip:" + clientIP) {
		middleware.JSONError(w, "too many redemption attempts: try again later", http.StatusTooManyRequests)
		return
	}

	var req models.RedeemTokenRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		middleware.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		middleware.JSONError(w, "token is required", http.StatusBadRequest)
		return
	}

	rawSessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		middleware.JSONError(w, "failed to generate session credentials", http.StatusInternalServerError)
		return
	}

	deviceID, boardID, err := h.db.RedeemJoinToken(req.Token, rawSessionToken, req.Label)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			middleware.JSONError(w, "invalid join token", http.StatusNotFound)
			return
		}
		if errors.Is(err, db.ErrTokenExpired) || errors.Is(err, db.ErrTokenUsed) {
			middleware.JSONError(w, "token has expired or has already been used", http.StatusGone)
			return
		}
		middleware.JSONError(w, "failed to redeem token", http.StatusInternalServerError)
		return
	}

	// Set secure, HttpOnly session cookie
	isSecure := h.cfg.AppEnv == "production" || r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     "homeboard_session",
		Value:    rawSessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	})

	middleware.JSONResponse(w, map[string]interface{}{
		"device_id":     deviceID,
		"board_id":      boardID,
		"session_token": rawSessionToken,
	}, http.StatusOK)
}

// ServePhonePage handles GET /j/{token}
func (h *JoinHandler) ServePhonePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/j/")
	token = strings.TrimSpace(token)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(phoneWebHTML(token)))
}

func phoneWebHTML(initialToken string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>HomeBoard Mobile</title>
  <style>
    :root {
      --bg: #0d1117;
      --card-bg: #161b22;
      --border: #30363d;
      --text: #f0f6fc;
      --text-muted: #8b949e;
      --primary: #58a6ff;
      --accent: #238636;
      --accent-hover: #2ea043;
      --danger: #da3633;
      --font-stack: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; font-family: var(--font-stack); }
    body { background-color: var(--bg); color: var(--text); padding: 16px; min-height: 100vh; display: flex; flex-direction: column; }
    header { display: flex; align-items: center; justify-content: space-between; padding-bottom: 16px; border-bottom: 1px solid var(--border); margin-bottom: 20px; }
    .logo { font-size: 1.25rem; font-weight: 700; color: var(--primary); display: flex; align-items: center; gap: 8px; }
    .status-badge { font-size: 0.75rem; padding: 4px 8px; border-radius: 12px; background: rgba(88, 166, 255, 0.15); color: var(--primary); }
    .status-badge.live { background: rgba(35, 134, 54, 0.2); color: #3fb950; }
    .status-badge.err { background: rgba(218, 54, 51, 0.2); color: var(--danger); }
    
    .panel { background: var(--card-bg); border: 1px solid var(--border); border-radius: 12px; padding: 16px; margin-bottom: 20px; box-shadow: 0 4px 12px rgba(0,0,0,0.2); }
    .type-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 6px; margin-bottom: 14px; }
    .type-btn { background: #21262d; border: 1px solid var(--border); color: var(--text-muted); padding: 8px 2px; border-radius: 8px; font-size: 0.75rem; font-weight: 600; cursor: pointer; text-align: center; transition: all 0.2s; }
    .type-btn.active { background: #1f6feb; color: #fff; border-color: #58a6ff; }
    .type-btn.ai { border-color: #d29922; color: #e3b341; }
    .type-btn.ai.active { background: #9e6a03; color: #fff; border-color: #d29922; }
    
    .input-group { display: flex; flex-direction: column; gap: 10px; }
    input[type="text"] { width: 100%; background: #0d1117; border: 1px solid var(--border); border-radius: 8px; padding: 12px 14px; color: var(--text); font-size: 1rem; outline: none; transition: border-color 0.2s; }
    input[type="text"]:focus { border-color: var(--primary); }
    
    .btn-send { background: var(--accent); color: white; border: none; border-radius: 8px; padding: 12px; font-size: 1rem; font-weight: 600; cursor: pointer; transition: background 0.2s; display: flex; align-items: center; justify-content: center; gap: 6px; }
    .btn-send:hover { background: var(--accent-hover); }
    .btn-send:disabled { opacity: 0.5; cursor: not-allowed; }

    .items-title { font-size: 1rem; font-weight: 600; margin-bottom: 12px; color: var(--text-muted); display: flex; justify-content: space-between; align-items: center; }
    .item-list { display: flex; flex-direction: column; gap: 8px; }
    .item-card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px; padding: 12px 14px; display: flex; align-items: center; justify-content: space-between; gap: 12px; }
    .item-card.done { opacity: 0.5; text-decoration: line-through; }
    .item-left { display: flex; align-items: center; gap: 12px; flex: 1; }
    .item-type { font-size: 0.65rem; text-transform: uppercase; font-weight: 700; padding: 2px 6px; border-radius: 4px; background: #21262d; color: var(--text-muted); }
    .item-text { font-size: 0.95rem; word-break: break-word; }
    .btn-action { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 4px 8px; font-size: 1rem; }
    .btn-action:hover { color: var(--danger); }
    .empty-state { text-align: center; color: var(--text-muted); padding: 32px 16px; font-size: 0.9rem; }
    .msg-banner { padding: 12px; border-radius: 8px; margin-bottom: 16px; font-size: 0.85rem; display: none; }
    .msg-banner.error { background: rgba(218,54,51,0.15); border: 1px solid var(--danger); color: #f85149; display: block; }
    .msg-banner.info { background: rgba(88,166,255,0.15); border: 1px solid var(--primary); color: #58a6ff; display: block; }
  </style>
</head>
<body>
  <header>
    <div class="logo">📺 HomeBoard</div>
    <div id="connStatus" class="status-badge">Connecting...</div>
  </header>

  <div id="msgBanner" class="msg-banner"></div>

  <section class="panel" id="addPanel">
    <div class="type-grid">
      <button type="button" class="type-btn active" data-type="reminder">Reminder</button>
      <button type="button" class="type-btn" data-type="shopping">Shopping</button>
      <button type="button" class="type-btn" data-type="event">Event</button>
      <button type="button" class="type-btn" data-type="movie">Movie</button>
      <button type="button" class="type-btn ai" data-type="auto">✨ AI Auto</button>
    </div>
    <form id="itemForm" class="input-group">
      <input type="text" id="itemTextInput" placeholder="Add note or type naturally..." maxlength="200" autocomplete="off" required>
      <button type="submit" id="btnSubmit" class="btn-send">
        <span>Post to TV</span>
      </button>
    </form>
  </section>

  <div class="items-title">
    <span>Household Items</span>
    <span id="itemCount" style="font-size:0.8rem">0</span>
  </div>
  <div id="itemList" class="item-list">
    <div class="empty-state">No items yet. Add something above!</div>
  </div>

  <script>
    let currentBoardId = null;
    let selectedType = "reminder";
    let ws = null;
    const initialToken = "` + initialToken + `";

    const banner = document.getElementById("msgBanner");
    const connStatus = document.getElementById("connStatus");
    const itemForm = document.getElementById("itemForm");
    const itemInput = document.getElementById("itemTextInput");
    const btnSubmit = document.getElementById("btnSubmit");
    const itemList = document.getElementById("itemList");
    const itemCount = document.getElementById("itemCount");

    function showError(msg) {
      banner.textContent = msg;
      banner.className = "msg-banner error";
    }

    function hideError() {
      banner.textContent = "";
      banner.className = "msg-banner";
    }

    // Type selector buttons
    document.querySelectorAll(".type-btn").forEach(btn => {
      btn.addEventListener("click", () => {
        document.querySelectorAll(".type-btn").forEach(b => b.classList.remove("active"));
        btn.classList.add("active");
        selectedType = btn.getAttribute("data-type");
        itemInput.focus();
      });
    });

    async function initSession() {
      if (initialToken && initialToken !== "") {
        try {
          connStatus.textContent = "Pairing...";
          const res = await fetch("/v1/join/redeem", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ token: initialToken, label: navigator.userAgent.slice(0, 30) })
          });
          if (!res.ok) {
            if (res.status === 410) {
              showError("This QR token has expired or was already used. Please scan the current QR code on your TV.");
            } else {
              showError("Unable to join HomeBoard session. Please rescan the TV QR code.");
            }
            connStatus.textContent = "Disconnected";
            connStatus.className = "status-badge err";
            return;
          }
          const data = await res.json();
          currentBoardId = data.board_id;
          // Clean token from visible URL and history immediately (per security requirements)
          window.history.replaceState({}, document.title, window.location.pathname.replace(/\/j\/.*$/, "/"));
          onConnected();
        } catch (e) {
          showError("Network error while redeeming pairing token.");
          connStatus.textContent = "Offline";
          connStatus.className = "status-badge err";
        }
      }
    }

    async function onConnected() {
      connStatus.textContent = "Live Sync";
      connStatus.className = "status-badge live";
      hideError();
      await fetchItems();
      connectWebSocket();
    }

    async function fetchItems() {
      if (!currentBoardId) return;
      try {
        const res = await fetch("/v1/boards/" + currentBoardId + "/items");
        if (res.ok) {
          const items = await res.json();
          renderItems(items);
        }
      } catch (err) {
        console.error("fetch items failed", err);
      }
    }

    function renderItems(items) {
      itemCount.textContent = items.length;
      if (!items || items.length === 0) {
        itemList.innerHTML = "";
        const empty = document.createElement("div");
        empty.className = "empty-state";
        empty.textContent = "No items yet. Add something above!";
        itemList.appendChild(empty);
        return;
      }

      itemList.innerHTML = "";
      items.forEach(item => {
        const card = document.createElement("div");
        card.className = "item-card" + (item.done ? " done" : "");
        
        const left = document.createElement("div");
        left.className = "item-left";

        const chk = document.createElement("input");
        chk.type = "checkbox";
        chk.checked = item.done;
        chk.addEventListener("change", async () => {
          await toggleDone(item.id, chk.checked);
        });

        const badge = document.createElement("span");
        badge.className = "item-type";
        badge.textContent = item.type;

        const txt = document.createElement("span");
        txt.className = "item-text";
        txt.textContent = item.text; // Safe textContent to prevent XSS

        left.appendChild(chk);
        left.appendChild(badge);
        left.appendChild(txt);

        const delBtn = document.createElement("button");
        delBtn.className = "btn-action";
        delBtn.innerHTML = "&times;";
        delBtn.title = "Delete";
        delBtn.addEventListener("click", async () => {
          await deleteItem(item.id);
        });

        card.appendChild(left);
        card.appendChild(delBtn);
        itemList.appendChild(card);
      });
    }

    async function toggleDone(itemId, done) {
      if (!currentBoardId) return;
      try {
        await fetch("/v1/boards/" + currentBoardId + "/items/" + itemId, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ done: done })
        });
      } catch (e) {
        console.error(e);
      }
    }

    async function deleteItem(itemId) {
      if (!currentBoardId) return;
      try {
        await fetch("/v1/boards/" + currentBoardId + "/items/" + itemId, {
          method: "DELETE"
        });
        await fetchItems();
      } catch (e) {
        console.error(e);
      }
    }

    itemForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      const text = itemInput.value.trim();
      if (!text || !currentBoardId) return;

      btnSubmit.disabled = true;
      try {
        let postType = selectedType;
        let postText = text;
        let whenTS = null;

        if (selectedType === "auto") {
          const parseRes = await fetch("/v1/boards/" + currentBoardId + "/parse", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ text: text })
          });
          if (parseRes.ok) {
            const parsed = await parseRes.json();
            postType = parsed.type || "reminder";
            postText = parsed.text || text;
            whenTS = parsed.when_ts;
          }
        }

        const res = await fetch("/v1/boards/" + currentBoardId + "/items", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ type: postType, text: postText, when_ts: whenTS })
        });
        if (res.ok) {
          itemInput.value = "";
          await fetchItems();
        } else {
          const err = await res.json();
          showError(err.error || "Failed to create item");
        }
      } catch (err) {
        showError("Network error submitting item");
      } finally {
        btnSubmit.disabled = false;
      }
    });

    function connectWebSocket() {
      if (!currentBoardId) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = proto + "//" + window.location.host + "/v1/boards/" + currentBoardId + "/ws";

      ws = new WebSocket(wsUrl);
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "item.created" || msg.type === "item.updated" || msg.type === "item.deleted") {
            fetchItems();
          } else if (msg.type === "device.revoked") {
            showError("Your session was disconnected by the TV.");
            connStatus.textContent = "Revoked";
            connStatus.className = "status-badge err";
          }
        } catch (e) {}
      };
      ws.onclose = () => {
        setTimeout(connectWebSocket, 3000);
      };
    }

    initSession();
  </script>
</body>
</html>`
}
