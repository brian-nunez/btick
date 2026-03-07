package v1

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brian-nunez/go-echo-starter-template/internal/authorization"
	handlererrors "github.com/brian-nunez/go-echo-starter-template/internal/handlers/errors"
	"github.com/brian-nunez/go-echo-starter-template/internal/jobs"
	"github.com/brian-nunez/go-echo-starter-template/internal/runs"
	"github.com/labstack/echo/v4"
)

func writeAPIError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, authorization.ErrForbidden) {
		response := handlererrors.Forbidden().Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	if errors.Is(err, jobs.ErrNotFound) || errors.Is(err, runs.ErrRunNotFound) {
		response := handlererrors.NotFound().Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	var validationErrors jobs.ValidationErrors
	if errors.As(err, &validationErrors) {
		mapped := make([]handlererrors.ValidationError, 0, len(validationErrors))
		for _, item := range validationErrors {
			mapped = append(mapped, handlererrors.ValidationError{
				Field:   item.Field,
				Message: item.Message,
			})
		}
		response := handlererrors.InvalidRequest().WithValidation(mapped).WithMessage("Validation failed").Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	if errors.Is(err, sql.ErrNoRows) {
		response := handlererrors.NotFound().Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	response := handlererrors.InternalServerError().WithMessage(err.Error()).Build()
	return c.JSON(response.HTTPStatusCode, response)
}

func parseUUIDParam(c echo.Context, key string) (string, error) {
	value := c.Param(key)
	if value == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing path parameter")
	}
	return value, nil
}

func decodeJSONBody(c echo.Context, destination any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	return nil
}
