package repository

import (
	"context"

	db "world-of-wisdom/internal/database/generated"
)

type metricsRepo struct {
	queries *Queries
	db      db.DBTX
}

func (r *metricsRepo) Record(ctx context.Context, metric RecordMetricParams) error {
	return r.queries.RecordMetric(ctx, r.db, metric)
}

func (r *metricsRepo) GetSystem(ctx context.Context) ([]GetSystemMetricsRow, error) {
	return r.queries.GetSystemMetrics(ctx, r.db)
}

func (r *metricsRepo) GetByTimeRange(ctx context.Context, params GetMetricsByTimeRangeParams) ([]GetMetricsByTimeRangeRow, error) {
	return r.queries.GetMetricsByTimeRange(ctx, r.db, params)
}

func (r *metricsRepo) GetAggregated(ctx context.Context, params GetAggregatedMetricsParams) ([]GetAggregatedMetricsRow, error) {
	return r.queries.GetAggregatedMetrics(ctx, r.db, params)
}