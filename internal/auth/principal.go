package auth

import "github.com/google/uuid"

type PrincipalKind string

const (
	PrincipalKindAPIKey PrincipalKind = "api_key"
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

func SystemPrincipal() *Principal {
	return &Principal{
		SubjectID: "system",
		Roles:     []string{"admin"},
		Scopes: []string{
			"jobs:read",
			"jobs:write",
			"jobs:delete",
			"jobs:trigger",
			"runs:read",
			"keys:read",
			"keys:write",
		},
		Kind: PrincipalKindSystem,
	}
}
