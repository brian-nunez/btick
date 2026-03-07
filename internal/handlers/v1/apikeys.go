package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/brian-nunez/go-echo-starter-template/internal/apikeys"
	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/authorization"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	handlererrors "github.com/brian-nunez/go-echo-starter-template/internal/handlers/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type APIKeysHandler struct {
	service *apikeys.Service
}

func NewAPIKeysHandler(service *apikeys.Service) *APIKeysHandler {
	return &APIKeysHandler{service: service}
}

func (h *APIKeysHandler) CreateAPIKey(c echo.Context) error {
	var request struct {
		Name      string           `json:"name"`
		Scopes    []string         `json:"scopes"`
		ExpiresAt *json.RawMessage `json:"expires_at"`
	}
	if err := decodeJSONBody(c, &request); err != nil {
		return err
	}

	var expiresAt *time.Time
	if request.ExpiresAt != nil && string(*request.ExpiresAt) != "null" {
		var parsed time.Time
		if err := parsed.UnmarshalJSON(*request.ExpiresAt); err != nil {
			response := handlererrors.InvalidRequest().WithMessage("expires_at must be RFC3339 timestamp").Build()
			return c.JSON(response.HTTPStatusCode, response)
		}
		expiresAt = &parsed
	}

	result, err := h.service.Create(c.Request().Context(), auth.MustPrincipal(c), apikeys.CreateAPIKeyInput{
		Name:      request.Name,
		Scopes:    request.Scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if errors.Is(err, authorization.ErrForbidden) {
			response := handlererrors.Forbidden().Build()
			return c.JSON(response.HTTPStatusCode, response)
		}
		response := handlererrors.InvalidRequest().WithMessage(err.Error()).Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"api_key": toAPIKeyResponse(result.APIKey),
		"raw_key": result.RawKey,
	})
}

func (h *APIKeysHandler) ListAPIKeys(c echo.Context) error {
	keys, err := h.service.List(c.Request().Context(), auth.MustPrincipal(c))
	if err != nil {
		if errors.Is(err, authorization.ErrForbidden) {
			response := handlererrors.Forbidden().Build()
			return c.JSON(response.HTTPStatusCode, response)
		}
		response := handlererrors.InternalServerError().WithMessage(err.Error()).Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	response := make([]any, 0, len(keys))
	for _, key := range keys {
		response = append(response, toAPIKeyResponse(key))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"api_keys": response,
	})
}

func (h *APIKeysHandler) RevokeAPIKey(c echo.Context) error {
	keyID, err := uuid.Parse(c.Param("keyId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid keyId")
	}

	revoked, err := h.service.Revoke(c.Request().Context(), auth.MustPrincipal(c), keyID)
	if err != nil {
		if errors.Is(err, authorization.ErrForbidden) {
			response := handlererrors.Forbidden().Build()
			return c.JSON(response.HTTPStatusCode, response)
		}
		if errors.Is(err, apikeys.ErrAPIKeyNotFound) {
			response := handlererrors.NotFound().Build()
			return c.JSON(response.HTTPStatusCode, response)
		}
		response := handlererrors.InternalServerError().WithMessage(err.Error()).Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"api_key": toAPIKeyResponse(revoked),
	})
}

func toAPIKeyResponse(key sqlc.APIKey) map[string]any {
	var scopes []string
	if len(key.Scopes) > 0 {
		_ = json.Unmarshal(key.Scopes, &scopes)
	}

	return map[string]any{
		"id":           key.ID,
		"name":         key.Name,
		"key_prefix":   key.KeyPrefix,
		"scopes":       scopes,
		"tenant_id":    key.TenantID,
		"created_by":   key.CreatedBy,
		"last_used_at": key.LastUsedAt,
		"expires_at":   key.ExpiresAt,
		"revoked_at":   key.RevokedAt,
		"created_at":   key.CreatedAt,
	}
}
