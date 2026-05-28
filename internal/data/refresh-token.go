package data

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"
)

type RefreshTokenModel struct {
	DB *sql.DB
}

type RefreshToken struct {
	PlainText      string
	Hash           []byte
	UserID         int64
	ExpiresAt      time.Time
	CreatedAt      time.Time
	RevokedAt      *time.Time
	FamilyID       string
	ReplacedByHash []byte
}

func generateRefreshToken(userID int64, ttl time.Duration, familyID string) (*RefreshToken, error) {
	token := &RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
		FamilyID:  familyID,
	}

	if token.FamilyID == "" {
		familyBytes := make([]byte, 16)
		_, err := rand.Read(familyBytes)
		if err != nil {
			return nil, err
		}
		token.FamilyID = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(familyBytes)
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
