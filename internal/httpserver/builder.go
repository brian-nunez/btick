package httpserver

import (
	"errors"
	"net/http"

	handlererrors "github.com/brian-nunez/go-echo-starter-template/internal/handlers/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ServerBuilder struct {
	e *echo.Echo
}

func New() *ServerBuilder {
	server := echo.New()
	server.HideBanner = true
	server.HidePort = true
	return &ServerBuilder{e: server}
}

func (b *ServerBuilder) WithDefaultMiddleware() *ServerBuilder {
	b.e.Use(middleware.Recover())
	b.e.Use(middleware.RequestID())
	b.e.Use(middleware.CORS())
	b.e.Use(middleware.Logger())
	return b
}

func (b *ServerBuilder) WithRoutes(register func(e *echo.Echo)) *ServerBuilder {
	register(b.e)
	return b
}

func (b *ServerBuilder) WithErrorHandler() *ServerBuilder {
	b.e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		status := http.StatusInternalServerError
		message := "internal server error"
		if httpErr, ok := err.(*echo.HTTPError); ok {
			status = httpErr.Code
			if msg, ok := httpErr.Message.(string); ok && msg != "" {
				message = msg
			}
		}
		if errors.Is(err, echo.ErrNotFound) {
			status = http.StatusNotFound
			message = "not found"
		}

		response := handlererrors.GenerateByStatusCode(status).WithMessage(message).Build()
		_ = c.JSON(response.HTTPStatusCode, response)
	}
	return b
}

func (b *ServerBuilder) WithNotFound() *ServerBuilder {
	notFound := func(c echo.Context) error {
		response := handlererrors.NotFound().Build()
		return c.JSON(response.HTTPStatusCode, response)
	}
	b.e.RouteNotFound("*", notFound)
	b.e.RouteNotFound("/*", notFound)
	return b
}

func (b *ServerBuilder) WithStaticAssets(directories map[string]string) *ServerBuilder {
	for path, dir := range directories {
		if dir == "" {
			continue
		}
		b.e.Static(path, dir)
	}
	return b
}

func (b *ServerBuilder) Build() *echo.Echo {
	return b.e
}
