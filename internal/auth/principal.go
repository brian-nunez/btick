package auth

import "github.com/google/uuid"

type PrincipalKind string

const (
	PrincipalKindAPIKey PrincipalKind = "api_key"
	PrincipalKindUser   PrincipalKind = "user"
	PrincipalKindSystem PrincipalKind = "system"
)

type Principal struct {
	SubjectID string
	TenantID  *uuid.UUID
	Roles     []string
	Scopes    []string
	Kind      PrincipalKind
	APIKeyID  *uuid.UUID
}

func (p Principal) GetRoles() []string {
	return p.Roles
}

func DefaultAdminScopes() []string {
	return []string{
		"jobs:read",
		"jobs:write",
		"jobs:delete",
		"jobs:trigger",
		"runs:read",
		"keys:read",
		"keys:write",
	}
}

func NewAdminPrincipal(subjectID string, kind PrincipalKind) *Principal {
	if subjectID == "" {
		subjectID = "admin"
	}
	return &Principal{
		SubjectID: subjectID,
		Roles:     []string{"admin"},
		Scopes:    DefaultAdminScopes(),
		Kind:      kind,
	}
}

func SystemPrincipal() *Principal {
	return NewAdminPrincipal("system", PrincipalKindSystem)
}
