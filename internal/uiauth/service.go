package uiauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "btick_session"

type Config struct {
	SessionSecret string
	SessionTTL    time.Duration
}

type Service struct {
	queries *sqlc.Queries
	config  Config
	key     []byte
}

type sessionClaims struct {
	SubjectID string   `json:"sub"`
	TenantID  *string  `json:"tenant_id,omitempty"`
	Roles     []string `json:"roles"`
	Scopes    []string `json:"scopes"`
	Kind      string   `json:"kind"`
	ExpiresAt int64    `json:"exp"`
}

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrEmailAlreadyExists = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

func NewService(queries *sqlc.Queries, config Config) (*Service, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	if strings.TrimSpace(config.SessionSecret) == "" {
		return nil, fmt.Errorf("ui session secret is required")
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = 12 * time.Hour
	}

	return &Service{
		queries: queries,
		config:  config,
		key:     []byte(config.SessionSecret),
	}, nil
}

func (s *Service) Register(ctx context.Context, email string, password string) (*auth.Principal, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	rolesJSON, err := json.Marshal([]string{"user"})
	if err != nil {
		return nil, fmt.Errorf("marshal roles: %w", err)
	}
	scopesJSON, err := json.Marshal(auth.DefaultAdminScopes())
	if err != nil {
		return nil, fmt.Errorf("marshal scopes: %w", err)
	}
	tenantID := uuid.New()

	user, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           uuid.New(),
		Email:        normalizedEmail,
		PasswordHash: string(passwordHash),
		Roles:        rolesJSON,
		Scopes:       scopesJSON,
		TenantID:     &tenantID,
	})
	if err != nil {
		if isUniqueEmailViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	principal, err := buildPrincipalFromUser(user)
	if err != nil {
		return nil, err
	}
	return principal, nil
}

func (s *Service) Login(ctx context.Context, email string, password string) (*auth.Principal, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.queries.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	principal, err := buildPrincipalFromUser(user)
	if err != nil {
		return nil, err
	}
	return principal, nil
}

func (s *Service) NewSessionCookie(principal *auth.Principal, secure bool) (*http.Cookie, error) {
	if principal == nil {
		return nil, fmt.Errorf("principal is required")
	}

	var tenantID *string
	if principal.TenantID != nil {
		value := principal.TenantID.String()
		tenantID = &value
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.config.SessionTTL)
	claims := sessionClaims{
		SubjectID: principal.SubjectID,
		TenantID:  tenantID,
		Roles:     principal.Roles,
		Scopes:    principal.Scopes,
		Kind:      string(principal.Kind),
		ExpiresAt: expiresAt.Unix(),
	}
	token, err := s.signClaims(claims)
	if err != nil {
		return nil, err
	}

	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	}, nil
}

func (s *Service) ClearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func (s *Service) PrincipalFromRequest(request *http.Request) (*auth.Principal, bool) {
	cookie, err := request.Cookie(SessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, false
	}

	claims, err := s.verifyToken(cookie.Value)
	if err != nil {
		return nil, false
	}
	if time.Now().UTC().Unix() >= claims.ExpiresAt {
		return nil, false
	}

	var tenantID *uuid.UUID
	if claims.TenantID != nil {
		parsed, err := uuid.Parse(*claims.TenantID)
		if err == nil {
			tenantID = &parsed
		}
	}

	kind := auth.PrincipalKind(claims.Kind)
	if kind == "" {
		kind = auth.PrincipalKindUser
	}

	principal := &auth.Principal{
		SubjectID: claims.SubjectID,
		TenantID:  tenantID,
		Roles:     claims.Roles,
		Scopes:    claims.Scopes,
		Kind:      kind,
	}
	if principal.Kind != auth.PrincipalKindSystem && principal.TenantID == nil {
		return nil, false
	}
	return principal, true
}

func (s *Service) signClaims(claims sessionClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal session claims: %w", err)
	}

	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign([]byte(payloadPart))
	signaturePart := base64.RawURLEncoding.EncodeToString(signature)

	return payloadPart + "." + signaturePart, nil
}

func (s *Service) verifyToken(token string) (*sessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session token")
	}

	payloadPart := parts[0]
	signaturePart := parts[1]

	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return nil, fmt.Errorf("decode session signature: %w", err)
	}

	expectedSignature := s.sign([]byte(payloadPart))
	if subtle.ConstantTimeCompare(signature, expectedSignature) != 1 {
		return nil, fmt.Errorf("invalid session signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, fmt.Errorf("decode session payload: %w", err)
	}

	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("decode session claims: %w", err)
	}

	return &claims, nil
}

func (s *Service) sign(value []byte) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write(value)
	return mac.Sum(nil)
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func isUniqueEmailViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "users_email_key") || strings.Contains(message, "duplicate")
}

func buildPrincipalFromUser(user sqlc.User) (*auth.Principal, error) {
	roles := []string{"admin"}
	if len(user.Roles) > 0 {
		if err := json.Unmarshal(user.Roles, &roles); err != nil {
			return nil, fmt.Errorf("decode user roles: %w", err)
		}
	}

	scopes := auth.DefaultAdminScopes()
	if len(user.Scopes) > 0 {
		if err := json.Unmarshal(user.Scopes, &scopes); err != nil {
			return nil, fmt.Errorf("decode user scopes: %w", err)
		}
	}

	return &auth.Principal{
		SubjectID: user.ID.String(),
		TenantID:  user.TenantID,
		Roles:     roles,
		Scopes:    scopes,
		Kind:      auth.PrincipalKindUser,
	}, nil
}
