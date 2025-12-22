package flag

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

var (
	ErrFlagNotFound    = errors.New("flag not found")
	ErrInvalidFlagData = errors.New("invalid flag data")
)

type Service interface {
	Create(f *Flag) error
	GetByID(id string) (*Flag, error)
	List() ([]Flag, error)
	Update(f *Flag) error
	Delete(id string) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) Create(f *Flag) error {
	if err := s.validateFlag(f); err != nil {
		s.logger.Warn("flag validation failed",
			slog.String("name", f.Name),
			slog.String("error", err.Error()),
		)
		return err
	}

	if err := s.repo.Create(f); err != nil {
		s.logger.Error("failed to create flag",
			slog.String("name", f.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create flag: %w", err)
	}

	s.logger.Info("flag created",
		slog.String("id", f.ID),
		slog.String("name", f.Name),
		slog.String("project_id", f.ProjectID),
	)

	return nil
}

func (s *service) GetByID(id string) (*Flag, error) {
	if id == "" {
		return nil, ErrInvalidFlagData
	}

	flag, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found", slog.String("id", id))
			return nil, ErrFlagNotFound
		}
		s.logger.Error("failed to get flag",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get flag: %w", err)
	}

	return flag, nil
}

func (s *service) List() ([]Flag, error) {
	flags, err := s.repo.List()
	if err != nil {
		s.logger.Error("failed to list flags", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}

	if flags == nil {
		return []Flag{}, nil
	}

	return flags, nil
}

func (s *service) Update(f *Flag) error {
	if err := s.validateFlag(f); err != nil {
		s.logger.Warn("flag validation failed on update",
			slog.String("id", f.ID),
			slog.String("error", err.Error()),
		)
		return err
	}

	if f.ID == "" {
		return ErrInvalidFlagData
	}

	if err := s.repo.Update(f); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found on update", slog.String("id", f.ID))
			return ErrFlagNotFound
		}
		s.logger.Error("failed to update flag",
			slog.String("id", f.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update flag: %w", err)
	}

	s.logger.Info("flag updated",
		slog.String("id", f.ID),
		slog.String("name", f.Name),
	)

	return nil
}

func (s *service) Delete(id string) error {
	if id == "" {
		return ErrInvalidFlagData
	}

	if err := s.repo.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("flag not found on delete", slog.String("id", id))
			return ErrFlagNotFound
		}
		s.logger.Error("failed to delete flag",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	s.logger.Info("flag deleted", slog.String("id", id))

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
