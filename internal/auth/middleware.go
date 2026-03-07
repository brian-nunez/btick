package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, rawKey string) (*Principal, error)
}

var ErrInvalidToken = errors.New("invalid api key")

func APIKeyAuthMiddleware(authenticator APIKeyAuthenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authorizationHeader := strings.TrimSpace(c.Request().Header.Get("Authorization"))
			if !strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}

			rawToken := strings.TrimSpace(authorizationHeader[len("Bearer "):])
			if rawToken == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}

			principal, err := authenticator.Authenticate(c.Request().Context(), rawToken)
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
				}
				return err
			}

			SetPrincipal(c, principal)
			return next(c)
		}
	}
}
