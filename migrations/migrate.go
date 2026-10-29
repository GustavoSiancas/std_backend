package migrations

import (
	"embed"
	"fmt"
	"gorm.io/gorm"
)

//go:embed *.sql
var files embed.FS

// Apply serializes schema changes and records only fully committed migrations.
func Apply(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(718234510)").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; err != nil {
			return err
		}
		entries, err := files.ReadDir(".")
		if err != nil {
			return err
		}
		for _, entry := range entries {
			var count int64
			if err := tx.Table("schema_migrations").Where("version = ?", entry.Name()).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			sql, err := files.ReadFile(entry.Name())
			if err != nil {
				return err
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("migration %s: %w", entry.Name(), err)
			}
			if err := tx.Exec("INSERT INTO schema_migrations(version) VALUES (?)", entry.Name()).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
