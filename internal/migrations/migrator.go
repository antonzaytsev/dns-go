package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed *.sql
var migrationsFS embed.FS

// Migration file format: YYYYMMDDHHMMSS__description.sql
// Example: 20241101000000__create_dns_logs_table.sql

const migrationsTableName = "schema_migrations"

// MigrationRecord tracks which migrations have been applied
type MigrationRecord struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Version     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Description string    `gorm:"type:varchar(500)"`
	AppliedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
}

func (MigrationRecord) TableName() string {
	return migrationsTableName
}

// Migrator handles database migrations
type Migrator struct {
	db *gorm.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{db: db}
}

// Run checks for pending migrations and executes them
func (m *Migrator) Run(ctx context.Context) error {
	fmt.Println("🔍 Checking for pending database migrations...")

	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migration files
	migrationFiles, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	if len(migrationFiles) == 0 {
		fmt.Println("📝 No migration files found")
		return nil
	}

	// Get applied migrations
	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Identify pending migrations
	var pendingMigrations []string
	for _, file := range migrationFiles {
		version := extractVersion(file)
		if !appliedVersions[version] {
			pendingMigrations = append(pendingMigrations, file)
		}
	}

	// Report status
	totalMigrations := len(migrationFiles)
	appliedCount := len(appliedVersions)
	pendingCount := len(pendingMigrations)

	fmt.Printf("📊 Migration status: %d total, %d applied, %d pending\n", totalMigrations, appliedCount, pendingCount)

	if pendingCount == 0 {
		fmt.Println("✅ All migrations are up to date")
		return nil
	}

	// Execute pending migrations
	fmt.Printf("🔄 Running %d pending migration(s)...\n", pendingCount)
	for _, file := range pendingMigrations {
		version := extractVersion(file)
		if err := m.executeMigration(ctx, file, version); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}
	}

	fmt.Printf("✅ All %d pending migration(s) completed successfully\n", pendingCount)
	return nil
}

// ensureMigrationsTable creates the migrations tracking table if it doesn't exist
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	return m.db.WithContext(ctx).AutoMigrate(&MigrationRecord{})
}

// getMigrationFiles returns all migration files sorted by version
func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match files with datetime prefix: YYYYMMDDHHMMSS__description.sql
		if strings.HasSuffix(name, ".sql") && len(name) >= 14 && strings.Contains(name, "__") {
			// Check if it starts with 14 digits (datetime format)
			if isDatetimePrefix(name[:14]) {
				files = append(files, name)
			}
		}
	}

	// Sort by datetime prefix (already in chronological order)
	sort.Strings(files)

	return files, nil
}

// isDatetimePrefix checks if a string is a valid datetime prefix (14 digits: YYYYMMDDHHMMSS)
func isDatetimePrefix(s string) bool {
	if len(s) != 14 {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// extractVersion extracts the datetime version from a migration filename
// Format: YYYYMMDDHHMMSS__{description}.sql
func extractVersion(filename string) string {
	parts := strings.Split(filename, "__")
	if len(parts) > 0 {
		// Return the datetime prefix (first 14 characters)
		version := parts[0]
		if len(version) >= 14 {
			return version[:14]
		}
		return version
	}
	return filename
}

// getAppliedVersions returns a map of applied migration versions
func (m *Migrator) getAppliedVersions(ctx context.Context) (map[string]bool, error) {
	var records []MigrationRecord
	if err := m.db.WithContext(ctx).Find(&records).Error; err != nil {
		return nil, err
	}

	applied := make(map[string]bool)
	for _, record := range records {
		applied[record.Version] = true
	}

	return applied, nil
}

// executeMigration executes a single migration file
func (m *Migrator) executeMigration(ctx context.Context, filename, version string) error {
	// Read migration file
	sql, err := migrationsFS.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Extract description from filename
	description := extractDescription(filename)

	// Execute migration in a transaction
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Execute SQL
		if err := tx.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}

		// Record migration
		record := MigrationRecord{
			Version:     version,
			Description: description,
			AppliedAt:   time.Now(),
		}

		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		fmt.Printf("✅ Applied migration %s: %s\n", version, description)
		return nil
	})
}

// extractDescription extracts the description from a migration filename
// Format: YYYYMMDDHHMMSS__{description}.sql
func extractDescription(filename string) string {
	parts := strings.Split(filename, "__")
	if len(parts) > 1 {
		desc := strings.TrimSuffix(parts[1], ".sql")
		return strings.ReplaceAll(desc, "_", " ")
	}
	// If no description found, return filename without extension and datetime prefix
	if strings.HasSuffix(filename, ".sql") {
		name := strings.TrimSuffix(filename, ".sql")
		if len(name) > 15 && strings.Contains(name, "__") {
			parts := strings.Split(name, "__")
			if len(parts) > 1 {
				return strings.ReplaceAll(parts[1], "_", " ")
			}
		}
		return name
	}
	return filename
}
