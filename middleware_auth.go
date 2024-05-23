package main

import (
	"fmt"
	"net/http"

	"github.com/frivas/rss-agg/internal/auth"
	"github.com/frivas/rss-agg/internal/database"
)

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func (cfg *blogatorAPIConfig) middlewareAuth(handler authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			JsonResponseError(w, http.StatusForbidden, fmt.Sprintf("Auth error: %v", err))
			return
		}
		dbUser, err := cfg.DB.GetUserByAPIKey(r.Context(), apiKey)
		if err != nil {
			JsonResponseError(w, http.StatusNotFound, fmt.Sprintf("Couldn't get user: %v", err))
			return
		}
		handler(w, r, dbUser)
	}
}
