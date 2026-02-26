package api

import (
	"net/http"
	"strings"

	"api_voty/internal/utils"

	"github.com/danielgtaylor/huma/v2"
)

func AuthMiddleware(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		reqCtx := ctx.Context()
		newCtx := utils.SetUserInContext(reqCtx, claims.UserID, claims.Email)
		newHumaCtx := huma.WithContext(ctx, newCtx)

		next(newHumaCtx)
	}
}
