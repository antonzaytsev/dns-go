package aggregation

import (
	"context"
	"fmt"
	"time"

	"dns-go/internal/postgres"
)

// Scheduler manages periodic background jobs
type Scheduler struct {
	pgClient *postgres.Client
	stopChan chan struct{}
	doneChan chan struct{}
}

// NewScheduler creates a new scheduler instance
func NewScheduler(pgClient *postgres.Client) *Scheduler {
	return &Scheduler{
		pgClient: pgClient,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start starts the hourly aggregation job
func (s *Scheduler) Start() error {
	if s.pgClient == nil {
		return fmt.Errorf("PostgreSQL client not available")
	}

	// Run initial aggregation immediately
	go func() {
		if err := s.runAggregation(); err != nil {
			fmt.Printf("⚠️  Failed to run initial aggregation: %v\n", err)
		} else {
			fmt.Println("✅ Initial aggregation completed")
		}
	}()

	// Start hourly ticker
	ticker := time.NewTicker(1 * time.Hour)

	fmt.Println("🔄 Background aggregation scheduler started (runs hourly)")

	for {
		select {
		case <-ticker.C:
			if err := s.runAggregation(); err != nil {
				fmt.Printf("⚠️  Failed to run hourly aggregation: %v\n", err)
			} else {
				fmt.Printf("✅ Hourly aggregation completed at %s\n", time.Now().Format("2006-01-02 15:04:05"))
			}
		case <-s.stopChan:
			fmt.Println("🛑 Background aggregation scheduler stopping...")
			ticker.Stop()
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
