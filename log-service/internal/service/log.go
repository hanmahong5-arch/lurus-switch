package service

import (
	"context"
	"time"

	"github.com/pocketzworld/lurus-switch/log-service/internal/biz"
	"go.uber.org/zap"
)

// LogService is the log service implementation
type LogService struct {
	uc     *biz.LogUsecase
	logger *zap.Logger
}

// NewLogService creates a new log service
func NewLogService(uc *biz.LogUsecase, logger *zap.Logger) *LogService {
	return &LogService{
		uc:     uc,
		logger: logger,
	}
}

// WriteLog writes a single log entry
func (s *LogService) WriteLog(ctx context.Context, log *biz.RequestLog) error {
	return s.uc.WriteLog(ctx, log)
}

// WriteLogs writes multiple log entries
func (s *LogService) WriteLogs(ctx context.Context, logs []*biz.RequestLog) (int, int, error) {
	return s.uc.WriteLogs(ctx, logs)
}

// QueryLogs queries logs with filters
func (s *LogService) QueryLogs(ctx context.Context, filter *biz.LogFilter) ([]*biz.RequestLog, int64, error) {
	return s.uc.QueryLogs(ctx, filter)
}

// GetStats returns aggregated statistics
func (s *LogService) GetStats(ctx context.Context, userID, platform string, startTime, endTime time.Time) (*biz.LogStats, error) {
	filter := &biz.StatsFilter{
		UserID:    userID,
		Platform:  platform,
		StartTime: startTime,
		EndTime:   endTime,
	}
	return s.uc.GetStats(ctx, filter)
}

// GetHourlyStats returns hourly statistics
func (s *LogService) GetHourlyStats(ctx context.Context, userID, platform string, startTime, endTime time.Time) ([]*biz.HourlyStat, error) {
	filter := &biz.StatsFilter{
		UserID:    userID,
		Platform:  platform,
		StartTime: startTime,
		EndTime:   endTime,
	}
	return s.uc.GetHourlyStats(ctx, filter)
}

// GetDailyStats returns daily statistics
func (s *LogService) GetDailyStats(ctx context.Context, userID, platform string, startTime, endTime time.Time) ([]*biz.DailyStat, error) {
	filter := &biz.StatsFilter{
		UserID:    userID,
		Platform:  platform,
		StartTime: startTime,
		EndTime:   endTime,
	}
	return s.uc.GetDailyStats(ctx, filter)
}

// GetModelUsage returns per-model usage statistics
func (s *LogService) GetModelUsage(ctx context.Context, userID, platform string, startTime, endTime time.Time, limit int) ([]*biz.ModelUsageStat, error) {
	filter := &biz.StatsFilter{
		UserID:    userID,
		Platform:  platform,
		StartTime: startTime,
		EndTime:   endTime,
	}
	return s.uc.GetModelUsage(ctx, filter, limit)
}
