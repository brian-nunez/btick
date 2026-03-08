package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/brian-nunez/btick/internal/auth"
	"github.com/brian-nunez/btick/internal/authorization"
	"github.com/brian-nunez/btick/internal/db/sqlc"
	"github.com/google/uuid"
)

var (
	ErrAPIKeyNotFound = errors.New("api key not found")
)

var allowedScopes = []string{
	"jobs:read",
	"jobs:write",
	"jobs:delete",
	"jobs:trigger",
	"runs:read",
	"keys:read",
	"keys:write",
}

type Service struct {
	queries    *sqlc.Queries
	authorizer *authorization.Authorizer
}

type CreateAPIKeyInput struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type CreateAPIKeyOutput struct {
	APIKey sqlc.APIKey `json:"api_key"`
	RawKey string      `json:"raw_key"`
}

func NewService(queries *sqlc.Queries, authorizer *authorization.Authorizer) *Service {
	return &Service{
		queries:    queries,
		authorizer: authorizer,
	}
}

func (s *Service) Create(ctx context.Context, principal *auth.Principal, input CreateAPIKeyInput) (CreateAPIKeyOutput, error) {
	if err := s.authorizer.Require(principal, authorization.ActionKeysWrite, authorization.Resource{TenantID: principal.TenantID}); err != nil {
		return CreateAPIKeyOutput{}, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return CreateAPIKeyOutput{}, fmt.Errorf("name is required")
	}
	if len(input.Scopes) == 0 {
		return CreateAPIKeyOutput{}, fmt.Errorf("at least one scope is required")
	}
	for _, scope := range input.Scopes {
		if !slices.Contains(allowedScopes, scope) {
			return CreateAPIKeyOutput{}, fmt.Errorf("invalid scope %q", scope)
		}
	}

	rawKey, keyPrefix, keyHash, err := generateRawKey()
	if err != nil {
		return CreateAPIKeyOutput{}, fmt.Errorf("generate api key: %w", err)
	}

	scopesJSON, err := json.Marshal(input.Scopes)
	if err != nil {
		return CreateAPIKeyOutput{}, fmt.Errorf("encode scopes: %w", err)
	}

	var createdBy *string
	if principal.SubjectID != "" {
		createdBy = &principal.SubjectID
	}

	key, err := s.queries.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(input.Name),
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
		Scopes:    scopesJSON,
		TenantID:  principal.TenantID,
		CreatedBy: createdBy,
		ExpiresAt: input.ExpiresAt,
	})
	if err != nil {
		return CreateAPIKeyOutput{}, err
	}

	return CreateAPIKeyOutput{
		APIKey: key,
		RawKey: rawKey,
	}, nil
}

func (s *Service) List(ctx context.Context, principal *auth.Principal) ([]sqlc.APIKey, error) {
	if err := s.authorizer.Require(principal, authorization.ActionKeysRead, authorization.Resource{TenantID: principal.TenantID}); err != nil {
		return nil, err
	}

	keys, err := s.queries.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	if principal.Kind == auth.PrincipalKindSystem && principal.TenantID == nil {
		return keys, nil
	}

	filtered := make([]sqlc.APIKey, 0, len(keys))
	for _, key := range keys {
		if key.TenantID != nil && principal.TenantID != nil && *key.TenantID == *principal.TenantID {
			filtered = append(filtered, key)
		}
	}
	return filtered, nil
}

func (s *Service) Revoke(ctx context.Context, principal *auth.Principal, keyID uuid.UUID) (sqlc.APIKey, error) {
	target, err := s.queries.GetAPIKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.APIKey{}, ErrAPIKeyNotFound
		}
		return sqlc.APIKey{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionKeysWrite, authorization.Resource{TenantID: target.TenantID, OwnerID: target.CreatedBy}); err != nil {
		return sqlc.APIKey{}, err
	}

	revoked, err := s.queries.RevokeAPIKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.APIKey{}, ErrAPIKeyNotFound
		}
		return sqlc.APIKey{}, err
	}

	return revoked, nil
}

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*auth.Principal, error) {
	keyPrefix, err := parseKeyPrefix(rawKey)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	storedKey, err := s.queries.GetAPIKeyByPrefix(ctx, keyPrefix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}

	if storedKey.RevokedAt != nil {
		return nil, auth.ErrInvalidToken
	}
	if storedKey.ExpiresAt != nil && storedKey.ExpiresAt.Before(time.Now().UTC()) {
		return nil, auth.ErrInvalidToken
	}

	presentedHash := hashAPIKey(rawKey)
	if subtle.ConstantTimeCompare([]byte(presentedHash), []byte(storedKey.KeyHash)) != 1 {
		return nil, auth.ErrInvalidToken
	}

	var scopes []string
	if err := json.Unmarshal(storedKey.Scopes, &scopes); err != nil {
		return nil, fmt.Errorf("decode api key scopes: %w", err)
	}

	_ = s.queries.UpdateAPIKeyLastUsedAt(ctx, storedKey.ID)

	principal := &auth.Principal{
		SubjectID: storedKey.ID.String(),
		TenantID:  storedKey.TenantID,
		Roles:     []string{"api_key"},
		Scopes:    scopes,
		Kind:      auth.PrincipalKindAPIKey,
		APIKeyID:  &storedKey.ID,
	}

	return principal, nil
}

func generateRawKey() (rawKey string, keyPrefix string, keyHash string, err error) {
	prefixBytes := make([]byte, 6)
	if _, err := rand.Read(prefixBytes); err != nil {
		return "", "", "", err
	}
	secretBytes := make([]byte, 24)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", "", err
	}

	prefix := hex.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	raw := fmt.Sprintf("skd_%s_%s", prefix, secret)
	return raw, "skd_" + prefix, hashAPIKey(raw), nil
}

func parseKeyPrefix(rawKey string) (string, error) {
	trimmed := strings.TrimSpace(rawKey)
	if trimmed == "" {
		return "", fmt.Errorf("invalid api key format")
	}

	parts := strings.SplitN(trimmed, "_", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid api key format")
	}
	if parts[0] != "skd" || parts[1] == "" || parts[2] == "" {
		return "", fmt.Errorf("invalid api key format")
	}
	return "skd_" + parts[1], nil
}

func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}
