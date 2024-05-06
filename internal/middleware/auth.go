package middleware

import (
	"context"
	"net/http"
	"strings"

	"aidanwoods.dev/go-paseto"
)

type CheckAuthzConfig struct {
	Paseto struct {
		Key    *paseto.V4SymmetricKey
		Parser *paseto.Parser
	}
}

func CheckAuthzHeader(config *CheckAuthzConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authzHeader := r.Header.Get("Authorization")
			if len(authzHeader) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if strings.HasPrefix(authzHeader, "Bearer ") == false {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			token := strings.TrimPrefix(authzHeader, "Bearer ")
			parsedToken, err := config.Paseto.Parser.ParseV4Local(*config.Paseto.Key, token, nil)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			userAliasId, ok := parsedToken.Claims()["user_id"].(string)
			if !ok || userAliasId == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx := context.WithValue(r.Context(), "userAliasId", userAliasId)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
