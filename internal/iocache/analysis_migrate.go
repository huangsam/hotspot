package iocache

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/huangsam/hotspot/schema"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateAnalysis runs database migrations for the analysis store.
// - If targetVersion < 0, it migrates to the latest version.
// - If targetVersion == 0, it rolls back all migrations (to initial state).
// - If targetVersion > 0, it migrates to the specified version.
func MigrateAnalysis(backend schema.CacheBackend, connStr string, targetVersion int) error {
	if backend == schema.NoneBackend {
		return fmt.Errorf("migrations are not supported for NoneBackend")
	}

	// Open database connection
	var db *sql.DB
	var err error
	var driverName string

	switch backend {
	case schema.SQLiteBackend:
		driverName = "sqlite3"
		dbPath := connStr
		if dbPath == "" {
			dbPath = GetAnalysisDBFilePath()
		}
		db, err = sql.Open(driverName, dbPath)
		if err != nil {
			return fmt.Errorf("failed to open SQLite database: %w", err)
		}
		defer func() { _ = db.Close() }()

	case schema.MySQLBackend:
		driverName = "mysql"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return fmt.Errorf("failed to open MySQL database: %w", err)
		}
		defer func() { _ = db.Close() }()

	case schema.PostgreSQLBackend:
		driverName = "pgx"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}
		defer func() { _ = db.Close() }()

	default:
		return fmt.Errorf("unsupported backend: %s", backend)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create a migrate driver instance
	var driver database.Driver
	switch backend {
	case schema.SQLiteBackend:
		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("failed to create SQLite migrate driver: %w", err)
		}

	case schema.MySQLBackend:
		driver, err = mysql.WithInstance(db, &mysql.Config{})
		if err != nil {
			return fmt.Errorf("failed to create MySQL migrate driver: %w", err)
		}

	case schema.PostgreSQLBackend:
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL migrate driver: %w", err)
		}
	}

	// Get the migrations subdirectory
	migrationFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to access migrations directory: %w", err)
	}

	// Create source driver from embedded FS
	sourceDriver, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "hotspot", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Get current version
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in a dirty state at version %d. Please fix manually or force version", currentVersion)
	}

	// Perform migration
	if targetVersion < 0 {
		// Migrate to latest version
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to migrate to latest version: %w", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migration needed. Database is already at the latest version.")
		} else {
			newVersion, _, _ := m.Version()
			fmt.Printf("Successfully migrated from version %d to version %d\n", currentVersion, newVersion)
		}
	} else if targetVersion == 0 {
		// Special case: migrate all the way down to version 0 (no migrations applied)
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to roll back to version 0: %w", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migration needed. Database is already at version 0")
		} else {
			fmt.Printf("Successfully rolled back from version %d to version 0\n", currentVersion)
		}
	} else {
		// Migrate to specific version
		err = m.Migrate(uint(targetVersion))
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to migrate to version %d: %w", targetVersion, err)
		}
		if err == migrate.ErrNoChange {
			fmt.Printf("No migration needed. Database is already at version %d\n", targetVersion)
		} else {
			fmt.Printf("Successfully migrated from version %d to version %d\n", currentVersion, targetVersion)
		}
	}

	return nil
}
