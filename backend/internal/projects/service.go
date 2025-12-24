package projects

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
)

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, tenantID, name string) (*Project, error) {
	project, err := s.repo.Create(ctx, tenantID, name)
	if err != nil {
		s.logger.Error("failed to create project",
			slog.String("tenant_id", tenantID),
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("project created",
		slog.String("id", project.ID),
		slog.String("name", project.Name),
		slog.String("tenant_id", tenantID),
	)

	return project, nil
}

func (s *Service) GetByID(ctx context.Context, id string, tenantID string) (*Project, error) {
	project, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("project not found or forbidden",
				slog.String("id", id),
				slog.String("tenant_id", tenantID),
			)
			return nil, pkgErrors.ErrNotFound
		}
		s.logger.Error("failed to get project",
			slog.String("id", id),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return project, nil
}

func (s *Service) ListByTenantID(ctx context.Context, tenantID string) ([]Project, error) {
	projects, err := s.repo.ListByTenantID(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to list projects",
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return projects, nil
}

func (s *Service) Delete(ctx context.Context, id string, tenantID string) error {
	err := s.repo.Delete(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("project not found or forbidden on delete",
				slog.String("id", id),
				slog.String("tenant_id", tenantID),
			)
			return pkgErrors.ErrNotFound
		}
		s.logger.Error("failed to delete project",
			slog.String("id", id),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("project deleted",
		slog.String("id", id),
		slog.String("tenant_id", tenantID),
	)

	return nil
}
