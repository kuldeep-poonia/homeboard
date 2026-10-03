package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kuldeep-poonia/homeboard/backend/internal/auth"
	"github.com/kuldeep-poonia/homeboard/backend/internal/models"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrTokenExpired   = errors.New("join token has expired")
	ErrTokenUsed      = errors.New("join token has already been used")
	ErrDeviceRevoked  = errors.New("device session has been revoked")
	ErrBoardLimitItem = errors.New("board has reached maximum active items limit")
)

// Board operations

func (d *DB) CreateBoard(boardID, tvSecretHash string) (*models.Board, error) {
	now := time.Now().UTC()
	query := `INSERT INTO boards (id, tv_secret_hash, created_at) VALUES (?, ?, ?)`
	_, err := d.Exec(query, boardID, tvSecretHash, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert board: %w", err)
	}
	return &models.Board{
		ID:           boardID,
		TVSecretHash: tvSecretHash,
		CreatedAt:    now,
	}, nil
}

func (d *DB) GetBoard(boardID string) (*models.Board, error) {
	query := `SELECT id, tv_secret_hash, created_at FROM boards WHERE id = ?`
	row := d.QueryRow(query, boardID)

	var b models.Board
	err := row.Scan(&b.ID, &b.TVSecretHash, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (d *DB) ValidateTVSecret(boardID, rawSecret string) (bool, error) {
	b, err := d.GetBoard(boardID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	computedHash := auth.HashSecret(rawSecret)
	return auth.CompareHash(b.TVSecretHash, computedHash), nil
}

// Join token operations

func (d *DB) CreateJoinToken(boardID, rawToken string, ttl time.Duration) (*models.JoinToken, error) {
	tokenHash := auth.HashSecret(rawToken)
	expiresAt := time.Now().UTC().Add(ttl)

	query := `INSERT INTO join_tokens (token_hash, board_id, expires_at) VALUES (?, ?, ?)`
	_, err := d.Exec(query, tokenHash, boardID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert join token: %w", err)
	}

	return &models.JoinToken{
		TokenHash: tokenHash,
		BoardID:   boardID,
		ExpiresAt: expiresAt,
	}, nil
}

func (d *DB) RedeemJoinToken(rawToken, rawSessionToken, label string) (string, string, error) {
	tokenHash := auth.HashSecret(rawToken)
	now := time.Now().UTC()

	tx, err := d.Begin()
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	// Find token
	var boardID string
	var expiresAt time.Time
	var usedAt sql.NullTime

	query := `SELECT board_id, expires_at, used_at FROM join_tokens WHERE token_hash = ?`
	err = tx.QueryRow(query, tokenHash).Scan(&boardID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}

	if usedAt.Valid {
		return "", "", ErrTokenUsed
	}

	if now.After(expiresAt) {
		return "", "", ErrTokenExpired
	}

	// Invalidate token immediately (single-use)
	updateQuery := `UPDATE join_tokens SET used_at = ? WHERE token_hash = ?`
	if _, err := tx.Exec(updateQuery, now, tokenHash); err != nil {
		return "", "", err
	}

	// Create device session
	deviceID := uuid.New().String()
	sessionHash := auth.HashSecret(rawSessionToken)
	deviceLabel := label
	if deviceLabel == "" {
		deviceLabel = "Phone Client"
	}

	insertDev := `INSERT INTO devices (id, board_id, session_hash, label, created_at, last_seen) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := tx.Exec(insertDev, deviceID, boardID, sessionHash, deviceLabel, now, now); err != nil {
		return "", "", fmt.Errorf("failed to insert device: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", "", err
	}

	return deviceID, boardID, nil
}

// Device operations

func (d *DB) GetDeviceBySession(rawSessionToken string) (*models.Device, error) {
	sessionHash := auth.HashSecret(rawSessionToken)
	query := `SELECT id, board_id, label, created_at, last_seen, revoked_at FROM devices WHERE session_hash = ?`
	row := d.QueryRow(query, sessionHash)

	var dev models.Device
	var revokedAt sql.NullTime
	err := row.Scan(&dev.ID, &dev.BoardID, &dev.Label, &dev.CreatedAt, &dev.LastSeen, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if revokedAt.Valid {
		dev.RevokedAt = &revokedAt.Time
		return &dev, ErrDeviceRevoked
	}

	return &dev, nil
}

func (d *DB) UpdateDeviceLastSeen(deviceID string) error {
	now := time.Now().UTC()
	_, err := d.Exec(`UPDATE devices SET last_seen = ? WHERE id = ?`, now, deviceID)
	return err
}

func (d *DB) ListDevices(boardID string) ([]models.Device, error) {
	query := `SELECT id, board_id, label, created_at, last_seen, revoked_at FROM devices WHERE board_id = ? ORDER BY created_at DESC`
	rows, err := d.Query(query, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var dev models.Device
		var revokedAt sql.NullTime
		if err := rows.Scan(&dev.ID, &dev.BoardID, &dev.Label, &dev.CreatedAt, &dev.LastSeen, &revokedAt); err != nil {
			return nil, err
		}
		if revokedAt.Valid {
			dev.RevokedAt = &revokedAt.Time
		}
		devices = append(devices, dev)
	}
	return devices, rows.Err()
}

func (d *DB) RevokeDevice(boardID, deviceID string) error {
	now := time.Now().UTC()
	res, err := d.Exec(`UPDATE devices SET revoked_at = ? WHERE id = ? AND board_id = ? AND revoked_at IS NULL`, now, deviceID, boardID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Items operations

func (d *DB) CountActiveItems(boardID string) (int, error) {
	query := `SELECT COUNT(*) FROM items WHERE board_id = ? AND archived = 0`
	var count int
	err := d.QueryRow(query, boardID).Scan(&count)
	return count, err
}

func (d *DB) CreateItem(item *models.Item) error {
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.CreatedAt = now
	item.UpdatedAt = now

	query := `INSERT INTO items (id, board_id, type, text, when_ts, done, archived, created_by_device, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	doneInt := 0
	if item.Done {
		doneInt = 1
	}
	archivedInt := 0
	if item.Archived {
		archivedInt = 1
	}

	_, err := d.Exec(query, item.ID, item.BoardID, item.Type, item.Text, item.WhenTS, doneInt, archivedInt, item.CreatedByDevice, item.CreatedAt, item.UpdatedAt)
	return err
}

func (d *DB) GetItem(boardID, itemID string) (*models.Item, error) {
	query := `SELECT id, board_id, type, text, when_ts, done, archived, created_by_device, created_at, updated_at
			  FROM items WHERE id = ? AND board_id = ?`

	row := d.QueryRow(query, itemID, boardID)
	var item models.Item
	var whenTS sql.NullTime
	var createdBy sql.NullString
	var doneInt, archivedInt int

	err := row.Scan(&item.ID, &item.BoardID, &item.Type, &item.Text, &whenTS, &doneInt, &archivedInt, &createdBy, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if whenTS.Valid {
		item.WhenTS = &whenTS.Time
	}
	if createdBy.Valid {
		item.CreatedByDevice = &createdBy.String
	}
	item.Done = (doneInt == 1)
	item.Archived = (archivedInt == 1)

	return &item, nil
}

func (d *DB) ListItems(boardID string, includeArchived bool) ([]models.Item, error) {
	query := `SELECT id, board_id, type, text, when_ts, done, archived, created_by_device, created_at, updated_at
			  FROM items WHERE board_id = ?`
	if !includeArchived {
		query += ` AND archived = 0`
	}
	query += ` ORDER BY done ASC, created_at DESC`

	rows, err := d.Query(query, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		var item models.Item
		var whenTS sql.NullTime
		var createdBy sql.NullString
		var doneInt, archivedInt int

		err := rows.Scan(&item.ID, &item.BoardID, &item.Type, &item.Text, &whenTS, &doneInt, &archivedInt, &createdBy, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if whenTS.Valid {
			item.WhenTS = &whenTS.Time
		}
		if createdBy.Valid {
			item.CreatedByDevice = &createdBy.String
		}
		item.Done = (doneInt == 1)
		item.Archived = (archivedInt == 1)

		items = append(items, item)
	}

	return items, rows.Err()
}

func (d *DB) UpdateItem(boardID, itemID string, req *models.UpdateItemRequest) (*models.Item, error) {
	item, err := d.GetItem(boardID, itemID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if req.Text != nil {
		item.Text = *req.Text
	}
	if req.WhenTS != nil {
		item.WhenTS = req.WhenTS
	}
	if req.Done != nil {
		item.Done = *req.Done
	}
	if req.Archived != nil {
		item.Archived = *req.Archived
	}
	item.UpdatedAt = now

	doneInt := 0
	if item.Done {
		doneInt = 1
	}
	archivedInt := 0
	if item.Archived {
		archivedInt = 1
	}

	query := `UPDATE items SET text = ?, when_ts = ?, done = ?, archived = ?, updated_at = ? WHERE id = ? AND board_id = ?`
	res, err := d.Exec(query, item.Text, item.WhenTS, doneInt, archivedInt, item.UpdatedAt, itemID, boardID)
	if err != nil {
		return nil, err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return item, nil
}

func (d *DB) DeleteItem(boardID, itemID string) error {
	query := `DELETE FROM items WHERE id = ? AND board_id = ?`
	res, err := d.Exec(query, itemID, boardID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
