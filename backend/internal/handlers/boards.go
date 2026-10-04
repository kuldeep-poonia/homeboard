package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kuldeep-poonia/homeboard/backend/internal/auth"
	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
	"github.com/kuldeep-poonia/homeboard/backend/internal/security"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

type BoardHandler struct {
	cfg          *config.Config
	db           *db.DB
	hub          *ws.Hub
	tokenLimiter *security.RateLimiter
}

func NewBoardHandler(cfg *config.Config, database *db.DB, hub *ws.Hub, tokenLimiter *security.RateLimiter) *BoardHandler {
	return &BoardHandler{
		cfg:          cfg,
		db:           database,
		hub:          hub,
		tokenLimiter: tokenLimiter,
	}
}

// CreateBoard handles POST /v1/boards
func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	boardID := uuid.New().String()
	tvSecret, err := auth.GenerateTVSecret()
	if err != nil {
		middleware.JSONError(w, "internal security error generating credentials", http.StatusInternalServerError)
		return
	}

	tvSecretHash := auth.HashSecret(tvSecret)
	board, err := h.db.CreateBoard(boardID, tvSecretHash)
	if err != nil {
		middleware.JSONError(w, "failed to create board", http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, models.CreateBoardResponse{
		BoardID:  board.ID,
		TVSecret: tvSecret,
	}, http.StatusCreated)
}

// CreateJoinToken handles POST /v1/boards/{id}/join-tokens
func (h *BoardHandler) CreateJoinToken(w http.ResponseWriter, r *http.Request, boardID string) {
	if r.Method != http.MethodPost {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TV authentication required
	if err := middleware.RequireTVAuth(h.db, boardID, r); err != nil {
		middleware.JSONError(w, "unauthorized: valid TV credentials required", http.StatusUnauthorized)
		return
	}

	// Rate limit token creation per board
	if !h.tokenLimiter.Allow("board:" + boardID) {
		middleware.JSONError(w, "rate limit exceeded: too many join tokens generated", http.StatusTooManyRequests)
		return
	}

	rawToken, err := auth.GenerateJoinToken()
	if err != nil {
		middleware.JSONError(w, "failed to generate secure token", http.StatusInternalServerError)
		return
	}

	jt, err := h.db.CreateJoinToken(boardID, rawToken, h.cfg.TokenTTL)
	if err != nil {
		middleware.JSONError(w, "failed to store join token", http.StatusInternalServerError)
		return
	}

	joinURL := fmt.Sprintf("%s/j/%s", h.cfg.BaseURL, rawToken)
	middleware.JSONResponse(w, models.CreateJoinTokenResponse{
		Token:     rawToken,
		URL:       joinURL,
		ExpiresAt: jt.ExpiresAt,
	}, http.StatusCreated)
}

// ListDevices handles GET /v1/boards/{id}/devices
func (h *BoardHandler) ListDevices(w http.ResponseWriter, r *http.Request, boardID string) {
	if r.Method != http.MethodGet {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := middleware.RequireTVAuth(h.db, boardID, r); err != nil {
		middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	devices, err := h.db.ListDevices(boardID)
	if err != nil {
		middleware.JSONError(w, "failed to fetch devices", http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, devices, http.StatusOK)
}

// RevokeDevice handles DELETE /v1/boards/{id}/devices/{did}
func (h *BoardHandler) RevokeDevice(w http.ResponseWriter, r *http.Request, boardID, deviceID string) {
	if r.Method != http.MethodDelete {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := middleware.RequireTVAuth(h.db, boardID, r); err != nil {
		middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.db.RevokeDevice(boardID, deviceID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			middleware.JSONError(w, "device not found", http.StatusNotFound)
			return
		}
		middleware.JSONError(w, "failed to revoke device", http.StatusInternalServerError)
		return
	}

	// Disconnect device's live WebSocket connection immediately
	h.hub.DisconnectDevice(boardID, deviceID)
	h.hub.BroadcastEvent(boardID, "device.revoked", map[string]string{"device_id": deviceID})

	w.WriteHeader(http.StatusNoContent)
}

// Helper to route board sub-paths
func (h *BoardHandler) HandleBoardRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/boards")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		if r.Method == http.MethodPost {
			h.CreateBoard(w, r)
			return
		}
		middleware.JSONError(w, "not found", http.StatusNotFound)
		return
	}

	boardID := parts[0]

	if len(parts) == 1 {
		middleware.JSONError(w, "endpoint not found", http.StatusNotFound)
		return
	}

	subResource := parts[1]
	switch subResource {
	case "join-tokens":
		h.CreateJoinToken(w, r, boardID)
	case "devices":
		if len(parts) == 2 {
			h.ListDevices(w, r, boardID)
		} else if len(parts) == 3 && r.Method == http.MethodDelete {
			h.RevokeDevice(w, r, boardID, parts[2])
		} else {
			middleware.JSONError(w, "not found", http.StatusNotFound)
		}
	default:
		middleware.JSONError(w, "not found", http.StatusNotFound)
	}
}
