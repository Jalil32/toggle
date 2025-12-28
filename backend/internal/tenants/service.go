package tenants

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jalil32/toggle/internal/pkg/slugs"
	"github.com/jalil32/toggle/internal/pkg/transaction"
)

// UserRepository defines the minimal interface needed from users package
// This avoids circular dependency with users package
type UserRepository interface {
	UpdateLastActiveTenant(ctx context.Context, userID, tenantID string) error
}

type Service struct {
	repo      Repository
	usersRepo UserRepository
	uow       transaction.UnitOfWork
	logger    *slog.Logger
}

func NewService(repo Repository, uow transaction.UnitOfWork, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		uow:    uow,
		logger: logger,
	}
}

// SetUsersRepo sets the users repository (called after service initialization to avoid circular dependency)
func (s *Service) SetUsersRepo(usersRepo UserRepository) {
	s.usersRepo = usersRepo
}

// CreateWithOwner creates a tenant and adds the specified user as owner
// This is an atomic operation using UnitOfWork
func (s *Service) CreateWithOwner(ctx context.Context, name string, userID string) (*Tenant, error) {
	var tenant *Tenant

	// Execute tenant creation with ownership within a transaction
	err := s.uow.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Generate slug from name
		slug := slugs.Generate(name)

		// Check if slug already exists
		exists, err := s.repo.SlugExists(txCtx, slug)
		if err != nil {
			return fmt.Errorf("check slug existence: %w", err)
		}

		// If slug exists, use fallback with UUID suffix
		if exists {
			slug = slugs.WithFallback(name)
		}

		// Create tenant
		tenant, err = s.repo.Create(txCtx, name, slug)
		if err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}

		// Create membership (user is owner)
		err = s.repo.CreateMembership(txCtx, userID, tenant.ID, "owner")
		if err != nil {
			return fmt.Errorf("create tenant membership: %w", err)
		}

		// Update user's last active tenant
		if s.usersRepo != nil {
			err = s.usersRepo.UpdateLastActiveTenant(txCtx, userID, tenant.ID)
			if err != nil {
				return fmt.Errorf("update last active tenant: %w", err)
			}
		}

		s.logger.Info("tenant created with owner",
			slog.String("tenant_id", tenant.ID),
			slog.String("tenant_name", tenant.Name),
			slog.String("tenant_slug", tenant.Slug),
			slog.String("user_id", userID),
		)

		return nil
	})

	if err != nil {
		s.logger.Error("failed to create tenant with owner",
			slog.String("name", name),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return tenant, nil
}

func (s *Service) Create(ctx context.Context, name string) (*Tenant, error) {
	// Generate slug from name
	slug := slugs.Generate(name)

	// Check if slug already exists
	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		s.logger.Error("failed to check slug existence",
			slog.String("slug", slug),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// If slug exists, use fallback with UUID suffix
	if exists {
		slug = slugs.WithFallback(name)
	}

	tenant, err := s.repo.Create(ctx, name, slug)
	if err != nil {
		s.logger.Error("failed to create tenant",
			slog.String("name", name),
			slog.String("slug", slug),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("tenant created",
		slog.String("id", tenant.ID),
		slog.String("name", tenant.Name),
		slog.String("slug", tenant.Slug),
	)

	return tenant, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get tenant",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return tenant, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	tenant, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		s.logger.Error("failed to get tenant by slug",
			slog.String("slug", slug),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	return tenant, nil
}

func (s *Service) Update(ctx context.Context, id, name string) (*Tenant, error) {
	tenant, err := s.repo.Update(ctx, id, name)
	if err != nil {
		s.logger.Error("failed to update tenant",
			slog.String("id", id),
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("tenant updated",
		slog.String("id", tenant.ID),
		slog.String("name", tenant.Name),
	)

	return tenant, nil
}

// Membership methods

// GetMembership returns the role of a user in a tenant
func (s *Service) GetMembership(ctx context.Context, userID, tenantID string) (string, error) {
	return s.repo.GetMembership(ctx, userID, tenantID)
}

// ListUserTenants returns all tenants that a user is a member of
func (s *Service) ListUserTenants(ctx context.Context, userID string) ([]*TenantMembership, error) {
	return s.repo.ListUserTenants(ctx, userID)
}
