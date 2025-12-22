package organizations

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

func (s *Service) Create(ctx context.Context, name string) (*Organization, error) {
	org, err := s.repo.Create(ctx, name)
	if err != nil {
		s.logger.Error("failed to create organization",
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("organization created",
		slog.String("id", org.ID),
		slog.String("name", org.Name),
	)

	return org, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Organization, error) {
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get organization",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return org, nil
}

func (s *Service) Update(ctx context.Context, id, name string) (*Organization, error) {
	org, err := s.repo.Update(ctx, id, name)
	if err != nil {
		s.logger.Error("failed to update organization",
			slog.String("id", id),
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("organization updated",
		slog.String("id", org.ID),
		slog.String("name", org.Name),
	)

	return org, nil
}
