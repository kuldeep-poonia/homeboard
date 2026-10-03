package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
	"github.com/skip2/go-qrcode"
)

type QRHandler struct {
	cfg *config.Config
}

func NewQRHandler(cfg *config.Config) *QRHandler {
	return &QRHandler{cfg: cfg}
}

// ServeHTTP handles GET /qr/{token}.png
func (h *QRHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/qr/")
	token = strings.TrimSuffix(token, ".png")
	token = strings.TrimSpace(token)

	if token == "" || len(token) > 64 {
		middleware.JSONError(w, "invalid token parameter", http.StatusBadRequest)
		return
	}

	joinURL := fmt.Sprintf("%s/j/%s", h.cfg.BaseURL, token)

	png, err := qrcode.Encode(joinURL, qrcode.Medium, 256)
	if err != nil {
		middleware.JSONError(w, "failed to generate qr code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}
