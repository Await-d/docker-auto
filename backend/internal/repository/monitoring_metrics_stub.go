package repository

import (
	"context"
	"docker-auto/internal/model"
	"docker-auto/pkg/docker"
	"time"

	"gorm.io/gorm"
)

// noopMonitoringMetricsRepository is a stub implementation for testing
type noopMonitoringMetricsRepository struct {
	db *gorm.DB
}

// NewMonitoringMetricsRepository creates a new monitoring metrics repository (stub)
func NewMonitoringMetricsRepository(db *gorm.DB) MonitoringMetricsRepository {
	return &noopMonitoringMetricsRepository{db: db}
}

func (r *noopMonitoringMetricsRepository) Create(ctx context.Context, metrics *model.MonitoringMetrics) error {
	return nil
}

func (r *noopMonitoringMetricsRepository) GetByID(ctx context.Context, id int64) (*model.MonitoringMetrics, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *noopMonitoringMetricsRepository) Update(ctx context.Context, metrics *model.MonitoringMetrics) error {
	return nil
}

func (r *noopMonitoringMetricsRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *noopMonitoringMetricsRepository) List(ctx context.Context, filter *model.MonitoringMetricsFilter) ([]*model.MonitoringMetrics, int64, error) {
	return []*model.MonitoringMetrics{}, 0, nil
}

func (r *noopMonitoringMetricsRepository) GetByContainerID(ctx context.Context, containerID string, limit, offset int) ([]*model.MonitoringMetrics, int64, error) {
	return []*model.MonitoringMetrics{}, 0, nil
}

func (r *noopMonitoringMetricsRepository) GetByTimeRange(ctx context.Context, containerID string, start, end time.Time) ([]*model.MonitoringMetrics, error) {
	return []*model.MonitoringMetrics{}, nil
}

func (r *noopMonitoringMetricsRepository) GetLatest(ctx context.Context, containerID string) (*model.MonitoringMetrics, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *noopMonitoringMetricsRepository) GetAverageMetrics(ctx context.Context, containerID string, start, end time.Time) (*model.MonitoringMetrics, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *noopMonitoringMetricsRepository) GetMaxMetrics(ctx context.Context, containerID string, start, end time.Time) (*model.MonitoringMetrics, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *noopMonitoringMetricsRepository) GetMetricsSummary(ctx context.Context, containerID string, duration time.Duration) (*docker.MetricsSummary, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *noopMonitoringMetricsRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	return 0, nil
}

func (r *noopMonitoringMetricsRepository) CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	return 0, nil
}

func (r *noopMonitoringMetricsRepository) CreateBatch(ctx context.Context, metrics []*model.MonitoringMetrics) error {
	return nil
}

func (r *noopMonitoringMetricsRepository) GetHighUsageContainers(ctx context.Context, cpuThreshold, memoryThreshold float64) ([]*model.MonitoringMetrics, error) {
	return []*model.MonitoringMetrics{}, nil
}
