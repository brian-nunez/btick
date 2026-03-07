package authorization

import (
	"errors"
	"slices"

	"github.com/brian-nunez/baccess"
	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/google/uuid"
)

const (
	ActionJobsRead    = "jobs_read"
	ActionJobsWrite   = "jobs_write"
	ActionJobsDelete  = "jobs_delete"
	ActionJobsTrigger = "jobs_trigger"
	ActionRunsRead    = "runs_read"
	ActionKeysRead    = "keys_read"
	ActionKeysWrite   = "keys_write"
)

var ErrForbidden = errors.New("forbidden")

type Resource struct {
	TenantID *uuid.UUID
	OwnerID  *string
}

type subject struct {
	principal *auth.Principal
}

func (s subject) GetRoles() []string {
	if s.principal == nil {
		return nil
	}
	return s.principal.Roles
}

type Authorizer struct {
	evaluator *baccess.Evaluator[subject, Resource]
}

func NewAuthorizer() (*Authorizer, error) {
	rbac := baccess.NewRBAC[subject, Resource]()
	registry := baccess.NewRegistry[subject, Resource]()

	registry.Register("jobs_read_allowed", scopeAndTenantPredicate("jobs:read"))
	registry.Register("jobs_write_allowed", scopeAndTenantPredicate("jobs:write"))
	registry.Register("jobs_delete_allowed", scopeAndTenantPredicate("jobs:delete"))
	registry.Register("jobs_trigger_allowed", scopeAndTenantPredicate("jobs:trigger"))
	registry.Register("runs_read_allowed", scopeAndTenantPredicate("runs:read"))
	registry.Register("keys_read_allowed", scopeAndTenantPredicate("keys:read"))
	registry.Register("keys_write_allowed", scopeAndTenantPredicate("keys:write"))

	tenantScopedAllows := []string{
		ActionJobsRead + ":jobs_read_allowed",
		ActionJobsWrite + ":jobs_write_allowed",
		ActionJobsDelete + ":jobs_delete_allowed",
		ActionJobsTrigger + ":jobs_trigger_allowed",
		ActionRunsRead + ":runs_read_allowed",
		ActionKeysRead + ":keys_read_allowed",
		ActionKeysWrite + ":keys_write_allowed",
	}

	cfg, err := baccess.LoadConfigFromMap(map[string]any{
		"policies": map[string]any{
			"admin": map[string]any{
				"allow": tenantScopedAllows,
			},
			"user": map[string]any{
				"allow": tenantScopedAllows,
			},
			"api_key": map[string]any{
				"allow": tenantScopedAllows,
			},
			"system_admin": map[string]any{
				"allow": []string{"*"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	evaluator, err := baccess.BuildEvaluator(cfg, rbac, registry)
	if err != nil {
		return nil, err
	}

	return &Authorizer{evaluator: evaluator}, nil
}

func (a *Authorizer) Require(principal *auth.Principal, action string, resource Resource) error {
	if principal == nil {
		return ErrForbidden
	}

	allowed := a.evaluator.Evaluate(baccess.AccessRequest[subject, Resource]{
		Subject:  subject{principal: principal},
		Resource: resource,
		Action:   action,
	})
	if !allowed {
		return ErrForbidden
	}

	return nil
}

func scopeAndTenantPredicate(requiredScope string) baccess.Predicate[baccess.AccessRequest[subject, Resource]] {
	return func(req baccess.AccessRequest[subject, Resource]) bool {
		if req.Subject.principal == nil {
			return false
		}
		if req.Subject.principal.Kind == auth.PrincipalKindSystem {
			return true
		}
		if !slices.Contains(req.Subject.principal.Scopes, requiredScope) {
			return false
		}
		if req.Resource.TenantID == nil || req.Subject.principal.TenantID == nil {
			return false
		}

		return *req.Resource.TenantID == *req.Subject.principal.TenantID
	}
}
