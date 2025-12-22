package users

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jalil32/toggle/internal/organizations"
)

type Service struct {
	repo    Repository
	orgRepo organizations.Repository
	logger  *slog.Logger
}

func NewService(repo Repository, orgRepo organizations.Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		orgRepo: orgRepo,
		logger:  logger,
	}
}

func (s *Service) GetByAuth0ID(ctx context.Context, auth0ID string) (*User, error) {
	user, err := s.repo.GetByAuth0ID(ctx, auth0ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("failed to get user by auth0 id",
			slog.String("auth0_id", auth0ID),
			slog.String("error", err.Error()),
		)
	}
	return user, err
}

func (s *Service) GetOrCreate(ctx context.Context, auth0ID, firstname, lastname string) (*User, error) {
	user, err := s.repo.GetByAuth0ID(ctx, auth0ID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("failed to get user by auth0 id in GetOrCreate",
			slog.String("auth0_id", auth0ID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// First time user - create org and user
	s.logger.Info("creating new user and organization",
		slog.String("auth0_id", auth0ID),
		slog.String("firstname", firstname),
		slog.String("lastname", lastname),
	)

	org, err := s.orgRepo.Create(ctx, firstname+" "+lastname+"'s Organization")
	if err != nil {
		s.logger.Error("failed to create organization for new user",
			slog.String("auth0_id", auth0ID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Using default values for now
	email := "default@example.com"

	user, err = s.repo.Create(ctx, auth0ID, org.ID, email, firstname, lastname, "owner")
	if err != nil {
		s.logger.Error("failed to create user",
			slog.String("auth0_id", auth0ID),
			slog.String("org_id", org.ID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("successfully created new user and organization",
		slog.String("user_id", user.ID),
		slog.String("org_id", org.ID),
		slog.String("auth0_id", auth0ID),
	)

	return user, nil
}
