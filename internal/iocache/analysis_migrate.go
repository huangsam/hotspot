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
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/huangsam/hotspot/schema"
)

// Target migration version constants.
const (
	targetLatestVersion  = -1
	targetInitialVersion = 0
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationBuilder handles the construction of migration components.
type MigrationBuilder struct {
	backend schema.CacheBackend
	connStr string
	db      *sql.DB
	driver  database.Driver
	source  source.Driver
	m       *migrate.Migrate
}

// NewMigrationBuilder creates a new MigrationBuilder instance.
func NewMigrationBuilder(backend schema.CacheBackend, connStr string) *MigrationBuilder {
	return &MigrationBuilder{
		backend: backend,
		connStr: connStr,
	}
}

// buildDatabase opens and verifies the database connection.
func (b *MigrationBuilder) buildDatabase() error {
	var err error
	var driverName string

	switch b.backend {
	case schema.SQLiteBackend:
		driverName = "sqlite3"
		dbPath := b.connStr
		if dbPath == "" {
			dbPath = GetAnalysisDBFilePath()
		}
		b.db, err = sql.Open(driverName, dbPath)
		if err != nil {
			return fmt.Errorf("failed to open SQLite database: %w", err)
		}
		b.db.SetMaxOpenConns(1)

	case schema.MySQLBackend:
		driverName = "mysql"
		b.db, err = sql.Open(driverName, b.connStr)
		if err != nil {
			return fmt.Errorf("failed to open MySQL database: %w", err)
		}

	case schema.PostgreSQLBackend:
		driverName = "pgx"
		b.db, err = sql.Open(driverName, b.connStr)
		if err != nil {
			return fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}

	default:
		return fmt.Errorf("unsupported backend: %s", b.backend)
	}

	// Verify connection
	if err := b.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// buildDriver creates the appropriate migrate driver instance.
func (b *MigrationBuilder) buildDriver() error {
	var err error
	switch b.backend {
	case schema.SQLiteBackend:
		b.driver, err = sqlite3.WithInstance(b.db, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("failed to create SQLite migrate driver: %w", err)
		}

	case schema.MySQLBackend:
		b.driver, err = mysql.WithInstance(b.db, &mysql.Config{})
		if err != nil {
			return fmt.Errorf("failed to create MySQL migrate driver: %w", err)
		}

	case schema.PostgreSQLBackend:
		b.driver, err = postgres.WithInstance(b.db, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL migrate driver: %w", err)
		}
	default:
		return fmt.Errorf("unsupported backend: %s", b.backend)
	}
	return nil
}

// buildSource sets up the migration source from embedded FS.
func (b *MigrationBuilder) buildSource() error {
	migrationFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to access migrations directory: %w", err)
	}

	b.source, err = iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}
	return nil
}

// buildMigrate creates the migrate instance.
func (b *MigrationBuilder) buildMigrate() error {
	var err error
	b.m, err = migrate.NewWithInstance("iofs", b.source, "hotspot", b.driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return nil
}

// executeMigration performs the migration based on the target version.
func executeMigration(m *migrate.Migrate, targetVersion int) error {
	// Get current version
	currentVersion, dirty, err := m.Version()
	// Track if this is a new database with no migrations applied yet
	isNewDatabase := err == migrate.ErrNilVersion
	if err != nil && !isNewDatabase {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in a dirty state at version %d. Please fix manually or force version", currentVersion)
	}

	// Perform migration
	switch {
	case targetVersion == targetLatestVersion:
		// Migrate to latest version
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to migrate to latest version: %w", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migration needed. Database is already at the latest version")
		} else {
			newVersion, _, _ := m.Version()
			if isNewDatabase {
				fmt.Printf("Successfully migrated new database to version %d.\n", newVersion)
			} else {
				fmt.Printf("Successfully migrated from version %d to version %d.\n", currentVersion, newVersion)
			}
		}
	case targetVersion == targetInitialVersion:
		// Special case: migrate all the way down to version 0 (no migrations applied)
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to roll back to version 0: %w", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migration needed. Database is already at version 0.")
		} else {
			fmt.Printf("Successfully rolled back from version %d to version 0.\n", currentVersion)
		}
	case targetVersion > targetInitialVersion:
		// Migrate to specific version
		err = m.Migrate(uint(targetVersion))
		if err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to migrate to version %d: %w", targetVersion, err)
		}
		if err == migrate.ErrNoChange {
			fmt.Printf("No migration needed. Database is already at version %d.\n", targetVersion)
		} else {
			if isNewDatabase {
				fmt.Printf("Successfully migrated new database to version %d.\n", targetVersion)
			} else {
				fmt.Printf("Successfully migrated from version %d to version %d.\n", currentVersion, targetVersion)
			}
		}
	default:
		return fmt.Errorf("invalid target version: %d", targetVersion)
	}

	return nil
}

// MigrateAnalysis runs database migrations for the analysis store.
// If targetVersion == -1, it migrates to the latest version.
// If targetVersion == 0, it rolls back all migrations (to initial state).
// If targetVersion > 0, it migrates to the specified version.
// If targetVersion is invalid, it returns an error.
func MigrateAnalysis(backend schema.CacheBackend, connStr string, targetVersion int) error {
	if backend == schema.NoneBackend {
		return fmt.Errorf("migrations are not supported for NoneBackend")
	}

	builder := NewMigrationBuilder(backend, connStr)
	if err := builder.buildDatabase(); err != nil {
		return err
	}
	defer func() { _ = builder.db.Close() }()

	if err := builder.buildDriver(); err != nil {
		return err
	}

	if err := builder.buildSource(); err != nil {
		return err
	}

	if err := builder.buildMigrate(); err != nil {
		return err
	}
	defer func() { _, _ = builder.m.Close() }()

	return executeMigration(builder.m, targetVersion)
}
