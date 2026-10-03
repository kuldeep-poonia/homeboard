package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

type WSHandler struct {
	db  *db.DB
	hub *ws.Hub
}

func NewWSHandler(database *db.DB, hub *ws.Hub) *WSHandler {
	return &WSHandler{
		db:  database,
		hub: hub,
	}
}

// ServeHTTP handles GET /v1/boards/{id}/ws
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/boards/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) != 2 || parts[1] != "ws" {
		middleware.JSONError(w, "not found", http.StatusNotFound)
		return
	}

	boardID := parts[0]

	// Authenticate WebSocket connection:
	// Support header, query param (?tv_secret= or ?session=), or session cookie
	var isTV bool
	var deviceID string

	// 1. TV Secret via Header or Query
	tvSecret := r.Header.Get("X-TV-Secret")
	if tvSecret == "" {
		tvSecret = r.URL.Query().Get("tv_secret")
	}

	if tvSecret != "" {
		valid, err := h.db.ValidateTVSecret(boardID, tvSecret)
		if err != nil || !valid {
			middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		isTV = true
		deviceID = "tv-client"
	} else {
		// 2. Phone session via Cookie, Header, or Query
		var sessionToken string
		if cookie, err := r.Cookie("homeboard_session"); err == nil && cookie.Value != "" {
			sessionToken = cookie.Value
		} else if hdr := r.Header.Get("X-Session-Token"); hdr != "" {
			sessionToken = hdr
		} else {
			sessionToken = r.URL.Query().Get("session")
		}

		if sessionToken == "" {
			middleware.JSONError(w, "unauthorized: authentication required", http.StatusUnauthorized)
			return
		}

		dev, err := h.db.GetDeviceBySession(sessionToken)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) || errors.Is(err, db.ErrDeviceRevoked) {
				middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			middleware.JSONError(w, "authentication error", http.StatusInternalServerError)
			return
		}

		if dev.BoardID != boardID {
			middleware.JSONError(w, "unauthorized: cross-board access prohibited", http.StatusUnauthorized)
			return
		}

		isTV = false
		deviceID = dev.ID
	}

	// Upgrade and attach to hub
	if err := ws.UpgradeConnection(h.hub, w, r, boardID, deviceID, isTV); err != nil {
		// UpgradeConnection writes error response if header flush failed
		return
	}
}
