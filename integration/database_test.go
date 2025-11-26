//go:build database

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestHotspotWithMySQL tests the hotspot CLI with a MySQL backend.
func TestHotspotWithMySQL(t *testing.T) {
	ctx := context.Background()

	// Start MySQL container
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8",
		ExposedPorts: []string{"3306:3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "secret123",
			"MYSQL_DATABASE":      "hotspot",
		},
		WaitingFor: wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(30 * time.Second),
	}
	mysqlC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() { _ = mysqlC.Terminate(ctx) }()

	// Get connection details
	host, err := mysqlC.Host(ctx)
	require.NoError(t, err)
	port, err := mysqlC.MappedPort(ctx, "3306")
	require.NoError(t, err)

	connStr := fmt.Sprintf("root:secret123@tcp(%s:%s)/hotspot?parseTime=true", host, port.Port())

	// Set environment variables
	_ = os.Setenv("HOTSPOT_CACHE_BACKEND", "mysql")
	_ = os.Setenv("HOTSPOT_CACHE_DB_CONNECT", connStr)
	_ = os.Setenv("HOTSPOT_ANALYSIS_BACKEND", "mysql")
	_ = os.Setenv("HOTSPOT_ANALYSIS_DB_CONNECT", connStr)
	defer func() { _ = os.Unsetenv("HOTSPOT_CACHE_BACKEND") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_CACHE_DB_CONNECT") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_ANALYSIS_BACKEND") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_ANALYSIS_DB_CONNECT") }()

	// Run hotspot cache clear
	err = runHotspotCommand(t, "cache", "clear")
	require.NoError(t, err)

	// Run hotspot analysis clear
	err = runHotspotCommand(t, "analysis", "clear")
	require.NoError(t, err)

	// Run hotspot files (on current dir)
	err = runHotspotCommand(t, "files", "--limit", "5")
	require.NoError(t, err)

	// Run hotspot cache status
	err = runHotspotCommand(t, "cache", "status")
	require.NoError(t, err)

	// Run hotspot analysis status
	err = runHotspotCommand(t, "analysis", "status")
	require.NoError(t, err)
}

// TestHotspotWithPostgres tests the hotspot CLI with a PostgreSQL backend.
func TestHotspotWithPostgres(t *testing.T) {
	ctx := context.Background()

	// Start Postgres container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432:5432/tcp"},
		Env: map[string]string{
			"POSTGRES_HOST_AUTH_METHOD": "trust",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(30 * time.Second),
	}
	pgC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() { _ = pgC.Terminate(ctx) }()
	time.Sleep(5 * time.Second)

	// Get connection details
	host, err := pgC.Host(ctx)
	require.NoError(t, err)
	port, err := pgC.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf("host=%s port=%s user=postgres dbname=postgres", host, port.Port())

	// Set environment variables
	_ = os.Setenv("HOTSPOT_CACHE_BACKEND", "postgresql")
	_ = os.Setenv("HOTSPOT_CACHE_DB_CONNECT", connStr)
	_ = os.Setenv("HOTSPOT_ANALYSIS_BACKEND", "postgresql")
	_ = os.Setenv("HOTSPOT_ANALYSIS_DB_CONNECT", connStr)
	defer func() { _ = os.Unsetenv("HOTSPOT_CACHE_BACKEND") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_CACHE_DB_CONNECT") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_ANALYSIS_BACKEND") }()
	defer func() { _ = os.Unsetenv("HOTSPOT_ANALYSIS_DB_CONNECT") }()

	// Run hotspot cache clear
	err = runHotspotCommand(t, "cache", "clear")
	require.NoError(t, err)

	// Run hotspot analysis clear
	err = runHotspotCommand(t, "analysis", "clear")
	require.NoError(t, err)

	// Run hotspot files (on current dir)
	err = runHotspotCommand(t, "files", "--limit", "5")
	require.NoError(t, err)

	// Run hotspot cache status
	err = runHotspotCommand(t, "cache", "status")
	require.NoError(t, err)

	// Run hotspot analysis status
	err = runHotspotCommand(t, "analysis", "status")
	require.NoError(t, err)
}

func runHotspotCommand(t *testing.T, args ...string) error {
	hotspotPath := getHotspotBinary()
	cmd := exec.Command(hotspotPath, args...)
	cmd.Dir = "../" // Run from project root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command failed: %s\nOutput: %s", cmd.String(), string(output))
		return err
	}
	return nil
}
