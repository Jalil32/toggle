package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jalil32/toggle/internal/pkg/slugs"
	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/tenants"
)

type Service struct {
	repo       Repository
	tenantRepo tenants.Repository
	uow        transaction.UnitOfWork
	logger     *slog.Logger
}

func NewService(repo Repository, tenantRepo tenants.Repository, uow transaction.UnitOfWork, logger *slog.Logger) *Service {
	return &Service{
		repo:       repo,
		tenantRepo: tenantRepo,
		uow:        uow,
		logger:     logger,
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
	var user *User

	// Execute onboarding logic within a transaction
	err := s.uow.RunInTransaction(ctx, func(txCtx context.Context) error {
		var err error

		// Try to get existing user
		user, err = s.repo.GetByAuth0ID(txCtx, auth0ID)
		if err == nil {
			// User exists, check if has tenant membership
			hasMemberships, err := s.tenantRepo.HasMemberships(txCtx, user.ID)
			if err != nil {
				return fmt.Errorf("check memberships: %w", err)
			}
			if hasMemberships {
				return nil // User already onboarded
			}
			// Fall through to create default tenant
		}

		// Create user if doesn't exist
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Info("creating new user and tenant",
				slog.String("auth0_id", auth0ID),
				slog.String("firstname", firstname),
				slog.String("lastname", lastname),
			)

			// Using default values for now
			email := "default@example.com"

			user, err = s.repo.Create(txCtx, auth0ID, email, firstname, lastname)
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("get user by auth0 id: %w", err)
		}

		// Create default tenant workspace with slug generation
		tenantName := fmt.Sprintf("%s %s's Workspace", firstname, lastname)
		slug := slugs.Generate(tenantName)

		// Check slug uniqueness
		exists, err := s.tenantRepo.SlugExists(txCtx, slug)
		if err != nil {
			return fmt.Errorf("check slug existence: %w", err)
		}
		if exists {
			slug = slugs.WithFallback(tenantName)
		}

		tenant, err := s.tenantRepo.Create(txCtx, tenantName, slug)
		if err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}

		// Add user as owner of tenant
		err = s.tenantRepo.CreateMembership(txCtx, user.ID, tenant.ID, "owner")
		if err != nil {
			return fmt.Errorf("create tenant membership: %w", err)
		}

		// Update user's last active tenant ID
		err = s.repo.UpdateLastActiveTenant(txCtx, user.ID, tenant.ID)
		if err != nil {
			return fmt.Errorf("update last active tenant: %w", err)
		}

		// Update the in-memory user object to reflect the database change
		user.LastActiveTenantID = &tenant.ID

		s.logger.Info("successfully created new user and tenant",
			slog.String("user_id", user.ID),
			slog.String("tenant_id", tenant.ID),
			slog.String("tenant_slug", tenant.Slug),
			slog.String("auth0_id", auth0ID),
		)

		return nil
	})

	if err != nil {
		return nil, err
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
