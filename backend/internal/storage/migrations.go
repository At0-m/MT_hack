package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tramflow/migrations"
)

var migrationNames = []string{
	"001_initial.sql", "002_sessions_anchors.sql", "003_weather_geography.sql",
	"004_login_limits.sql", "005_calendar_flags.sql", "006_migration_checksums.sql",
}

type migrationQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func migrationSQL(index int) ([]byte, string, error) {
	sql, err := migrations.Files.ReadFile(migrationNames[index])
	sql = bytes.ReplaceAll(sql, []byte("\r\n"), []byte("\n"))
	return sql, fmt.Sprintf("%x", sha256.Sum256(sql)), err
}

// Validate an exact contiguous prefix, not just MAX(version).
func migrationState(ctx context.Context, query migrationQuery, allowLegacy bool) (int, error) {
	var exists, checksums bool
	if err := query.QueryRow(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	if err := query.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns
 WHERE table_schema='public' AND table_name='schema_migrations' AND column_name='checksum')`).Scan(&checksums); err != nil {
		return 0, err
	}
	sql := "SELECT version,NULL::text FROM schema_migrations ORDER BY version"
	if checksums {
		sql = "SELECT version,checksum FROM schema_migrations ORDER BY version"
	}
	rows, err := query.Query(ctx, sql)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var version int
		var digest *string
		if err := rows.Scan(&version, &digest); err != nil {
			return 0, err
		}
		if version != count+1 || version > len(migrationNames) {
			return 0, fmt.Errorf("migration versions have a gap or unsupported version %d", version)
		}
		_, expected, err := migrationSQL(count)
		if err != nil {
			return 0, err
		}
		if checksums {
			if digest == nil || *digest != expected {
				return 0, fmt.Errorf("migration checksum drift at version %d", version)
			}
		} else if !allowLegacy || version >= 6 {
			return 0, fmt.Errorf("migration checksums absent; apply checksum adoption migration")
		}
		count++
	}
	return count, rows.Err()
}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741852)"); err != nil {
		return err
	}
	count, err := migrationState(ctx, tx, true)
	if err != nil {
		return err
	}
	for index := count; index < len(migrationNames); index++ {
		sql, _, err := migrationSQL(index)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
	}
	// Old releases had no checksum evidence. Adoption records the current SQL;
	// it cannot prove which bytes were applied in the past. Future drift is refused.
	for index := range migrationNames {
		_, digest, err := migrationSQL(index)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "UPDATE schema_migrations SET checksum=$2 WHERE version=$1 AND checksum IS NULL", index+1, digest); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "ALTER TABLE schema_migrations ALTER COLUMN checksum SET NOT NULL"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CheckSchema(ctx context.Context) error {
	count, err := migrationState(ctx, s.Pool, false)
	if err != nil {
		return err
	}
	var extension string
	if err := s.Pool.QueryRow(ctx, "SELECT postgis_version()").Scan(&extension); err != nil {
		return err
	}
	if count != len(migrationNames) || extension == "" {
		return fmt.Errorf("schema not ready")
	}
	return nil
}
