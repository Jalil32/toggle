package users

import (
	"context"
	"fmt"
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

func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *Service) UpdateLastActiveTenant(ctx context.Context, userID, tenantID string) error {
	err := s.repo.UpdateLastActiveTenant(ctx, userID, tenantID)
	if err != nil {
		s.logger.Error("failed to update last active tenant",
			slog.String("user_id", userID),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("updated last active tenant",
		slog.String("user_id", userID),
		slog.String("tenant_id", tenantID),
	)

	return nil
}
