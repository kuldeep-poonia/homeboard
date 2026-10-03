package middleware

import (
	"errors"
	"net/http"

	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
)

type contextKey string

const (
	AuthTVKey     contextKey = "auth_tv"
	AuthDeviceKey contextKey = "auth_device"
	BoardIDKey    contextKey = "board_id"
)

type AuthInfo struct {
	IsTV     bool
	Device   *models.Device
	BoardID  string
}

// ExtractAuth attempts to authenticate the request via either X-TV-Secret or Phone session cookie/header
func AuthenticateRequest(database *db.DB, r *http.Request, targetBoardID string) (*AuthInfo, error) {
	// 1. Check TV Secret Header
	tvSecret := r.Header.Get("X-TV-Secret")
	if tvSecret != "" {
		valid, err := database.ValidateTVSecret(targetBoardID, tvSecret)
		if err != nil {
			return nil, err
		}
		if valid {
			return &AuthInfo{IsTV: true, BoardID: targetBoardID}, nil
		}
		// Invalid TV secret
		return nil, db.ErrUnauthorized
	}

	// 2. Check Device Session (Cookie or X-Session-Token header)
	var sessionToken string
	cookie, err := r.Cookie("homeboard_session")
	if err == nil && cookie.Value != "" {
		sessionToken = cookie.Value
	} else {
		sessionToken = r.Header.Get("X-Session-Token")
	}

	if sessionToken != "" {
		dev, err := database.GetDeviceBySession(sessionToken)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) || errors.Is(err, db.ErrDeviceRevoked) {
				return nil, db.ErrUnauthorized
			}
			return nil, err
		}

		// Verify cross-board tenant isolation: device board must match target board
		if dev.BoardID != targetBoardID {
			return nil, db.ErrUnauthorized
		}

		// Update last seen asynchronously
		go database.UpdateDeviceLastSeen(dev.ID)

		return &AuthInfo{IsTV: false, Device: dev, BoardID: dev.BoardID}, nil
	}

	return nil, db.ErrUnauthorized
}

// RequireTVAuth ensures the caller is the authenticated Fire TV client
func RequireTVAuth(database *db.DB, boardID string, r *http.Request) error {
	tvSecret := r.Header.Get("X-TV-Secret")
	if tvSecret == "" {
		return db.ErrUnauthorized
	}

	valid, err := database.ValidateTVSecret(boardID, tvSecret)
	if err != nil || !valid {
		return db.ErrUnauthorized
	}

	return nil
}
