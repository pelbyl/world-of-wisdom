package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "world-of-wisdom/internal/database/generated"
)

type solutionRepo struct {
	queries *Queries
	db      db.DBTX
}

func (r *solutionRepo) Create(ctx context.Context, solution CreateSolutionParams) (Solution, error) {
	return r.queries.CreateSolution(ctx, r.db, solution)
}

func (r *solutionRepo) GetByID(ctx context.Context, id uuid.UUID) (Solution, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	return r.queries.GetSolution(ctx, r.db, pgID)
}

func (r *solutionRepo) GetByChallenge(ctx context.Context, challengeID uuid.UUID) ([]Solution, error) {
	pgID := pgtype.UUID{Bytes: challengeID, Valid: true}
	return r.queries.GetSolutionsByChallenge(ctx, r.db, pgID)
}

func (r *solutionRepo) GetRecent(ctx context.Context, limit int32) ([]GetRecentSolutionsRow, error) {
	return r.queries.GetRecentSolutions(ctx, r.db, limit)
}