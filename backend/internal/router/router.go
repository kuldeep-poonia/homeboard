package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/kuldeep-poonia/homeboard/backend/internal/ai"
	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/handlers"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/kuldeep-poonia/homeboard/backend/internal/security"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

func NewRouter(cfg *config.Config, database *db.DB, hub *ws.Hub) http.Handler {
	// Initialize rate limiters
	tokenGenLimiter := security.NewRateLimiter(cfg.RateLimitTokenGenPerMin, time.Minute)
	redeemLimiter := security.NewRateLimiter(cfg.RateLimitRedeemPerMin, time.Minute)

	// Initialize handlers
	healthH := handlers.HealthCheckHandler()
	boardH := handlers.NewBoardHandler(cfg, database, hub, tokenGenLimiter)
	itemsH := handlers.NewItemsHandler(cfg, database, hub)
	joinH := handlers.NewJoinHandler(cfg, database, hub, redeemLimiter)
	wsH := handlers.NewWSHandler(database, hub)
	qrH := handlers.NewQRHandler(cfg)
	aiParser := ai.NewParser(cfg)
	parseH := handlers.NewParseHandler(database, aiParser)

	tvH := handlers.NewTVHandler(cfg.BaseURL)

	mux := http.NewServeMux()

	// 1. Health check
	mux.HandleFunc("/healthz", healthH)

	// 2. Fire TV 10-foot ambient display
	mux.Handle("/tv", tvH)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/tv", http.StatusFound)
			return
		}
		middleware.JSONError(w, "not found", http.StatusNotFound)
	})

	// 3. QR Image generation
	mux.Handle("/qr/", qrH)

	// 4. Mobile web onboarding page
	mux.HandleFunc("/j/", joinH.ServePhonePage)

	// 5. Token redemption
	mux.HandleFunc("/v1/join/redeem", joinH.Redeem)

	// 5. REST & WebSocket board routes
	mux.HandleFunc("/v1/boards/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/boards/")
		parts := strings.Split(strings.Trim(path, "/"), "/")

		if len(parts) >= 2 {
			switch parts[1] {
			case "items":
				itemsH.ServeHTTP(w, r)
				return
			case "parse":
				parseH.ServeHTTP(w, r)
				return
			case "ws":
				wsH.ServeHTTP(w, r)
				return
			case "join-tokens", "devices":
				boardH.HandleBoardRoute(w, r)
				return
			}
		}

		// Root /v1/boards
		boardH.HandleBoardRoute(w, r)
	})

	// Wrap entire router with security headers, body limit (64KB), and logging
	handler := middleware.SecurityHeaders(mux)
	handler = middleware.LimitBody(64 * 1024)(handler)
	handler = middleware.RequestLogger(handler)

	return handler
}
