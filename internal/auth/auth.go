package auth

import (
	"auth-service/pkg/cookie"
	"net/http"

	"github.com/rs/zerolog/log"
)

func CheckAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		token, err := cookie.Read(r, "accessToken")

		if err != nil {
			log.Error().Err(err).Msg("Error encrypting token")
		}

		err = VerifyToken(token)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Error().Err(err).Msg("Failed to verify token")
			return
		}

		next.ServeHTTP(w, r)
	}
}
