package seeding

import (
	"context"

	"expense-splitter/types"
)

type Seeder struct {
	db     *types.DBPool
	cfg    *types.Config
	logger *types.Logger
}

func New(db *types.DBPool, cfg *types.Config, logger *types.Logger) *Seeder {
	return &Seeder{db: db, cfg: cfg, logger: logger}
}

func (s *Seeder) Run(ctx context.Context) error {
	const q = `
INSERT INTO users (email, is_global_admin, verification_status)
VALUES ($1, true, 'verified')
ON CONFLICT (email) DO NOTHING`

	tag, err := s.db.Exec(ctx, q, s.cfg.GlobalAdminEmail)
	if err != nil {
		s.logger.Errorw("seed global admin", "error", err)
		return err
	}
	if tag.RowsAffected() == 1 {
		s.logger.Infow("seeded default global admin", "email", s.cfg.GlobalAdminEmail)
	} else {
		s.logger.Infow("default global admin already present", "email", s.cfg.GlobalAdminEmail)
	}
	return nil
}
