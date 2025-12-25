package evaluation

import (
	"context"
	"log/slog"

	flag "github.com/jalil32/toggle/internal/flags"
	appContext "github.com/jalil32/toggle/internal/pkg/context"
)

type Service interface {
	EvaluateAll(ctx context.Context, projectID string, evalCtx EvaluationContext) (*EvaluationResponse, error)
	EvaluateSingle(ctx context.Context, flagID string, tenantID string, evalCtx EvaluationContext) (*SingleEvaluationResponse, error)
}

type service struct {
	flagRepo  flag.Repository
	evaluator *Evaluator
	logger    *slog.Logger
}

func NewService(flagRepo flag.Repository, logger *slog.Logger) Service {
	return &service{
		flagRepo:  flagRepo,
		evaluator: NewEvaluator(),
		logger:    logger,
	}
}

// EvaluateAll evaluates all flags for a project
func (s *service) EvaluateAll(ctx context.Context, projectID string, evalCtx EvaluationContext) (*EvaluationResponse, error) {
	// Extract tenant ID from context (injected by API key middleware)
	tenantID := appContext.MustTenantID(ctx)

	// Fetch all flags for this project
	flags, err := s.flagRepo.ListByProject(ctx, projectID, tenantID)
	if err != nil {
		s.logger.Error("failed to fetch flags for evaluation",
			slog.String("project_id", projectID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Evaluate each flag
	results := make(map[string]bool)
	for _, f := range flags {
		enabled := s.evaluator.Evaluate(&f, evalCtx)
		results[f.ID] = enabled

		s.logger.Debug("flag evaluated",
			slog.String("flag_id", f.ID),
			slog.String("flag_name", f.Name),
			slog.Bool("enabled", enabled),
			slog.String("user_id", evalCtx.UserID),
		)
	}

	s.logger.Info("bulk evaluation completed",
		slog.String("project_id", projectID),
		slog.String("user_id", evalCtx.UserID),
		slog.Int("flags_evaluated", len(results)),
	)

	return &EvaluationResponse{Flags: results}, nil
}

// EvaluateSingle evaluates a single flag
func (s *service) EvaluateSingle(ctx context.Context, flagID string, tenantID string, evalCtx EvaluationContext) (*SingleEvaluationResponse, error) {
	// Fetch flag
	f, err := s.flagRepo.GetByID(ctx, flagID, tenantID)
	if err != nil {
		s.logger.Error("failed to fetch flag for evaluation",
			slog.String("flag_id", flagID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Evaluate
	enabled := s.evaluator.Evaluate(f, evalCtx)

	s.logger.Info("flag evaluated",
		slog.String("flag_id", flagID),
		slog.String("flag_name", f.Name),
		slog.Bool("enabled", enabled),
		slog.String("user_id", evalCtx.UserID),
	)

	return &SingleEvaluationResponse{
		Enabled: enabled,
		FlagID:  flagID,
	}, nil
}
