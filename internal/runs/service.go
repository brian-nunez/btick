package runs

import (
	"context"
	"database/sql"
	"errors"

	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/authorization"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/google/uuid"
)

var ErrRunNotFound = errors.New("run not found")

type Service struct {
	queries    *sqlc.Queries
	authorizer *authorization.Authorizer
}

func NewService(queries *sqlc.Queries, authorizer *authorization.Authorizer) *Service {
	return &Service{
		queries:    queries,
		authorizer: authorizer,
	}
}

func (s *Service) ListByJob(ctx context.Context, principal *auth.Principal, jobID uuid.UUID, limit int32) ([]sqlc.JobRun, error) {
	job, err := s.queries.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionRunsRead, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	return s.queries.ListRunsByJob(ctx, jobID, limit)
}

func (s *Service) Get(ctx context.Context, principal *auth.Principal, runID uuid.UUID) (sqlc.JobRun, error) {
	run, err := s.queries.GetRun(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.JobRun{}, ErrRunNotFound
		}
		return sqlc.JobRun{}, err
	}

	job, err := s.queries.GetJob(ctx, run.JobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.JobRun{}, ErrRunNotFound
		}
		return sqlc.JobRun{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionRunsRead, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return sqlc.JobRun{}, err
	}

	return run, nil
}
