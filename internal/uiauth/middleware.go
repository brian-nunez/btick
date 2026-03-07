package uiauth

import (
	"net/http"
	"strings"

	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/labstack/echo/v4"
)

func SessionMiddleware(service *Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			principal, ok := service.PrincipalFromRequest(c.Request())
			if ok {
				auth.SetPrincipal(c, principal)
			}
			return next(c)
		}
	}
}

func RequireLogin(service *Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			principal := auth.MustPrincipal(c)
			if principal.SubjectID == "" {
				if strings.EqualFold(c.Request().Header.Get("HX-Request"), "true") {
					c.Response().Header().Set("HX-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			return next(c)
		}
	}
}

func RequireGuest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		principal := auth.MustPrincipal(c)
		if principal.SubjectID != "" {
			return c.Redirect(http.StatusSeeOther, "/ui/jobs")
		}
		return next(c)
	}
}
