package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "world-of-wisdom/internal/database/generated"
)

type challengeRepo struct {
	queries *Queries
	db      db.DBTX
}

func (r *challengeRepo) Create(ctx context.Context, challenge CreateChallengeParams) (Challenge, error) {
	return r.queries.CreateChallenge(ctx, r.db, challenge)
}

func (r *challengeRepo) GetByID(ctx context.Context, id uuid.UUID) (Challenge, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	return r.queries.GetChallenge(ctx, r.db, pgID)
}

func (r *challengeRepo) GetByClientID(ctx context.Context, clientID string) (Challenge, error) {
	return r.queries.GetChallengeByClientID(ctx, r.db, clientID)
}

func (r *challengeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ChallengeStatus) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	params := UpdateChallengeStatusParams{
		ID:     pgID,
		Status: status,
	}
	_, err := r.queries.UpdateChallengeStatus(ctx, r.db, params)
	return err
}

func (r *challengeRepo) GetFiltered(ctx context.Context, params GetChallengesFilteredParams) ([]GetChallengesFilteredRow, error) {
	return r.queries.GetChallengesFiltered(ctx, r.db, params)
}

func (r *challengeRepo) GetRecent(ctx context.Context, limit int32) ([]Challenge, error) {
	return r.queries.GetRecentChallenges(ctx, r.db, limit)
}

func (r *challengeRepo) GetStats(ctx context.Context) (GetChallengeStatsRow, error) {
	return r.queries.GetChallengeStats(ctx, r.db)
}