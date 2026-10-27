package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		if os.IsNotExist(err) {
			entries, err = os.ReadDir(filepath.Join("backend", "migrations"))
		}
		if err != nil {
			return fmt.Errorf("read database migrations: %w", err)
		}
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(74182026)`); err != nil {
		return fmt.Errorf("lock database migrations: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(74182026)`) }()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}

	var hasBaseSchema bool
	if err := conn.QueryRow(ctx, `SELECT to_regclass('public.pa_requests') IS NOT NULL`).Scan(&hasBaseSchema); err != nil {
		return fmt.Errorf("check base schema: %w", err)
	}
	var recorded int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&recorded); err != nil {
		return fmt.Errorf("read migration history: %w", err)
	}
	if hasBaseSchema && recorded == 0 {
		if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ('001_init.sql') ON CONFLICT DO NOTHING`); err != nil {
			return fmt.Errorf("record existing base schema: %w", err)
		}
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		var exists bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if exists {
			continue
		}
		script, err := os.ReadFile(filepath.Join("migrations", name))
		if err != nil && os.IsNotExist(err) {
			script, err = os.ReadFile(filepath.Join("backend", "migrations", name))
		}
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := conn.Conn().PgConn().Exec(ctx, string(script)).ReadAll(); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}
