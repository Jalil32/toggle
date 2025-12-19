package projects

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, orgID, name string) (*Project, error) {
	return s.repo.Create(ctx, orgID, name)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByOrgID(ctx context.Context, orgID string) ([]Project, error) {
	return s.repo.ListByOrgID(ctx, orgID)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
