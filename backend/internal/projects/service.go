package projects

import (
	"context"
	"log/slog"
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

func (s *Service) GetByID(ctx context.Context, id string) (*Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get project",
			slog.String("id", id),
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

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("failed to delete project",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("project deleted", slog.String("id", id))

	return nil
}
