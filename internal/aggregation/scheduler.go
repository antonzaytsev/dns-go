package aggregation

import (
	"context"
	"fmt"
	"time"

	"dns-go/internal/postgres"
)

const (
	// DefaultRetentionDays is the default number of days to keep DNS logs
	DefaultRetentionDays = 30
	// CleanupInterval is how often to run the cleanup job
	CleanupInterval = 24 * time.Hour
)

// Scheduler manages periodic background jobs
type Scheduler struct {
	pgClient      *postgres.Client
	retentionDays int
	stopChan      chan struct{}
	doneChan      chan struct{}
}

// NewScheduler creates a new scheduler instance
func NewScheduler(pgClient *postgres.Client) *Scheduler {
	return &Scheduler{
		pgClient:      pgClient,
		retentionDays: DefaultRetentionDays,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}
}

// SetRetentionDays sets the log retention period in days
func (s *Scheduler) SetRetentionDays(days int) {
	if days > 0 {
		s.retentionDays = days
	}
}

// Start starts the background jobs (aggregation hourly, cleanup daily)
func (s *Scheduler) Start() error {
	if s.pgClient == nil {
		return fmt.Errorf("PostgreSQL client not available")
	}

	// Run initial jobs immediately
	go func() {
		if err := s.runAggregation(); err != nil {
			fmt.Printf("⚠️  Failed to run initial aggregation: %v\n", err)
		} else {
			fmt.Println("✅ Initial aggregation completed")
		}

		// Run initial cleanup
		if err := s.runCleanup(); err != nil {
			fmt.Printf("⚠️  Failed to run initial cleanup: %v\n", err)
		}
	}()

	// Start tickers
	aggregationTicker := time.NewTicker(1 * time.Hour)
	cleanupTicker := time.NewTicker(CleanupInterval)

	fmt.Println("🔄 Background scheduler started:")
	fmt.Println("   - Aggregation: runs hourly")
	fmt.Printf("   - Log cleanup: runs daily (retention: %d days)\n", s.retentionDays)

	for {
		select {
		case <-aggregationTicker.C:
			if err := s.runAggregation(); err != nil {
				fmt.Printf("⚠️  Failed to run hourly aggregation: %v\n", err)
			} else {
				fmt.Printf("✅ Hourly aggregation completed at %s\n", time.Now().Format("2006-01-02 15:04:05"))
			}
		case <-cleanupTicker.C:
			if err := s.runCleanup(); err != nil {
				fmt.Printf("⚠️  Failed to run daily cleanup: %v\n", err)
			}
		case <-s.stopChan:
			fmt.Println("🛑 Background scheduler stopping...")
			aggregationTicker.Stop()
			cleanupTicker.Stop()
			close(s.doneChan)
			return nil
		}
	}
}

// runAggregation executes the aggregation and stores the results
func (s *Scheduler) runAggregation() error {
	start := time.Now()
	if err := s.pgClient.CalculateAndStoreAggregatedStats(); err != nil {
		return fmt.Errorf("aggregation failed: %w", err)
	}
	duration := time.Since(start)
	fmt.Printf("📊 Aggregation completed in %v\n", duration)
	return nil
}

// runCleanup deletes old logs based on the retention policy
func (s *Scheduler) runCleanup() error {
	start := time.Now()
	deletedCount, err := s.pgClient.DeleteOldLogs(s.retentionDays)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	duration := time.Since(start)

	if deletedCount > 0 {
		fmt.Printf("🧹 Cleanup completed: deleted %d logs older than %d days (took %v)\n",
			deletedCount, s.retentionDays, duration)
	} else {
		fmt.Printf("🧹 Cleanup completed: no logs older than %d days to delete\n", s.retentionDays)
	}
	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop(ctx context.Context) error {
	close(s.stopChan)

	// Wait for current job to finish or timeout
	select {
	case <-s.doneChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
