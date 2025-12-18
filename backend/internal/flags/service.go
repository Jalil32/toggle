package flag

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
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
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Create(f *Flag) error {
	if err := s.validateFlag(f); err != nil {
		return err
	}

	f.ID = uuid.New().String()

	if err := s.repo.Create(f); err != nil {
		return fmt.Errorf("failed to create flag: %w", err)
	}

	return nil
}

func (s *service) GetByID(id string) (*Flag, error) {
	if id == "" {
		return nil, ErrInvalidFlagData
	}

	flag, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFlagNotFound
		}
		return nil, fmt.Errorf("failed to get flag: %w", err)
	}

	return flag, nil
}

func (s *service) List() ([]Flag, error) {
	flags, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}

	if flags == nil {
		return []Flag{}, nil
	}

	return flags, nil
}

func (s *service) Update(f *Flag) error {
	if err := s.validateFlag(f); err != nil {
		return err
	}

	if f.ID == "" {
		return ErrInvalidFlagData
	}

	if err := s.repo.Update(f); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFlagNotFound
		}
		return fmt.Errorf("failed to update flag: %w", err)
	}

	return nil
}

func (s *service) Delete(id string) error {
	if id == "" {
		return ErrInvalidFlagData
	}

	if err := s.repo.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFlagNotFound
		}
		return fmt.Errorf("failed to delete flag: %w", err)
	}

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
