package handlers

import (
	"net/http"
)

type TVHandler struct {
	baseURL string
}

func NewTVHandler(baseURL string) *TVHandler {
	return &TVHandler{baseURL: baseURL}
}

func (h *TVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(tvHTML(h.baseURL)))
}

func tvHTML(baseURL string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>HomeBoard — Fire TV 10-Foot Experience</title>
  <style>
    :root {
      --bg: #ffffff;
      --card-bg: #ffffff;
      --border: #e5e7eb;
      --border-subtle: #f3f4f6;
      --text: #111827;
      --text-muted: #6b7280;
      --focus-color: #111827;
      --accent: #2563eb;
      --done-color: #9ca3af;
      --font-stack: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; font-family: var(--font-stack); -webkit-font-smoothing: antialiased; }
    body {
      background-color: var(--bg);
      color: var(--text);
      width: 100vw;
      height: 100vh;
      overflow: hidden;
      display: flex;
      flex-direction: column;
      user-select: none;
    }
    
    /* 10-Foot TV Top Header */
    header {
      height: 76px;
      padding: 0 48px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-bottom: 1px solid var(--border);
      background: #ffffff;
    }
    .header-left { display: flex; align-items: center; gap: 18px; }
    .brand { font-size: 26px; font-weight: 800; color: #111827; letter-spacing: -0.5px; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 999px;
      background: #f3f4f6;
      border: 1px solid var(--border);
      font-size: 12px;
      font-weight: 600;
    }
    .badge-dot { width: 7px; height: 7px; border-radius: 50%; background: #10b981; }
    .room-name { font-size: 14px; color: var(--text-muted); font-weight: 500; }
    .header-right { text-align: right; }
    .clock-time { font-size: 26px; font-weight: 800; line-height: 1; letter-spacing: -0.5px; color: #111827; }
    .clock-date { font-size: 13px; color: var(--text-muted); margin-top: 3px; font-weight: 500; }

    /* 4 Category Cards Area */
    main {
      flex: 1;
      padding: 24px 48px;
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 20px;
      overflow: hidden;
      background: #ffffff;
    }
    .category-card {
      background: #ffffff;
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      display: flex;
      flex-direction: column;
      box-shadow: 0 1px 3px rgba(0,0,0,0.04);
      transition: border-color 0.15s ease;
    }
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-bottom: 14px;
      margin-bottom: 14px;
      border-bottom: 1px solid var(--border-subtle);
    }
    .card-title-group { display: flex; align-items: center; gap: 10px; }
    .card-title { font-size: 20px; font-weight: 700; color: #111827; letter-spacing: -0.3px; }
    .item-counter {
      background: #f3f4f6;
      padding: 2px 8px;
      border-radius: 999px;
      font-size: 12px;
      color: #374151;
      font-weight: 700;
    }
    .category-tag { font-size: 11px; text-transform: uppercase; font-weight: 700; color: #9ca3af; letter-spacing: 0.8px; }

    .card-items {
      flex: 1;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .card-items::-webkit-scrollbar { width: 0; }

    .tv-item {
      background: #fafafa;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px 14px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      transition: all 0.15s ease-out;
      cursor: pointer;
    }
    .tv-item.focused {
      border-color: #111827 !important;
      background: #ffffff !important;
      outline: 2px solid #111827;
      box-shadow: 0 4px 12px rgba(0,0,0,0.06);
      transform: translateY(-1px);
    }
    .tv-item.done {
      opacity: 0.6;
      background: #ffffff;
      border-color: #f3f4f6;
    }
    .tv-item.done .tv-item-text {
      text-decoration: line-through;
      color: var(--done-color);
    }
    .item-left { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }
    .chk-box {
      width: 20px;
      height: 20px;
      border-radius: 5px;
      border: 1.5px solid #d1d5db;
      background: #ffffff;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 12px;
      font-weight: bold;
      color: transparent;
      transition: all 0.15s ease;
      flex-shrink: 0;
    }
    .tv-item.done .chk-box {
      background: #111827;
      border-color: #111827;
      color: #ffffff;
    }
    .tv-item-text {
      font-size: 16px;
      color: #111827;
      font-weight: 500;
      line-height: 1.35;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .item-type-badge {
      font-size: 10px;
      text-transform: uppercase;
      font-weight: 700;
      color: #6b7280;
      background: #f3f4f6;
      padding: 3px 6px;
      border-radius: 4px;
      margin-left: 8px;
      letter-spacing: 0.3px;
      flex-shrink: 0;
    }

    .empty-state {
      flex: 1;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: #9ca3af;
      padding: 24px 0;
    }
    .empty-emoji { font-size: 28px; margin-bottom: 6px; opacity: 0.6; }
    .empty-title { font-size: 14px; font-weight: 600; color: #6b7280; }
    .empty-sub { font-size: 12px; color: #9ca3af; margin-top: 2px; }

    /* Bottom Bar: Remote Hints & Persistent Corner QR */
    footer {
      height: 96px;
      padding: 0 48px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-top: 1px solid var(--border);
      background: #ffffff;
    }
    .remote-hints { display: flex; align-items: center; gap: 12px; }
    .hint-pill {
      display: flex;
      align-items: center;
      gap: 6px;
      background: #f9fafb;
      border: 1px solid var(--border);
      padding: 6px 12px;
      border-radius: 8px;
    }
    .hint-key { font-size: 11px; font-weight: 800; color: #111827; letter-spacing: 0.5px; }
    .hint-action { font-size: 12px; color: var(--text-muted); font-weight: 500; }

    .qr-container {
      display: flex;
      align-items: center;
      background: #ffffff;
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 8px 12px;
      gap: 12px;
    }
    .qr-box {
      width: 66px;
      height: 66px;
      background: #ffffff;
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 3px;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .qr-img { width: 58px; height: 58px; object-fit: contain; }
    .qr-details h4 { font-size: 13px; font-weight: 700; color: #111827; }
    .qr-details p { font-size: 11px; color: var(--text-muted); margin-top: 1px; }
    .qr-timer { font-size: 11px; font-weight: 600; color: #2563eb; margin-top: 2px; }

    /* Ambient Screensaver / Idle Mode */
    #idleOverlay {
      display: none;
      position: fixed;
      top: 0; left: 0; right: 0; bottom: 0;
      background: #ffffff;
      z-index: 9999;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }
    #idleOverlay.active { display: flex; }
    .idle-clock { font-size: 96px; font-weight: 800; letter-spacing: -3px; color: #111827; }
    .idle-date { font-size: 24px; color: var(--text-muted); margin-top: 6px; font-weight: 500; }
    .idle-badge { margin-top: 24px; padding: 6px 18px; border-radius: 999px; background: #f3f4f6; border: 1px solid var(--border); color: #374151; font-size: 14px; font-weight: 600; }
    .idle-hint { margin-top: 40px; color: #9ca3af; font-size: 13px; }
  </style>
</head>
<body>
  <!-- Ambient Idle Screensaver -->
  <div id="idleOverlay">
    <div id="idleDrifter" style="display:flex; flex-direction:column; align-items:center;">
      <div id="idleClock" class="idle-clock">12:00 PM</div>
      <div id="idleDate" class="idle-date">Saturday, Oct 3</div>
      <div id="idleBadge" class="idle-badge">HomeBoard Ambient Mode</div>
      <div class="idle-hint">Press any button on Fire TV remote to wake</div>
    </div>
  </div>

  <!-- TV Top Navigation Header -->
  <header>
    <div class="header-left">
      <div class="brand">📺 HomeBoard</div>
      <div class="badge">
        <div id="statusDot" class="badge-dot"></div>
        <span id="statusText" style="color:#3fb950;">LIVE SYNC</span>
      </div>
      <div class="room-name">Living Room</div>
    </div>
    <div class="header-right">
      <div id="clockTime" class="clock-time">--:--</div>
      <div id="clockDate" class="clock-date">---, --- --</div>
    </div>
  </header>

  <!-- 4 Category 10-Foot Cards -->
  <main>
    <div class="category-card" data-cat="today">
      <div class="card-header">
        <div class="card-title-group">
          <div class="card-title">Today</div>
          <div class="item-counter" id="count-today">0</div>
        </div>
        <div class="category-tag">Active</div>
      </div>
      <div class="card-items" id="items-today">
        <div class="empty-state"><div class="empty-emoji">📋</div><div class="empty-title">No items</div><div class="empty-sub">Scan QR to add</div></div>
      </div>
    </div>

    <div class="category-card" data-cat="upcoming">
      <div class="card-header">
        <div class="card-title-group">
          <div class="card-title">Upcoming</div>
          <div class="item-counter" id="count-upcoming">0</div>
        </div>
        <div class="category-tag">Events</div>
      </div>
      <div class="card-items" id="items-upcoming">
        <div class="empty-state"><div class="empty-emoji">📅</div><div class="empty-title">No items</div><div class="empty-sub">Scan QR to add</div></div>
      </div>
    </div>

    <div class="category-card" data-cat="buy">
      <div class="card-header">
        <div class="card-title-group">
          <div class="card-title">Buy</div>
          <div class="item-counter" id="count-buy">0</div>
        </div>
        <div class="category-tag">List</div>
      </div>
      <div class="card-items" id="items-buy">
        <div class="empty-state"><div class="empty-emoji">🛒</div><div class="empty-title">No items</div><div class="empty-sub">Scan QR to add</div></div>
      </div>
    </div>

    <div class="category-card" data-cat="family">
      <div class="card-header">
        <div class="card-title-group">
          <div class="card-title">Family & Movies</div>
          <div class="item-counter" id="count-family">0</div>
        </div>
        <div class="category-tag">Watch</div>
      </div>
      <div class="card-items" id="items-family">
        <div class="empty-state"><div class="empty-emoji">🎬</div><div class="empty-title">No items</div><div class="empty-sub">Scan QR to add</div></div>
      </div>
    </div>
  </main>

  <!-- Bottom Bar: Remote Control Key Hints & Dynamic QR -->
  <footer>
    <div class="remote-hints">
      <div class="hint-pill">
        <span class="hint-key">[ D-PAD ]</span>
        <span class="hint-action">Navigate Items</span>
      </div>
      <div class="hint-pill">
        <span class="hint-key">[ SELECT / OK ]</span>
        <span class="hint-action">Mark Done</span>
      </div>
      <div class="hint-pill">
        <span class="hint-key">[ HOLD OK / 'A' ]</span>
        <span class="hint-action">Archive</span>
      </div>
    </div>

    <div class="qr-container">
      <div class="qr-box">
        <img id="qrImage" class="qr-img" src="" alt="Pairing QR">
      </div>
      <div class="qr-details">
        <h4>Instant Phone Sync</h4>
        <p>Scan with mobile camera to add</p>
        <div id="qrTimer" class="qr-timer">Rotating in 8m 00s</div>
      </div>
    </div>
  </footer>

  <script>
    let boardId = localStorage.getItem("homeboard_tv_bid");
    let tvSecret = localStorage.getItem("homeboard_tv_sec");
    let items = [];
    let focusedIndex = 0;
    let focusableItems = [];
    let ws = null;
    let idleTimer = null;
    let qrSecondsLeft = 480;

    const clockTime = document.getElementById("clockTime");
    const clockDate = document.getElementById("clockDate");
    const idleClock = document.getElementById("idleClock");
    const idleDate = document.getElementById("idleDate");
    const idleBadge = document.getElementById("idleBadge");
    const idleOverlay = document.getElementById("idleOverlay");
    const qrImage = document.getElementById("qrImage");
    const qrTimer = document.getElementById("qrTimer");
    const statusDot = document.getElementById("statusDot");
    const statusText = document.getElementById("statusText");

    function updateClock() {
      const now = new Date();
      const t = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      const d = now.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
      clockTime.textContent = t;
      clockDate.textContent = d;
      idleClock.textContent = t;
      idleDate.textContent = now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' });
    }
    setInterval(updateClock, 1000);
    updateClock();

    function resetIdle() {
      if (idleOverlay.classList.contains("active")) {
        idleOverlay.classList.remove("active");
      }
      if (idleTimer) clearTimeout(idleTimer);
      idleTimer = setTimeout(() => {
        const pending = items.filter(it => !it.done && !it.archived).length;
        idleBadge.textContent = pending === 0 ? "All caught up" : pending + " items pending on HomeBoard";
        idleOverlay.classList.add("active");
      }, 180000); // 3 minutes
    }
    window.addEventListener("keydown", resetIdle);
    window.addEventListener("mousemove", resetIdle);
    resetIdle();

    async function initBoard() {
      if (!boardId || !tvSecret) {
        try {
          const res = await fetch("/v1/boards", { method: "POST" });
          const data = await res.json();
          boardId = data.board_id;
          tvSecret = data.tv_secret;
          localStorage.setItem("homeboard_tv_bid", boardId);
          localStorage.setItem("homeboard_tv_sec", tvSecret);
        } catch (e) {
          console.error("Board init error", e);
          return;
        }
      }
      await refreshQRToken();
      await fetchItems();
      connectWebSocket();
    }

    async function refreshQRToken() {
      try {
        const res = await fetch("/v1/boards/" + boardId + "/join-tokens", {
          method: "POST",
          headers: { "X-TV-Secret": tvSecret }
        });
        if (res.ok) {
          const data = await res.json();
          qrImage.src = "/qr/" + data.token + ".png";
          qrSecondsLeft = 480;
        }
      } catch (e) {
        console.error("QR token error", e);
      }
    }
    setInterval(refreshQRToken, 480000); // 8 minutes

    setInterval(() => {
      if (qrSecondsLeft > 0) qrSecondsLeft--;
      const m = Math.floor(qrSecondsLeft / 60);
      const s = qrSecondsLeft % 60;
      qrTimer.textContent = "Rotating in " + m + "m " + (s < 10 ? "0" : "") + s + "s";
    }, 1000);

    async function fetchItems() {
      try {
        const res = await fetch("/v1/boards/" + boardId + "/items", {
          headers: { "X-TV-Secret": tvSecret }
        });
        if (res.ok) {
          items = await res.json();
          renderCards();
        }
      } catch (e) {
        console.error("Fetch items error", e);
      }
    }

    function renderCards() {
      const active = items.filter(it => !it.archived);

      const catMap = {
        today: active.filter(it => it.type === 'reminder' || it.type === 'status'),
        upcoming: active.filter(it => it.type === 'event'),
        buy: active.filter(it => it.type === 'shopping'),
        family: active.filter(it => it.type === 'movie')
      };

      focusableItems = [];

      for (const [cat, catItems] of Object.entries(catMap)) {
        const container = document.getElementById("items-" + cat);
        const counter = document.getElementById("count-" + cat);
        counter.textContent = catItems.length;

        if (catItems.length === 0) {
          container.innerHTML = '<div class="empty-state"><div class="empty-emoji">' +
            (cat === 'today' ? '📋' : cat === 'upcoming' ? '📅' : cat === 'buy' ? '🛒' : '🎬') +
            '</div><div class="empty-title">No items</div><div class="empty-sub">Scan QR to add</div></div>';
        } else {
          container.innerHTML = "";
          catItems.forEach(item => {
            const el = document.createElement("div");
            el.className = "tv-item" + (item.done ? " done" : "");
            el.dataset.id = item.id;
            
            const left = document.createElement("div");
            left.className = "item-left";

            const chk = document.createElement("div");
            chk.className = "chk-box";
            if (item.done) chk.textContent = "✓";

            const txt = document.createElement("div");
            txt.className = "tv-item-text";
            txt.textContent = item.text; // Safe textContent against XSS

            left.appendChild(chk);
            left.appendChild(txt);

            const badge = document.createElement("div");
            badge.className = "item-type-badge";
            badge.textContent = item.type;

            el.appendChild(left);
            el.appendChild(badge);

            el.addEventListener("click", () => toggleDone(item.id, !item.done));
            container.appendChild(el);
            focusableItems.push({ id: item.id, element: el, done: item.done });
          });
        }
      }

      updateFocus();
    }

    function updateFocus() {
      focusableItems.forEach((f, idx) => {
        if (idx === focusedIndex) {
          f.element.classList.add("focused");
          f.element.scrollIntoView({ block: "nearest", behavior: "smooth" });
        } else {
          f.element.classList.remove("focused");
        }
      });
    }

    async function toggleDone(itemId, done) {
      resetIdle();
      try {
        await fetch("/v1/boards/" + boardId + "/items/" + itemId, {
          method: "PATCH",
          headers: {
            "X-TV-Secret": tvSecret,
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ done: done })
        });
        await fetchItems();
      } catch (e) {
        console.error("Toggle done error", e);
      }
    }

    async function archiveItem(itemId) {
      resetIdle();
      try {
        await fetch("/v1/boards/" + boardId + "/items/" + itemId, {
          method: "PATCH",
          headers: {
            "X-TV-Secret": tvSecret,
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ archived: true })
        });
        await fetchItems();
      } catch (e) {
        console.error("Archive error", e);
      }
    }

    // Fire TV Remote & D-Pad Keyboard Controller
    window.addEventListener("keydown", (e) => {
      resetIdle();
      if (focusableItems.length === 0) return;

      if (e.key === "ArrowDown" || e.keyCode === 40) {
        e.preventDefault();
        focusedIndex = (focusedIndex + 1) % focusableItems.length;
        updateFocus();
      } else if (e.key === "ArrowUp" || e.keyCode === 38) {
        e.preventDefault();
        focusedIndex = (focusedIndex - 1 + focusableItems.length) % focusableItems.length;
        updateFocus();
      } else if (e.key === "Enter" || e.keyCode === 13 || e.key === " ") {
        e.preventDefault();
        const current = focusableItems[focusedIndex];
        if (current) toggleDone(current.id, !current.done);
      } else if (e.key === "a" || e.key === "A" || e.key === "Backspace") {
        e.preventDefault();
        const current = focusableItems[focusedIndex];
        if (current) archiveItem(current.id);
      }
    });

    function connectWebSocket() {
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = proto + "//" + window.location.host + "/v1/boards/" + boardId + "/ws?tv_secret=" + encodeURIComponent(tvSecret);

      ws = new WebSocket(wsUrl);
      ws.onopen = () => {
        statusDot.style.background = "#3fb950";
        statusText.textContent = "LIVE SYNC";
        statusText.style.color = "#3fb950";
      };
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "item.created" || msg.type === "item.updated" || msg.type === "item.deleted") {
            fetchItems();
          }
        } catch (e) {}
      };
      ws.onclose = () => {
        statusDot.style.background = "#FF9900";
        statusText.textContent = "RECONNECTING";
        statusText.style.color = "#FF9900";
        setTimeout(connectWebSocket, 3000);
      };
    }

    initBoard();
  </script>
</body>
</html>`
}
