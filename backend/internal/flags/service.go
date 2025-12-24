package flag

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
	"github.com/jalil32/toggle/internal/pkg/validator"
)

var (
	ErrFlagNotFound    = errors.New("flag not found")
	ErrInvalidFlagData = errors.New("invalid flag data")
)

type Service interface {
	Create(ctx context.Context, f *Flag, tenantID string) error
	GetByID(ctx context.Context, id string, tenantID string) (*Flag, error)
	List(ctx context.Context, tenantID string) ([]Flag, error)
	Update(ctx context.Context, f *Flag, tenantID string) error
	Delete(ctx context.Context, id string, tenantID string) error
}

type service struct {
	repo      Repository
	validator validator.Validator
	logger    *slog.Logger
}

func NewService(repo Repository, val validator.Validator, logger *slog.Logger) Service {
	return &service{
		repo:      repo,
		validator: val,
		logger:    logger,
	}
}

func (s *service) Create(ctx context.Context, f *Flag, tenantID string) error {
	if err := s.validateFlag(f); err != nil {
		if f != nil {
			s.logger.Warn("flag validation failed",
				slog.String("name", f.Name),
				slog.String("error", err.Error()),
			)
		} else {
			s.logger.Warn("flag validation failed: nil flag",
				slog.String("error", err.Error()),
			)
		}
		return err
	}

	// CRITICAL: Validate project belongs to tenant before creating flag
	if err := s.validator.ValidateProjectOwnership(ctx, f.ProjectID, tenantID); err != nil {
		s.logger.Warn("project ownership validation failed",
			slog.String("project_id", f.ProjectID),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return pkgErrors.ErrProjectNotInTenant
	}

	if err := s.repo.Create(ctx, f); err != nil {
		s.logger.Error("failed to create flag",
			slog.String("name", f.Name),
			slog.String("project_id", f.ProjectID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create flag: %w", err)
	}

	s.logger.Info("flag created",
		slog.String("id", f.ID),
		slog.String("name", f.Name),
		slog.String("project_id", f.ProjectID),
		slog.String("tenant_id", tenantID),
	)

	return nil
}

func (s *service) GetByID(ctx context.Context, id string, tenantID string) (*Flag, error) {
	if id == "" {
		return nil, ErrInvalidFlagData
	}

	flag, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found or forbidden",
				slog.String("id", id),
				slog.String("tenant_id", tenantID),
			)
			return nil, pkgErrors.ErrNotFound
		}
		s.logger.Error("failed to get flag",
			slog.String("id", id),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get flag: %w", err)
	}

	return flag, nil
}

func (s *service) List(ctx context.Context, tenantID string) ([]Flag, error) {
	flags, err := s.repo.List(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to list flags",
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}

	if flags == nil {
		return []Flag{}, nil
	}

	return flags, nil
}

func (s *service) Update(ctx context.Context, f *Flag, tenantID string) error {
	if err := s.validateFlag(f); err != nil {
		if f != nil {
			s.logger.Warn("flag validation failed on update",
				slog.String("id", f.ID),
				slog.String("error", err.Error()),
			)
		} else {
			s.logger.Warn("flag validation failed on update: nil flag",
				slog.String("error", err.Error()),
			)
		}
		return err
	}

	if f.ID == "" {
		return ErrInvalidFlagData
	}

	// Validate project ownership if project_id is being changed
	if f.ProjectID != "" {
		if err := s.validator.ValidateProjectOwnership(ctx, f.ProjectID, tenantID); err != nil {
			s.logger.Warn("project ownership validation failed on update",
				slog.String("flag_id", f.ID),
				slog.String("project_id", f.ProjectID),
				slog.String("tenant_id", tenantID),
			)
			return pkgErrors.ErrProjectNotInTenant
		}
	}

	if err := s.repo.Update(ctx, f, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found or forbidden on update",
				slog.String("id", f.ID),
				slog.String("tenant_id", tenantID),
			)
			return pkgErrors.ErrNotFound
		}
		s.logger.Error("failed to update flag",
			slog.String("id", f.ID),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update flag: %w", err)
	}

	s.logger.Info("flag updated",
		slog.String("id", f.ID),
		slog.String("name", f.Name),
		slog.String("tenant_id", tenantID),
	)

	return nil
}

func (s *service) Delete(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return ErrInvalidFlagData
	}

	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found or forbidden on delete",
				slog.String("id", id),
				slog.String("tenant_id", tenantID),
			)
			return pkgErrors.ErrNotFound
		}
		s.logger.Error("failed to delete flag",
			slog.String("id", id),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	s.logger.Info("flag deleted",
		slog.String("id", id),
		slog.String("tenant_id", tenantID),
	)

	return nil
}

func (s *service) validateFlag(f *Flag) error {
	if f == nil {
		return ErrInvalidFlagData
	}

	if f.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidFlagData)
	}

	return nil
}

type CreateRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules"`
}

type UpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
	Rules       []Rule  `json:"rules"`
}
