package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jalil32/toggle/internal/organizations"
)

type Service struct {
	repo    Repository
	orgRepo organizations.Repository
}

func NewService(repo Repository, orgRepo organizations.Repository) *Service {
	return &Service{repo: repo, orgRepo: orgRepo}
}

func (s *Service) GetByAuth0ID(ctx context.Context, auth0ID string) (*User, error) {
	return s.repo.GetByAuth0ID(ctx, auth0ID)
}

func (s *Service) GetOrCreate(ctx context.Context, auth0ID, email, name string) (*User, error) {
	user, err := s.repo.GetByAuth0ID(ctx, auth0ID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// First time user - create org and user
	org, err := s.orgRepo.Create(ctx, name+"'s Organization")
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, auth0ID, org.ID, email, name, "owner")
}
