package repository

import (
	"context"

	db "world-of-wisdom/internal/database/generated"
)

type logRepo struct {
	queries *Queries
	db      db.DBTX
}

func (r *logRepo) Create(ctx context.Context, log CreateLogParams) (Log, error) {
	return r.queries.CreateLog(ctx, r.db, log)
}

func (r *logRepo) GetRecent(ctx context.Context, limit int32) ([]Log, error) {
	return r.queries.GetRecentLogs(ctx, r.db, limit)
}

func (r *logRepo) GetByLevel(ctx context.Context, params GetLogsByLevelParams) ([]Log, error) {
	return r.queries.GetLogsByLevel(ctx, r.db, params)
}

func (r *logRepo) GetPaginated(ctx context.Context, params GetLogsPaginatedParams) ([]Log, error) {
	return r.queries.GetLogsPaginated(ctx, r.db, params)
}