package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/yaredow/greenlight/internal/data"
	"github.com/yaredow/greenlight/internal/validator"
)

func (app *application) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshTokenPlainText string `json:"refresh_token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateRefreshToken(v, input.RefreshTokenPlainText); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.RefreshTokens.GetByPlainText(input.RefreshTokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidAuthenticationTokenResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	accessToken, err := app.generateJWT(token.UserID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	refreshToken, err := app.models.RefreshTokens.Rotate(token, 7*24*time.Hour)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidAuthenticationTokenResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"access_token": accessToken, "refresh_token": refreshToken}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) revokeRefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshTokenPlainText string `json:"refresh_token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateRefreshToken(v, input.RefreshTokenPlainText); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.RefreshTokens.GetByPlainText(input.RefreshTokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidAuthenticationTokenResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.RefreshTokens.RevokeByHash(token.Hash)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidAuthenticationTokenResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	env := envelope{"message": "refresh token revoked"}
	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
