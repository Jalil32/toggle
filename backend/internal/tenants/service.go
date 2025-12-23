package tenants

import (
	"context"
	"log/slog"

	"github.com/jalil32/toggle/internal/pkg/slugs"
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
