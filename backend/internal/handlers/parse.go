package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kuldeep-poonia/homeboard/backend/internal/ai"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
)

type ParseHandler struct {
	db     *db.DB
	parser *ai.Parser
}

func NewParseHandler(database *db.DB, parser *ai.Parser) *ParseHandler {
	return &ParseHandler{
		db:     database,
		parser: parser,
	}
}

type ParseRequest struct {
	Text string `json:"text"`
}

func (h *ParseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/boards/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) != 2 || parts[1] != "parse" {
		middleware.JSONError(w, "not found", http.StatusNotFound)
		return
	}

	boardID := parts[0]

	// Authenticate caller (TV or Phone device)
	_, err := middleware.AuthenticateRequest(h.db, r, boardID)
	if err != nil {
		if errors.Is(err, db.ErrUnauthorized) {
			middleware.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		middleware.JSONError(w, "authentication error", http.StatusInternalServerError)
		return
	}

	var req ParseRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		middleware.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		middleware.JSONError(w, "text cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := h.parser.ParseText(r.Context(), req.Text)
	if err != nil {
		middleware.JSONError(w, "failed to parse input", http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, result, http.StatusOK)
}
