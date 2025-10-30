package storage

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	db  *sql.DB
	log *zap.Logger
}

// NewSQLite opens (or creates) the SQLite file at dbPath and runs the
// migration that creates the `metrics` table if it does not exist.
// The caller must call Close() when the program shuts down.
func NewSQLite(dbPath string, log *zap.Logger) (*SQLite, error) {
	// The modernc.org driver is pure‑go and works without CGO.
	// DSN format: file:<path>?cache=shared&_fk=1
	dsn := fmt.Sprintf("file:%s?_fk=1", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Verify the connection quickly.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	s := &SQLite{db: db, log: log}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migration: %w", err)
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	const stmt = `
CREATE TABLE IF NOT EXISTS metrics (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    ts        DATETIME NOT NULL,
    name      TEXT NOT NULL,
    value     REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_metrics_name_ts ON metrics(name, ts);
`
	_, err := s.db.Exec(stmt)
	if err != nil {
		return fmt.Errorf("create metrics table: %w", err)
	}
	s.log.Info("SQLite migration applied")
	return nil
}

// Save stores a snapshot in a single transaction.
func (s *SQLite) Save(ctx context.Context, snap *MetricsSnapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO metrics (ts, name, value) VALUES (?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	ts := snap.CollectedAt.UTC()
	for _, m := range snap.Metrics {
		if _, err := stmt.ExecContext(ctx, ts, m.Name, m.Value); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec insert for %s: %w", m.Name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	s.log.Debug("snapshot persisted", zap.Time("ts", ts), zap.Int("metrics", len(snap.Metrics)))
	return nil
}

// Close shuts down the database connection.
func (s *SQLite) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
