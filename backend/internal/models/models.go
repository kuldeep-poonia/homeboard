package models

import (
	"errors"
	"strings"
	"time"
)

// Allowed item types
var ValidItemTypes = map[string]bool{
	"reminder": true,
	"shopping": true,
	"event":    true,
	"movie":    true,
	"status":   true,
}

type Board struct {
	ID           string    `json:"id"`
	TVSecretHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateBoardResponse struct {
	BoardID  string `json:"board_id"`
	TVSecret string `json:"tv_secret"`
}

type JoinToken struct {
	TokenHash string     `json:"-"`
	BoardID   string     `json:"board_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

type CreateJoinTokenResponse struct {
	Token     string    `json:"token"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RedeemTokenRequest struct {
	Token string `json:"token"`
	Label string `json:"label,omitempty"`
}

type Device struct {
	ID          string     `json:"id"`
	BoardID     string     `json:"board_id"`
	SessionHash string     `json:"-"`
	Label       string     `json:"label"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeen    time.Time  `json:"last_seen"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

type Item struct {
	ID              string     `json:"id"`
	BoardID         string     `json:"board_id"`
	Type            string     `json:"type"`
	Text            string     `json:"text"`
	WhenTS          *time.Time `json:"when_ts,omitempty"`
	Done            bool       `json:"done"`
	Archived        bool       `json:"archived"`
	CreatedByDevice *string    `json:"created_by_device,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateItemRequest struct {
	Type   string     `json:"type"`
	Text   string     `json:"text"`
	WhenTS *time.Time `json:"when_ts,omitempty"`
}

func (r *CreateItemRequest) Validate() error {
	r.Type = strings.TrimSpace(strings.ToLower(r.Type))
	if !ValidItemTypes[r.Type] {
		return errors.New("invalid item type: must be reminder, shopping, event, movie, or status")
	}

	r.Text = strings.TrimSpace(r.Text)
	if r.Text == "" {
		return errors.New("item text cannot be empty or whitespace-only")
	}

	// Count UTF-8 characters (runes), max 200 chars
	if len([]rune(r.Text)) > 200 {
		return errors.New("item text cannot exceed 200 characters")
	}

	return nil
}

type UpdateItemRequest struct {
	Text     *string    `json:"text,omitempty"`
	WhenTS   *time.Time `json:"when_ts,omitempty"`
	Done     *bool      `json:"done,omitempty"`
	Archived *bool      `json:"archived,omitempty"`
}

func (r *UpdateItemRequest) Validate() error {
	if r.Text != nil {
		trimmed := strings.TrimSpace(*r.Text)
		if trimmed == "" {
			return errors.New("item text cannot be empty or whitespace-only")
		}
		if len([]rune(trimmed)) > 200 {
			return errors.New("item text cannot exceed 200 characters")
		}
		*r.Text = trimmed
	}
	return nil
}
