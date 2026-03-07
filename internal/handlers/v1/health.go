package v1

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) V1Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HealthHandler) Healthz(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func (h *HealthHandler) Readyz(c echo.Context) error {
	if err := h.db.PingContext(c.Request().Context()); err != nil {
		return c.String(http.StatusServiceUnavailable, "not ready")
	}
	return c.String(http.StatusOK, "ready")
}
