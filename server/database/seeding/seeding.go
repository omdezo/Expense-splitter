package seeding

import (
	"context"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

type Seeder struct {
	q      *repo.Queries
	cfg    *types.Config
	logger *types.Logger
}

func New(db *types.DBPool, cfg *types.Config, logger *types.Logger) *Seeder {
	return &Seeder{q: repo.New(db), cfg: cfg, logger: logger}
}

func (s *Seeder) Run(ctx context.Context) error {
	rows, err := s.q.SeedGlobalAdmin(ctx, s.cfg.GlobalAdminEmail)
	if err != nil {
		s.logger.Errorw("seed global admin", "error", err)
		return err
	}
	if rows == 1 {
		s.logger.Infow("seeded default global admin", "email", s.cfg.GlobalAdminEmail)
	} else {
		s.logger.Infow("default global admin already present", "email", s.cfg.GlobalAdminEmail)
	}
	return nil
}
