package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "world-of-wisdom/internal/database/generated"
)

type connectionRepo struct {
	queries *Queries
	db      db.DBTX
}

func (r *connectionRepo) Create(ctx context.Context, conn CreateConnectionParams) (Connection, error) {
	return r.queries.CreateConnection(ctx, r.db, conn)
}

func (r *connectionRepo) GetByID(ctx context.Context, id uuid.UUID) (Connection, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	return r.queries.GetConnection(ctx, r.db, pgID)
}

func (r *connectionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ConnectionStatus) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	params := UpdateConnectionStatusParams{
		ID:     pgID,
		Status: status,
	}
	_, err := r.queries.UpdateConnectionStatus(ctx, r.db, params)
	return err
}

func (r *connectionRepo) GetActive(ctx context.Context) ([]Connection, error) {
	return r.queries.GetActiveConnections(ctx, r.db)
}

func (r *connectionRepo) GetFiltered(ctx context.Context, status ConnectionStatus) ([]Connection, error) {
	return r.queries.GetConnectionsFiltered(ctx, r.db, status)
}

func (r *connectionRepo) GetStats(ctx context.Context) (GetConnectionStatsRow, error) {
	return r.queries.GetConnectionStats(ctx, r.db)
}