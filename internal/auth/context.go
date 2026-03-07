package auth

import "github.com/labstack/echo/v4"

const principalContextKey = "principal"

func SetPrincipal(c echo.Context, principal *Principal) {
	c.Set(principalContextKey, principal)
}

func MustPrincipal(c echo.Context) *Principal {
	value := c.Get(principalContextKey)
	principal, ok := value.(*Principal)
	if !ok || principal == nil {
		return SystemPrincipal()
	}
	return principal
}
