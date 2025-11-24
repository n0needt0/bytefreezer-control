package storage

import (
	"context"

	"github.com/n0needt0/go-goodies/log"
)

// CleanupOldServiceOperations removes completed/failed operations older than 7 days
// Returns the number of rows deleted
func (s *PostgreSQLStorage) CleanupOldServiceOperations(ctx context.Context) (int64, error) {
	query := `SELECT cleanup_old_service_operations()`

	var deletedCount int64
	err := s.db.QueryRowContext(ctx, query).Scan(&deletedCount)
	if err != nil {
		log.Errorf("Failed to cleanup old service operations: %v", err)
		return 0, err
	}

	if deletedCount > 0 {
		log.Debugf("Cleaned up %d old service operations (7-day retention)", deletedCount)
	}

	return deletedCount, nil
}

// CleanupOldServiceMetrics removes metrics rollup data older than 30 days
// Returns the number of rows deleted
func (s *PostgreSQLStorage) CleanupOldServiceMetrics(ctx context.Context) (int64, error) {
	query := `SELECT cleanup_old_service_metrics()`

	var deletedCount int64
	err := s.db.QueryRowContext(ctx, query).Scan(&deletedCount)
	if err != nil {
		log.Errorf("Failed to cleanup old service metrics: %v", err)
		return 0, err
	}

	if deletedCount > 0 {
		log.Debugf("Cleaned up %d old service metrics records (30-day retention)", deletedCount)
	}

	return deletedCount, nil
}
