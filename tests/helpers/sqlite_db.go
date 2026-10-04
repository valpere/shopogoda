package helpers

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
)

// NewSQLiteDB opens a real, migrated SQLite database in a per-test temp dir
// and closes it when the test finishes.
func NewSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Connect(&config.DatabaseConfig{Path: filepath.Join(t.TempDir(), "test.db")})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
