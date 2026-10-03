package handlers

import (
	"net/http"
	"time"

	"github.com/kuldeep-poonia/homeboard/backend/internal/middleware"
)

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			middleware.JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		middleware.JSONResponse(w, map[string]interface{}{
			"status": "ok",
			"time":   time.Now().UTC(),
		}, http.StatusOK)
	}
}
