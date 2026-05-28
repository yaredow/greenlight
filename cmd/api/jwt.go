package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type authToken struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

var ErrInvalidJWTToken = errors.New("invalid jwt token")

func (app *application) generateJWT(userID int64) (*authToken, error) {
	expiry := time.Now().Add(10 * time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "greenlight",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(app.config.jwt.secret))
	if err != nil {
		return nil, err
	}

	return &authToken{
		Token:  tokenString,
		Expiry: expiry,
	}, nil
}

func (app *application) validateJWT(tokenPlainText string) (int64, error) {
	if app.config.jwt.secret == "" {
		return 0, ErrInvalidJWTToken
	}

	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenPlainText, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidJWTToken
		}
		return []byte(app.config.jwt.secret), nil
	})
	if err != nil {
		return 0, ErrInvalidJWTToken
	}

	if !token.Valid {
		return 0, ErrInvalidJWTToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, err
	}

	return userID, nil
}
