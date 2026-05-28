package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yaredow/greenlight/internal/validator"
)

type RefreshTokenModel struct {
	DB *sql.DB
}

type RefreshToken struct {
	PlainText      string     `json:"token"`
	Hash           []byte     `json:"-"`
	UserID         int64      `json:"-"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"-"`
	RevokedAt      *time.Time `json:"-"`
	FamilyID       string     `json:"-"`
	ReplacedByHash []byte     `json:"-"`
}

func generateRefreshToken(userID int64, ttl time.Duration, familyID string) (*RefreshToken, error) {
	token := &RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
		FamilyID:  familyID,
	}

	if token.FamilyID == "" {
		token.FamilyID = uuid.NewString()
	}

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:]

	return token, nil
}

func (m RefreshTokenModel) Insert(token *RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (hash, user_id, expires_at, family_id)
		VALUES ($1, $2, $3, $4)

		RETURNING created_at
	`

	args := []any{
		token.Hash,
		token.UserID,
		token.ExpiresAt,
		token.FamilyID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&token.CreatedAt)
}

func (m RefreshTokenModel) GetByPlainText(tokenPlainText string) (*RefreshToken, error) {
	hash := sha256.Sum256([]byte(tokenPlainText))
	query := `
		SELECT hash, user_id, expires_at, created_at, revoked_at, family_id, replaced_by_hash
		FROM refresh_tokens
		WHERE hash = $1
		AND expires_at > $2
		AND revoked_at IS NULL
	`

	var token RefreshToken
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, hash[:], time.Now()).Scan(
		&token.Hash,
		&token.UserID,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.FamilyID,
		&token.ReplacedByHash,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	token.PlainText = tokenPlainText

	return &token, nil
}

func ValidateRefreshToken(v *validator.Validator, tokenPlainText string) {
	v.Check(tokenPlainText != "", "refresh_token", "must be provided")
	v.Check(len(tokenPlainText) == 52, "refresh_token", "must be 52 bytes long")
}

func (m RefreshTokenModel) New(userID int64, ttl time.Duration) (*RefreshToken, error) {
	token, err := generateRefreshToken(userID, ttl, "")
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m RefreshTokenModel) Rotate(oldToken *RefreshToken, ttl time.Duration) (*RefreshToken, error) {
	newToken, err := generateRefreshToken(oldToken.UserID, ttl, oldToken.FamilyID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertQuery := `
			INSERT INTO refresh_tokens (hash, user_id, expires_at, family_id)
			VALUES ($1, $2, $3, $4)
			RETURNING created_at
		`

	insertArgs := []any{
		newToken.Hash,
		newToken.UserID,
		newToken.ExpiresAt,
		newToken.FamilyID,
	}

	err = tx.QueryRowContext(ctx, insertQuery, insertArgs...).Scan(&newToken.CreatedAt)
	if err != nil {
		return nil, err
	}

	updateQuery := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by_hash = $1
		WHERE hash = $2
		AND revoked_at IS NULL
	`
	updateArgs := []any{
		newToken.Hash,
		oldToken.Hash,
	}

	result, err := tx.ExecContext(ctx, updateQuery, updateArgs...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected != 1 {
		return nil, ErrRecordNotFound
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return newToken, nil
}

func (m RefreshTokenModel) RevokeByHash(hash []byte) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE hash = $1
		AND revoked_at IS NULL
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, hash)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return ErrRecordNotFound
	}

	return nil
}

func (m RefreshTokenModel) RevokeFamily(familyID string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE family_id = $1
		AND revoked_at IS NULL
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, familyID)
	return err
}
