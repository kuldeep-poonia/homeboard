package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

type ItemsHandler struct {
	cfg *config.Config
	db  *db.DB
	hub *ws.Hub
}

func NewItemsHandler(cfg *config.Config, database *db.DB, hub *ws.Hub) *ItemsHandler {
	return &ItemsHandler{
		cfg: cfg,
		db:  database,
		hub: hub,
	}
}

// ServeHTTP handles /v1/boards/{id}/items and /v1/boards/{id}/items/{iid}
func (h *ItemsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/boards/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 || parts[1] != "items" {
		middleware.JSONError(w, "not found", http.StatusNotFound)
		return
	}

	boardID := parts[0]

	// Authenticate caller (TV or Phone device)
	authInfo, err := middleware.AuthenticateRequest(h.db, r, boardID)
	if err != nil {
		if errors.Is(err, db.ErrUnauthorized) {
			middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		middleware.JSONError(w, "authentication error", http.StatusInternalServerError)
		return
	}

	// Route based on number of path segments
	if len(parts) == 2 {
		// /v1/boards/{id}/items
		switch r.Method {
		case http.MethodGet:
			h.ListItems(w, r, authInfo)
		case http.MethodPost:
			h.CreateItem(w, r, authInfo)
		default:
			middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 {
		// /v1/boards/{id}/items/{iid}
		itemID := parts[2]
		switch r.Method {
		case http.MethodPatch:
			h.UpdateItem(w, r, authInfo, itemID)
		case http.MethodDelete:
			h.DeleteItem(w, r, authInfo, itemID)
		default:
			middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	middleware.JSONError(w, "not found", http.StatusNotFound)
}

func (h *ItemsHandler) ListItems(w http.ResponseWriter, r *http.Request, auth *middleware.AuthInfo) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	items, err := h.db.ListItems(auth.BoardID, includeArchived)
	if err != nil {
		middleware.JSONError(w, "failed to list items", http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, items, http.StatusOK)
}

func (h *ItemsHandler) CreateItem(w http.ResponseWriter, r *http.Request, auth *middleware.AuthInfo) {
	// Check max items limit per board
	count, err := h.db.CountActiveItems(auth.BoardID)
	if err != nil {
		middleware.JSONError(w, "database error", http.StatusInternalServerError)
		return
	}
	if count >= h.cfg.MaxItemsPerBoard {
		middleware.JSONError(w, "board limit reached: max items exceeded", http.StatusBadRequest)
		return
	}

	var req models.CreateItemRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		middleware.JSONError(w, "invalid request body: malformed JSON or unknown fields", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		middleware.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var createdBy *string
	if auth.Device != nil {
		createdBy = &auth.Device.ID
	}

	item := &models.Item{
		BoardID:         auth.BoardID,
		Type:            req.Type,
		Text:            req.Text,
		WhenTS:          req.WhenTS,
		Done:            false,
		Archived:        false,
		CreatedByDevice: createdBy,
	}

	if err := h.db.CreateItem(item); err != nil {
		middleware.JSONError(w, "failed to create item", http.StatusInternalServerError)
		return
	}

	// Broadcast creation event via live WebSocket
	h.hub.BroadcastEvent(auth.BoardID, "item.created", item)

	middleware.JSONResponse(w, item, http.StatusCreated)
}

func (h *ItemsHandler) UpdateItem(w http.ResponseWriter, r *http.Request, auth *middleware.AuthInfo, itemID string) {
	var req models.UpdateItemRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		middleware.JSONError(w, "invalid request body: malformed JSON or unknown fields", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		middleware.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	item, err := h.db.UpdateItem(auth.BoardID, itemID, &req)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			middleware.JSONError(w, "item not found", http.StatusNotFound)
			return
		}
		middleware.JSONError(w, "failed to update item", http.StatusInternalServerError)
		return
	}

	// Broadcast update event via live WebSocket
	h.hub.BroadcastEvent(auth.BoardID, "item.updated", item)

	middleware.JSONResponse(w, item, http.StatusOK)
}

func (h *ItemsHandler) DeleteItem(w http.ResponseWriter, r *http.Request, auth *middleware.AuthInfo, itemID string) {
	if err := h.db.DeleteItem(auth.BoardID, itemID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			middleware.JSONError(w, "item not found", http.StatusNotFound)
			return
		}
		middleware.JSONError(w, "failed to delete item", http.StatusInternalServerError)
		return
	}

	// Broadcast deletion event via live WebSocket
	h.hub.BroadcastEvent(auth.BoardID, "item.deleted", map[string]string{"id": itemID})

	w.WriteHeader(http.StatusNoContent)
}
